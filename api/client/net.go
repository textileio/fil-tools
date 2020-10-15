package client

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/textileio/powergate/iplocation"
	n "github.com/textileio/powergate/net"
	"github.com/textileio/powergate/net/rpc"
)

// Net provides the Net API.
type Net struct {
	client rpc.RPCServiceClient
}

// ListenAddr returns listener address info for the local node.
func (net *Net) ListenAddr(ctx context.Context) (peer.AddrInfo, error) {
	resp, err := net.client.ListenAddr(ctx, &rpc.ListenAddrRequest{})
	if err != nil {
		return peer.AddrInfo{}, err
	}
	addrs := make([]ma.Multiaddr, len(resp.AddrInfo.Addrs))
	for i, addr := range resp.AddrInfo.Addrs {
		ma, err := ma.NewMultiaddr(addr)
		if err != nil {
			return peer.AddrInfo{}, err
		}
		addrs[i] = ma
	}
	id, err := peer.Decode(resp.AddrInfo.Id)
	if err != nil {
		return peer.AddrInfo{}, err
	}
	return peer.AddrInfo{
		ID:    id,
		Addrs: addrs,
	}, nil
}

// Peers returns a list of peers.
func (net *Net) Peers(ctx context.Context) ([]n.PeerInfo, error) {
	resp, err := net.client.Peers(ctx, &rpc.PeersRequest{})
	if err != nil {
		return nil, err
	}
	peerInfos := make([]n.PeerInfo, len(resp.Peers))
	for i, p := range resp.Peers {
		peerInfo, err := fromProtoPeerInfo(p)
		if err != nil {
			return nil, err
		}
		peerInfos[i] = peerInfo
	}
	return peerInfos, nil
}

// FindPeer finds a peer by peer id.
func (net *Net) FindPeer(ctx context.Context, peerID peer.ID) (n.PeerInfo, error) {
	resp, err := net.client.FindPeer(ctx, &rpc.FindPeerRequest{PeerId: peerID.String()})
	if err != nil {
		return n.PeerInfo{}, err
	}
	return fromProtoPeerInfo(resp.PeerInfo)
}

// Connectedness returns the connection status to a peer.
func (net *Net) Connectedness(ctx context.Context, peerID peer.ID) (n.Connectedness, error) {
	resp, err := net.client.Connectedness(ctx, &rpc.ConnectednessRequest{PeerId: peerID.String()})
	if err != nil {
		return n.Error, err
	}
	var con n.Connectedness
	switch resp.Connectedness {
	case rpc.Connectedness_CONNECTEDNESS_CAN_CONNECT:
		con = n.CanConnect
	case rpc.Connectedness_CONNECTEDNESS_CANNOT_CONNECT:
		con = n.CannotConnect
	case rpc.Connectedness_CONNECTEDNESS_CONNECTED:
		con = n.Connected
	case rpc.Connectedness_CONNECTEDNESS_NOT_CONNECTED:
		con = n.NotConnected
	case rpc.Connectedness_CONNECTEDNESS_ERROR:
		con = n.Error
	default:
		con = n.Unspecified
	}
	return con, nil
}

func fromProtoPeerInfo(proto *rpc.PeerInfo) (n.PeerInfo, error) {
	addrs := make([]ma.Multiaddr, len(proto.AddrInfo.Addrs))
	for i, addr := range proto.AddrInfo.Addrs {
		ma, err := ma.NewMultiaddr(addr)
		if err != nil {
			return n.PeerInfo{}, err
		}
		addrs[i] = ma
	}
	id, err := peer.Decode(proto.AddrInfo.Id)
	if err != nil {
		return n.PeerInfo{}, err
	}
	peerInfo := n.PeerInfo{
		AddrInfo: peer.AddrInfo{
			ID:    id,
			Addrs: addrs,
		},
	}
	if proto.Location != nil {
		peerInfo.Location = &iplocation.Location{
			Country:   proto.Location.Country,
			Latitude:  proto.Location.Latitude,
			Longitude: proto.Location.Longitude,
		}
	}

	return peerInfo, nil
}
