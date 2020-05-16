package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger2"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	logging "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/textileio/powergate/deals"
	dealsPb "github.com/textileio/powergate/deals/pb"
	"github.com/textileio/powergate/fchost"
	"github.com/textileio/powergate/ffs"
	"github.com/textileio/powergate/ffs/cidlogger"
	"github.com/textileio/powergate/ffs/coreipfs"
	"github.com/textileio/powergate/ffs/filcold"
	"github.com/textileio/powergate/ffs/filcold/lotuschain"
	"github.com/textileio/powergate/ffs/manager"
	"github.com/textileio/powergate/ffs/minerselector/reptop"
	ffsGrpc "github.com/textileio/powergate/ffs/rpc"
	ffsRpc "github.com/textileio/powergate/ffs/rpc"
	"github.com/textileio/powergate/ffs/scheduler"
	"github.com/textileio/powergate/ffs/scheduler/astore"
	"github.com/textileio/powergate/ffs/scheduler/cistore"
	"github.com/textileio/powergate/ffs/scheduler/jstore"
	"github.com/textileio/powergate/gateway"
	"github.com/textileio/powergate/health"
	healthRpc "github.com/textileio/powergate/health/rpc"
	askPb "github.com/textileio/powergate/index/ask/pb"
	askRpc "github.com/textileio/powergate/index/ask/rpc"
	ask "github.com/textileio/powergate/index/ask/runner"
	"github.com/textileio/powergate/index/miner"
	minerPb "github.com/textileio/powergate/index/miner/pb"
	"github.com/textileio/powergate/index/slashing"
	slashingPb "github.com/textileio/powergate/index/slashing/pb"
	"github.com/textileio/powergate/iplocation/ip2location"
	"github.com/textileio/powergate/lotus"
	pgnet "github.com/textileio/powergate/net"
	pgnetlotus "github.com/textileio/powergate/net/lotus"
	pgnetRpc "github.com/textileio/powergate/net/rpc"
	"github.com/textileio/powergate/reputation"
	reputationPb "github.com/textileio/powergate/reputation/pb"
	txndstr "github.com/textileio/powergate/txndstransform"
	"github.com/textileio/powergate/util"
	"github.com/textileio/powergate/wallet"
	walletPb "github.com/textileio/powergate/wallet/pb"
	"google.golang.org/grpc"
)

const (
	datastoreFolderName = "datastore"
)

var (
	log = logging.Logger("server")
)

// Server represents the configured lotus client and filecoin grpc server
type Server struct {
	ds datastore.TxnDatastore

	ip2l *ip2location.IP2Location
	ai   *ask.Runner
	mi   *miner.Index
	si   *slashing.Index
	dm   *deals.Module
	wm   *wallet.Module
	rm   *reputation.Module
	nm   pgnet.Module
	hm   *health.Module

	ffsManager *manager.Manager
	js         *jstore.Store
	cis        *cistore.Store
	as         *astore.Store
	sched      *scheduler.Scheduler
	hs         ffs.HotStorage
	l          *cidlogger.CidLogger

	grpcServer   *grpc.Server
	grpcWebProxy *http.Server

	gateway     *gateway.Gateway
	indexServer *http.Server

	closeLotus func()
}

// Config specifies server settings.
type Config struct {
	WalletInitialFunds  big.Int
	IpfsAPIAddr         ma.Multiaddr
	LotusAddress        ma.Multiaddr
	LotusAuthToken      string
	LotusMasterAddr     string
	Embedded            bool
	GrpcHostNetwork     string
	GrpcHostAddress     multiaddr.Multiaddr
	GrpcServerOpts      []grpc.ServerOption
	GrpcWebProxyAddress string
	RepoPath            string
	GatewayHostAddr     string
}

// NewServer starts and returns a new server with the given configuration.
func NewServer(conf Config) (*Server, error) {
	var err error
	var masterAddr address.Address
	c, cls, err := lotus.New(conf.LotusAddress, conf.LotusAuthToken)
	if err != nil {
		return nil, fmt.Errorf("connecting to lotus node: %s", err)
	}

	if conf.Embedded {
		// Wait for the devnet to bootstrap completely and generate at least 1 block.
		time.Sleep(time.Second * 6)
		if masterAddr, err = c.WalletDefaultAddress(context.Background()); err != nil {
			return nil, fmt.Errorf("getting default wallet addr as masteraddr: %s", err)
		}
	} else {
		if masterAddr, err = address.NewFromString(conf.LotusMasterAddr); err != nil {
			return nil, fmt.Errorf("parsing masteraddr: %s", err)
		}
	}

	fchost, err := fchost.New(!conf.Embedded)
	if err != nil {
		return nil, fmt.Errorf("creating filecoin host: %s", err)
	}
	if !conf.Embedded {
		if err := fchost.Bootstrap(); err != nil {
			return nil, fmt.Errorf("bootstrapping filecoin host: %s", err)
		}
	}

	path := filepath.Join(conf.RepoPath, datastoreFolderName)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return nil, fmt.Errorf("creating repo folder: %s", err)
	}

	opts := &badger.DefaultOptions
	opts.NumVersionsToKeep = 0
	ds, err := badger.NewDatastore(path, opts)
	if err != nil {
		return nil, fmt.Errorf("opening datastore on repo: %s", err)
	}

	ip2l := ip2location.New([]string{"./ip2location-ip4.bin"})
	ai, err := ask.New(txndstr.Wrap(ds, "index/ask"), c)
	if err != nil {
		return nil, fmt.Errorf("creating ask index: %s", err)
	}
	mi, err := miner.New(txndstr.Wrap(ds, "index/miner"), c, fchost, ip2l)
	if err != nil {
		return nil, fmt.Errorf("creating miner index: %s", err)
	}
	si, err := slashing.New(txndstr.Wrap(ds, "index/slashing"), c)
	if err != nil {
		return nil, fmt.Errorf("creating slashing index: %s", err)
	}
	dm, err := deals.New(c, deals.WithImportPath(filepath.Join(conf.RepoPath, "imports")))
	if err != nil {
		return nil, fmt.Errorf("creating deal module: %s", err)
	}
	wm, err := wallet.New(c, masterAddr, conf.WalletInitialFunds)
	if err != nil {
		return nil, fmt.Errorf("creating wallet module: %s", err)
	}
	rm := reputation.New(txndstr.Wrap(ds, "reputation"), mi, si, ai)
	nm := pgnetlotus.New(c, ip2l)
	hm := health.New(nm)

	ipfs, err := httpapi.NewApi(conf.IpfsAPIAddr)
	if err != nil {
		return nil, fmt.Errorf("creating ipfs client: %s", err)
	}

	lchain := lotuschain.New(c)
	ms := reptop.New(rm, ai)

	l := cidlogger.New(txndstr.Wrap(ds, "ffs/scheduler/logger"))
	cs := filcold.New(ms, dm, ipfs.Dag(), lchain, l)
	hs := coreipfs.New(ipfs, l)
	js := jstore.New(txndstr.Wrap(ds, "ffs/scheduler/jstore"))
	as := astore.New(txndstr.Wrap(ds, "ffs/scheduler/astore"))
	cis := cistore.New(txndstr.Wrap(ds, "ffs/scheduler/cistore"))
	sched := scheduler.New(js, as, cis, l, hs, cs)

	ffsManager, err := manager.New(txndstr.Wrap(ds, "ffs/manager"), wm, sched)
	if err != nil {
		return nil, fmt.Errorf("creating ffs instance: %s", err)
	}

	grpcServer, grpcWebProxy := createGRPCServer(conf.GrpcServerOpts, conf.GrpcWebProxyAddress)

	gateway := gateway.NewGateway(conf.GatewayHostAddr, ai, mi, si, rm)
	gateway.Start()

	s := &Server{
		ds: ds,

		ip2l: ip2l,

		ai: ai,
		mi: mi,
		si: si,
		dm: dm,
		wm: wm,
		rm: rm,
		nm: nm,
		hm: hm,

		ffsManager: ffsManager,
		sched:      sched,
		js:         js,
		cis:        cis,
		as:         as,
		hs:         hs,
		l:          l,

		grpcServer:   grpcServer,
		grpcWebProxy: grpcWebProxy,
		gateway:      gateway,
		closeLotus:   cls,
	}

	if err := startGRPCServices(grpcServer, grpcWebProxy, s, conf.GrpcHostNetwork, conf.GrpcHostAddress); err != nil {
		return nil, fmt.Errorf("starting GRPC services: %s", err)
	}

	s.indexServer = startIndexHTTPServer(s)

	return s, nil
}

func createGRPCServer(opts []grpc.ServerOption, webProxyAddr string) (*grpc.Server, *http.Server) {
	grpcServer := grpc.NewServer(opts...)

	wrappedServer := grpcweb.WrapServer(
		grpcServer,
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true
		}),
		grpcweb.WithWebsockets(true),
		grpcweb.WithWebsocketOriginFunc(func(req *http.Request) bool {
			return true
		}),
	)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wrappedServer.IsGrpcWebRequest(r) ||
			wrappedServer.IsAcceptableGrpcCorsRequest(r) ||
			wrappedServer.IsGrpcWebSocketRequest(r) {
			wrappedServer.ServeHTTP(w, r)
		}
	})
	grpcWebProxy := &http.Server{
		Addr:    webProxyAddr,
		Handler: handler,
	}
	return grpcServer, grpcWebProxy
}

func startGRPCServices(server *grpc.Server, webProxy *http.Server, s *Server, hostNetwork string, hostAddress multiaddr.Multiaddr) error {
	netService := pgnetRpc.NewService(s.nm)
	healthService := healthRpc.NewService(s.hm)
	dealsService := deals.NewService(s.dm)
	walletService := wallet.NewService(s.wm)
	reputationService := reputation.NewService(s.rm)
	askService := askRpc.New(s.ai)
	minerService := miner.NewService(s.mi)
	slashingService := slashing.NewService(s.si)
	ffsService := ffsGrpc.NewService(s.ffsManager, s.hs)

	hostAddr, err := util.TCPAddrFromMultiAddr(hostAddress)
	if err != nil {
		return fmt.Errorf("parsing host multiaddr: %s", err)
	}
	listener, err := net.Listen(hostNetwork, hostAddr)
	if err != nil {
		return fmt.Errorf("listening to grpc: %s", err)
	}
	go func() {
		pgnetRpc.RegisterNetServer(server, netService)
		healthRpc.RegisterHealthServer(server, healthService)
		dealsPb.RegisterAPIServer(server, dealsService)
		walletPb.RegisterAPIServer(server, walletService)
		reputationPb.RegisterAPIServer(server, reputationService)
		askPb.RegisterAPIServer(server, askService)
		minerPb.RegisterAPIServer(server, minerService)
		slashingPb.RegisterAPIServer(server, slashingService)
		ffsRpc.RegisterFFSServer(server, ffsService)
		if err := server.Serve(listener); err != nil {
			log.Errorf("serving grpc endpoint: %s", err)
		}
	}()

	go func() {
		if err := webProxy.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("error starting proxy: %v", err)
		}
	}()
	return nil
}

func startIndexHTTPServer(s *Server) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/index/ask", func(w http.ResponseWriter, r *http.Request) {
		index := s.ai.Get()
		buf, err := json.MarshalIndent(index, "", "  ")
		if err != nil {
			http.Error(w, "Error", http.StatusInternalServerError)
			return
		}
		if _, err := w.Write(buf); err != nil {
			log.Errorf("writing response body: %s", err)
		}
	})
	mux.HandleFunc("/index/miners", func(w http.ResponseWriter, r *http.Request) {
		index := s.mi.Get()
		buf, err := json.MarshalIndent(index, "", "  ")
		if err != nil {
			http.Error(w, "Error", http.StatusInternalServerError)
			return
		}
		if _, err := w.Write(buf); err != nil {
			log.Errorf("writing response body: %s", err)
		}
	})
	mux.HandleFunc("/index/slashing", func(w http.ResponseWriter, r *http.Request) {
		index := s.si.Get()
		buf, err := json.MarshalIndent(index, "", "  ")
		if err != nil {
			http.Error(w, "Error", http.StatusInternalServerError)
			return
		}
		if _, err := w.Write(buf); err != nil {
			log.Errorf("writing response body: %s", err)
		}
	})

	srv := &http.Server{Addr: ":8889", Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serving index http: %v", err)
		}
	}()
	return srv
}

// Close shuts down the server
func (s *Server) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.indexServer.Shutdown(ctx); err != nil {
		log.Errorf("shutting down index server: %s", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.grpcWebProxy.Shutdown(ctx); err != nil {
		log.Errorf("error shutting down proxy: %s", err)
	}
	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()
	t := time.NewTimer(10 * time.Second)
	select {
	case <-t.C:
		s.grpcServer.Stop()
	case <-stopped:
		t.Stop()
	}
	if err := s.ffsManager.Close(); err != nil {
		log.Errorf("closing ffs manager: %s", err)
	}
	if err := s.sched.Close(); err != nil {
		log.Errorf("closing ffs scheduler: %s", err)
	}
	if err := s.js.Close(); err != nil {
		log.Errorf("closing scheduler jobstore: %s", err)
	}
	if err := s.l.Close(); err != nil {
		log.Errorf("closing cid logger: %s", err)
	}
	if err := s.rm.Close(); err != nil {
		log.Errorf("closing reputation module: %s", err)
	}
	if err := s.ai.Close(); err != nil {
		log.Errorf("closing ask index: %s", err)
	}
	if err := s.mi.Close(); err != nil {
		log.Errorf("closing miner index: %s", err)
	}
	if err := s.si.Close(); err != nil {
		log.Errorf("closing slashing index: %s", err)
	}
	if err := s.ds.Close(); err != nil {
		log.Errorf("closing datastore: %s", err)
	}
	if err := s.gateway.Stop(); err != nil {
		log.Errorf("closing gateway: %s", err)
	}
	s.closeLotus()
	s.ip2l.Close()
}
