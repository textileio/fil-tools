package fchost

import (
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/config"
	"github.com/multiformats/go-multiaddr"
)

var (
	addrs = []string{
		"/dns4/bootstrap-0-sin.fil-test.net/tcp/1347/p2p/12D3KooWKNF7vNFEhnvB45E9mw2B5z6t419W3ziZPLdUDVnLLKGs",
		"/ip4/86.109.15.57/tcp/1347/p2p/12D3KooWKNF7vNFEhnvB45E9mw2B5z6t419W3ziZPLdUDVnLLKGs",
		"/dns4/bootstrap-0-dfw.fil-test.net/tcp/1347/p2p/12D3KooWECJTm7RUPyGfNbRwm6y2fK4wA7EB8rDJtWsq5AKi7iDr",
		"/ip4/139.178.84.45/tcp/1347/p2p/12D3KooWECJTm7RUPyGfNbRwm6y2fK4wA7EB8rDJtWsq5AKi7iDr",
		"/dns4/bootstrap-0-fra.fil-test.net/tcp/1347/p2p/12D3KooWC7MD6m7iNCuDsYtNr7xVtazihyVUizBbhmhEiyMAm9ym",
		"/ip4/136.144.49.17/tcp/1347/p2p/12D3KooWC7MD6m7iNCuDsYtNr7xVtazihyVUizBbhmhEiyMAm9ym",
		"/dns4/bootstrap-1-sin.fil-test.net/tcp/1347/p2p/12D3KooWD8eYqsKcEMFax6EbWN3rjA7qFsxCez2rmN8dWqkzgNaN",
		"/ip4/86.109.15.55/tcp/1347/p2p/12D3KooWD8eYqsKcEMFax6EbWN3rjA7qFsxCez2rmN8dWqkzgNaN",
		"/dns4/bootstrap-1-dfw.fil-test.net/tcp/1347/p2p/12D3KooWLB3RR8frLAmaK4ntHC2dwrAjyGzQgyUzWxAum1FxyyqD",
		"/ip4/139.178.84.41/tcp/1347/p2p/12D3KooWLB3RR8frLAmaK4ntHC2dwrAjyGzQgyUzWxAum1FxyyqD",
		"/dns4/bootstrap-1-fra.fil-test.net/tcp/1347/p2p/12D3KooWGPDJAw3HW4uVU3JEQBfFaZ1kdpg4HvvwRMVpUYbzhsLQ",
		"/ip4/136.144.49.131/tcp/1347/p2p/12D3KooWGPDJAw3HW4uVU3JEQBfFaZ1kdpg4HvvwRMVpUYbzhsLQ",
	}
)

func getBootstrapPeers() []peer.AddrInfo {
	maddrs := make([]multiaddr.Multiaddr, len(addrs))
	for i, addr := range addrs {
		var err error
		maddrs[i], err = multiaddr.NewMultiaddr(addr)
		if err != nil {
			panic(err)
		}
	}
	peers, err := peer.AddrInfosFromP2pAddrs(maddrs...)
	if err != nil {
		panic(err)
	}
	return peers
}

func getDefaultOpts() []config.Option {
	return []config.Option{libp2p.Defaults}
}
