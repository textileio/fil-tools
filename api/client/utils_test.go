package client

import (
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"github.com/textileio/powergate/api/server"
	"github.com/textileio/powergate/tests"
	"github.com/textileio/powergate/util"
	"google.golang.org/grpc"
)

var (
	grpcHostNetwork     = "tcp"
	grpcHostAddress     = "/ip4/127.0.0.1/tcp/5002"
	grpcWebProxyAddress = "127.0.0.1:6002"
	gatewayHostAddr     = "0.0.0.0:7000"
)

func defaultServerConfig(t *testing.T) server.Config {
	repoPath, err := ioutil.TempDir("/tmp/powergate", ".powergate-*")
	require.NoError(t, err)

	dipfs, cls := tests.LaunchIPFSDocker(t)
	t.Cleanup(func() { cls() })

	ipfsAddrStr := "/ip4/127.0.0.1/tcp/" + dipfs.GetPort("5001/tcp")
	ipfsAddr := util.MustParseAddr(ipfsAddrStr)

	devnet := tests.LaunchDevnetDocker(t, 1, ipfsAddrStr, false)
	devnetAddr := util.MustParseAddr("/ip4/127.0.0.1/tcp/" + devnet.GetPort("7777/tcp"))

	grpcMaddr := util.MustParseAddr(grpcHostAddress)
	conf := server.Config{
		WalletInitialFunds:          *big.NewInt(int64(4000000000)),
		IpfsAPIAddr:                 ipfsAddr,
		LotusAddress:                devnetAddr,
		LotusAuthToken:              "",
		LotusMasterAddr:             "",
		LotusConnectionRetries:      5,
		Devnet:                      true,
		GrpcHostNetwork:             grpcHostNetwork,
		GrpcHostAddress:             grpcMaddr,
		GrpcWebProxyAddress:         grpcWebProxyAddress,
		RepoPath:                    repoPath,
		GatewayHostAddr:             gatewayHostAddr,
		MaxMindDBFolder:             "../../iplocation/maxmind",
		MinerSelector:               "reputation",
		FFSDealFinalityTimeout:      time.Minute * 30,
		FFSMaxParallelDealPreparing: 1,
		DealWatchPollDuration:       time.Second * 15,
		SchedMaxParallel:            10,
		AskIndexQueryAskTimeout:     time.Second * 3,
		AskIndexRefreshInterval:     time.Second * 3,
		AskIndexRefreshOnStart:      true,
		AskindexMaxParallel:         2,
	}
	return conf
}

func setupServer(t *testing.T, conf server.Config) func() {
	server, err := server.NewServer(conf)
	require.NoError(t, err)

	return func() {
		server.Close()
	}
}

func setupConnection(t *testing.T) (*grpc.ClientConn, func()) {
	auth := TokenAuth{}
	ma, err := multiaddr.NewMultiaddr(grpcHostAddress)
	require.NoError(t, err)
	addr, err := util.TCPAddrFromMultiAddr(ma)
	require.NoError(t, err)
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithPerRPCCredentials(auth))
	require.NoError(t, err)
	return conn, func() {
		require.NoError(t, conn.Close())
	}
}
