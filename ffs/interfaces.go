package ffs

import (
	"context"
	"errors"
	"io"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-car"
)

// WalletManager provides access to a Lotus wallet for a Lotus node.
type WalletManager interface {
	// NewAddress creates a new address.
	NewAddress(context.Context, string) (string, error)
	// Balance returns the current balance for an address.
	Balance(context.Context, string) (uint64, error)
}

var (
	// ErrBothStoragesDisabled returned when both storages are disabled.
	ErrBothStoragesDisabled = errors.New("both Hot and Cold layers can't be disabled")
	// ErrHotStorageDisabled returned when trying to fetch a Cid when disabled on Hot Storage.
	// To retrieve the data, is necessary to call unfreeze by enabling the Enabled flag in
	// the Hot Storage for that Cid.
	ErrHotStorageDisabled = errors.New("cid disabled in hot storage")
)

// Scheduler enforces a CidConfig orchestrating Hot and Cold storages.
type Scheduler interface {
	// PushConfig push a new or modified configuration for a Cid. It returns
	// the JobID which tracks the current state of execution of that task.
	PushConfig(PushConfigAction) (JobID, error)

	// GetCidInfo returns the current Cid storing state. This state may be different
	// from CidConfig which is the *desired* state.
	GetCidInfo(cid.Cid) (CidInfo, error)
	// GetCidFromHot returns an Reader with the Cid data. If the data isn't in the Hot
	// Storage, it errors with ErrHotStorageDisabled.
	GetCidFromHot(context.Context, cid.Cid) (io.Reader, error)

	// GetJob gets the a Job.
	GetJob(JobID) (Job, error)

	// Watch returns a channel which will receive updates for all Jobs created by
	// an Instance.
	Watch(ApiID) <-chan Job
	// Unwatch unregisters a subscribed channel.
	Unwatch(<-chan Job)
}

// HotStorage is a fast datastorage layer for storing and retrieving raw
// data or Cids.
type HotStorage interface {
	Add(context.Context, io.Reader) (cid.Cid, error)
	Get(context.Context, cid.Cid) (io.Reader, error)
	Pin(context.Context, cid.Cid) (int, error)
	Put(context.Context, blocks.Block) error
}

// ColdStorage is a slow datastorage layer for storing Cids.
type ColdStorage interface {
	Store(context.Context, cid.Cid, string, FilConfig) (FilInfo, error)
	Retrieve(context.Context, cid.Cid, car.Store, string) (cid.Cid, error)

	EnsureRenewals(context.Context, cid.Cid, FilInfo, string, FilConfig) (FilInfo, error)
}

// MinerSelector returns miner addresses and ask storage information using a
// desired strategy.
type MinerSelector interface {
	GetMiners(int, MinerSelectorFilter) ([]MinerProposal, error)
}

// MinerSelectorFilter establishes filters that should be considered when
// returning miners.
type MinerSelectorFilter struct {
	// Blacklist contains miner names that should not be considered in
	// returned results. An empty list means no blacklisting.
	Blacklist []string
	// CountryCodes contains long-ISO country names that should be
	// considered in selected miners. An empty list means no filtering.
	CountryCodes []string
}

// MinerProposal contains a miners address and storage ask information
// to make a, most probably, successful deal.
type MinerProposal struct {
	Addr       string
	EpochPrice uint64
}
