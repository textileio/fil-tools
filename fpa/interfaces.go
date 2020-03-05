package fpa

import (
	"context"
	"io"

	"github.com/ipfs/go-cid"
)

// WalletManager provides access to a Lotus wallet for a Lotus node.
type WalletManager interface {
	NewWallet(ctx context.Context, typ string) (string, error)
	Balance(ctx context.Context, addr string) (uint64, error)
}

// Scheduler creates and manages Job which executes Cid configurations
// in Hot and Cold layers, enables retrieval from those layers, and
// allows watching for Job state changes.
type Scheduler interface {
	Enqueue(CidConfig) (JobID, error)
	GetFromHot(ctx context.Context, c cid.Cid) (io.Reader, error)

	Watch(InstanceID) <-chan Job
	Unwatch(<-chan Job)
}

// HotLyer is a fast datastorage layer for storing and retrieving raw
// data or Cids.
type HotLayer interface {
	Add(context.Context, io.Reader) (cid.Cid, error)
	Get(context.Context, cid.Cid) (io.Reader, error)
	Pin(context.Context, cid.Cid) (HotInfo, error)
}

// ColdLayer is a slow datastorage layer for storing Cids.
type ColdLayer interface {
	Store(ctx context.Context, c cid.Cid, conf ColdConfig) (ColdInfo, error)
}

// MinerSelector returns miner addresses and ask storage information using a
// desired strategy.
type MinerSelector interface {
	GetTopMiners(n int) ([]MinerProposal, error)
}

// MinerProposal contains a miners address and storage ask information
// to make a, most probably, successful deal.
type MinerProposal struct {
	Addr       string
	EpochPrice uint64
}
