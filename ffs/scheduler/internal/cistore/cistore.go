package cistore

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/textileio/powergate/ffs"
	"github.com/textileio/powergate/util"
)

var (
	// ErrNotFound indicates the instance doesn't exist.
	ErrNotFound = errors.New("storage info not found")
)

// Store is an Datastore implementation of StorageInfoStore.
type Store struct {
	ds datastore.Datastore
	// ToDo: Build some index in here for fetching by iid and cid etc
}

// New returns a new JobStore backed by the Datastore.
func New(ds datastore.Datastore) *Store {
	return &Store{
		ds: ds,
	}
}

// Get gets the current stored state of a Cid.
func (s *Store) Get(iid ffs.APIID, c cid.Cid) (ffs.StorageInfo, error) {
	var ci ffs.StorageInfo
	buf, err := s.ds.Get(makeKey(iid, c))
	if err == datastore.ErrNotFound {
		return ci, ErrNotFound
	}
	if err != nil {
		return ci, fmt.Errorf("getting cid info from datastore: %s", err)
	}
	if err := json.Unmarshal(buf, &ci); err != nil {
		return ci, fmt.Errorf("unmarshaling cid info from datastore: %s", err)
	}
	return ci, nil
}

// Put saves a new storage state for a Cid.
func (s *Store) Put(ci ffs.StorageInfo) error {
	if !ci.APIID.Valid() {
		return fmt.Errorf("instance id is invalid")
	}
	if !ci.Cid.Defined() {
		return fmt.Errorf("cid can't be undefined")
	}
	buf, err := json.Marshal(ci)
	if err != nil {
		return fmt.Errorf("marshaling storage info for datastore: %s", err)
	}
	if err := s.ds.Put(makeKey(ci.APIID, ci.Cid), buf); err != nil {
		return fmt.Errorf("put storage info in datastore: %s", err)
	}
	return nil
}

func makeKey(iid ffs.APIID, c cid.Cid) datastore.Key {
	return datastore.NewKey(iid.String()).ChildString(util.CidToString(c))
}
