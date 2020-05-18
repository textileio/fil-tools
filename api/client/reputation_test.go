package client

import (
	"testing"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/textileio/powergate/reputation/rpc"
)

func TestAddSource(t *testing.T) {
	skipIfShort(t)
	r, done := setupReputation(t)
	defer done()

	maddr, err := ma.NewMultiaddr("/dns4/lotus-bootstrap-0.sin.fil-test.net/tcp/1347/p2p/12D3KooWLZs8BWtEzRTYET4yR4jzDtPamaA1YsyPQJq6cf2RfxBD")
	checkErr(t, err)

	err = r.AddSource(ctx, "id", maddr)
	if err != nil {
		t.Fatalf("failed to call AddSource: %v", err)
	}
}

func TestGetTopMiners(t *testing.T) {
	skipIfShort(t)
	r, done := setupReputation(t)
	defer done()

	_, err := r.GetTopMiners(ctx, 10)
	if err != nil {
		t.Fatalf("failed to call GetTopMiners: %v", err)
	}
}

func setupReputation(t *testing.T) (*Reputation, func()) {
	serverDone := setupServer(t)
	conn, done := setupConnection(t)
	return &Reputation{client: rpc.NewRPCClient(conn)}, func() {
		done()
		serverDone()
	}
}
