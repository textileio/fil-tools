package coreipfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	ipfsfiles "github.com/ipfs/go-ipfs-files"
	logging "github.com/ipfs/go-log/v2"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/textileio/powergate/ffs"
)

var (
	log = logging.Logger("ffs-coreipfs")
)

// CoreIpfs is an implementation of HotStorage interface which saves data
// into a remote go-ipfs using the HTTP API.
type CoreIpfs struct {
	ipfs iface.CoreAPI
	l    ffs.CidLogger

	lock   sync.Mutex
	pinset map[cid.Cid]struct{}
}

var _ ffs.HotStorage = (*CoreIpfs)(nil)

// New returns a new CoreIpfs instance.
func New(ipfs iface.CoreAPI, l ffs.CidLogger) *CoreIpfs {
	return &CoreIpfs{
		ipfs: ipfs,
		l:    l,
	}
}

// Put saves a Block.
func (ci *CoreIpfs) Put(ctx context.Context, b blocks.Block) error {
	log.Debugf("putting block %s", b.Cid())
	if _, err := ci.ipfs.Block().Put(ctx, bytes.NewReader(b.RawData())); err != nil {
		return fmt.Errorf("adding block to ipfs node: %s", err)
	}
	return nil
}

// Remove removes a Cid from the Hot Storage.
func (ci *CoreIpfs) Remove(ctx context.Context, c cid.Cid) error {
	log.Debugf("removing cid %s", c)
	if err := ci.ipfs.Pin().Rm(ctx, path.IpfsPath(c), options.Pin.RmRecursive(true)); err != nil {
		return fmt.Errorf("unpinning cid from ipfs node: %s", err)
	}
	ci.l.Log(ctx, c, "Cid data was pinned in IPFS node.")
	return nil
}

// IsStored return if a particular Cid is stored.
func (ci *CoreIpfs) IsStored(ctx context.Context, c cid.Cid) (bool, error) {
	if ci.pinset == nil {
		if err := ci.ensurePinsetCache(ctx); err != nil {
			return false, err
		}
	}
	_, ok := ci.pinset[c]
	return ok, nil
}

// Add adds an io.Reader data as file in the IPFS node.
func (ci *CoreIpfs) Add(ctx context.Context, r io.Reader) (cid.Cid, error) {
	log.Debugf("adding data-stream...")
	p, err := ci.ipfs.Unixfs().Add(ctx, ipfsfiles.NewReaderFile(r), options.Unixfs.Pin(false))
	if err != nil {
		return cid.Undef, fmt.Errorf("adding data to ipfs: %s", err)
	}
	log.Debugf("data-stream added with cid %s", p.Cid())
	return p.Cid(), nil
}

// Get retrieves a cid from the IPFS node.
func (ci *CoreIpfs) Get(ctx context.Context, c cid.Cid) (io.Reader, error) {
	log.Debugf("getting cid %s", c)
	n, err := ci.ipfs.Unixfs().Get(ctx, path.IpfsPath(c))
	if err != nil {
		return nil, fmt.Errorf("getting cid %s from ipfs: %s", c, err)
	}
	file := ipfsfiles.ToFile(n)
	if file == nil {
		return nil, fmt.Errorf("node is a directory")
	}
	return file, nil
}

// Store stores a Cid in the HotStorage. At the IPFS level, it also mark the Cid as pinned.
func (ci *CoreIpfs) Store(ctx context.Context, c cid.Cid) (int, error) {
	log.Debugf("fetching and pinning cid %s", c)
	p := path.IpfsPath(c)
	if err := ci.ipfs.Pin().Add(ctx, p, options.Pin.Recursive(true)); err != nil {
		return 0, fmt.Errorf("pinning cid %s: %s", c, err)
	}
	s, err := ci.ipfs.Object().Stat(ctx, p)
	if err != nil {
		return 0, fmt.Errorf("getting stats of cid %s: %s", c, err)
	}
	if err := ci.ensurePinsetCache(ctx); err != nil {
		return 0, err
	}
	ci.lock.Lock()
	ci.pinset[c] = struct{}{}
	ci.lock.Unlock()
	return s.CumulativeSize, nil
}

// Replace replaces a stored Cid with other Cid.
func (ci *CoreIpfs) Replace(ctx context.Context, c1 cid.Cid, c2 cid.Cid) (int, error) {
	p1 := path.IpfsPath(c1)
	p2 := path.IpfsPath(c2)
	log.Debugf("updating pin from %s to %s", p1, p2)
	if err := ci.ipfs.Pin().Update(ctx, p1, p2); err != nil {
		return 0, fmt.Errorf("updating pin %s to %s: %s", c1, c2, err)
	}
	stat, err := ci.ipfs.Block().Stat(ctx, p2)
	if err != nil {
		return 0, fmt.Errorf("getting stats of cid %s: %s", c2, err)
	}
	ci.lock.Lock()
	delete(ci.pinset, c1)
	ci.pinset[c2] = struct{}{}
	ci.lock.Unlock()

	return stat.Size(), nil
}

func (ci *CoreIpfs) ensurePinsetCache(ctx context.Context) error {
	ci.lock.Lock()
	defer ci.lock.Unlock()
	if ci.pinset != nil {
		return nil
	}
	pins, err := ci.ipfs.Pin().Ls(ctx)
	if err != nil {
		ci.lock.Unlock()
		return fmt.Errorf("getting pins from IPFS: %s", err)
	}
	ci.pinset = make(map[cid.Cid]struct{}, len(pins))
	for _, p := range pins {
		ci.pinset[p.Path().Cid()] = struct{}{}
	}
	return nil
}
