package fchost

import (
	"context"
	"fmt"
	"sync"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/multiformats/go-multiaddr"
)

var (
	log = logging.Logger("fchost")
)

// FilecoinHost is a libp2p host connected to the FC network.
type FilecoinHost struct {
	ping *ping.PingService
	h    host.Host
	dht  *dht.IpfsDHT
}

// New returns a new FilecoinHost.
func New(network string, bootstrap bool) (*FilecoinHost, error) {
	ctx := context.Background()
	opts := getDefaultOpts()
	h, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	dht, err := dht.New(ctx, h, dht.ProtocolPrefix(protocol.ID("/fil/kad/"+network)))
	if err != nil {
		return nil, err
	}

	if bootstrap {
		if err := connectToBootstrapPeers(network, h); err != nil {
			return nil, err
		}
	}

	h = routedhost.Wrap(h, dht)
	return &FilecoinHost{
		h:    h,
		dht:  dht,
		ping: ping.NewPingService(h),
	}, nil
}

// Bootstrap connects to the bootstrap peers.
func (fc *FilecoinHost) Bootstrap() error {
	log.Info("bootstraping libp2p host dht")
	if err := fc.dht.Bootstrap(context.Background()); err != nil {
		return err
	}
	log.Info("dht bootstraped!")
	return nil
}

// Ping runs the ping protocol with a peer, returns true on success or false
// otherwise.
func (fc *FilecoinHost) Ping(ctx context.Context, pid peer.ID) bool {
	r := <-fc.ping.Ping(ctx, pid)
	return r.Error == nil
}

// GetAgentVersion returns the agent version of the peer, or empty otherwise.
func (fc *FilecoinHost) GetAgentVersion(pid peer.ID) string {
	if v, err := fc.h.Peerstore().Get(pid, "AgentVersion"); err == nil {
		agent, ok := v.(string)
		if ok {
			return agent
		}
	}
	return ""
}

// Addrs returns the known multiaddresses known of a peer.
func (fc *FilecoinHost) Addrs(pid peer.ID) []multiaddr.Multiaddr {
	return fc.h.Peerstore().Addrs(pid)
}

func connectToBootstrapPeers(network string, h host.Host) error {
	peers, err := getBootstrapPeers(network)
	if err != nil {
		return fmt.Errorf("getting bootstrap peers: %s", err)
	}
	ctx := context.Background()
	var lock sync.Mutex
	var success int
	var wg sync.WaitGroup
	wg.Add(len(peers))
	for _, ai := range peers {
		go func(ai peer.AddrInfo) {
			defer wg.Done()
			if err := h.Connect(ctx, ai); err != nil {
				return
			}
			lock.Lock()
			success++
			lock.Unlock()
		}(ai)
	}
	wg.Wait()
	if success == 0 {
		return fmt.Errorf("couldn't connect to any of bootstrap peers")
	}
	log.Infof("connected to %d out of %d bootstrap peers", success, len(peers))
	return nil
}
