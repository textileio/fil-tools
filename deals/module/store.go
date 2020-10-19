package module

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/textileio/powergate/deals"
	"github.com/textileio/powergate/util"
)

var (
	dsBaseStoragePending = datastore.NewKey("storage-pending")
	dsBaseStorageFinal   = datastore.NewKey("storage-final")
	dsBaseRetrieval      = datastore.NewKey("retrieval")

	// ErrNotFound indicates the instance doesn't exist.
	ErrNotFound = errors.New("cid info not found")
)

type store struct {
	ds   datastore.Datastore
	lock sync.Mutex
}

func newStore(ds datastore.Datastore) *store {
	return &store{
		ds: ds,
	}
}

func (s *store) putPendingDeal(dr deals.StorageDealRecord) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	buf, err := json.Marshal(dr)
	if err != nil {
		return fmt.Errorf("marshaling PendingDeal: %s", err)
	}
	if err := s.ds.Put(makePendingDealKey(dr.DealInfo.ProposalCid), buf); err != nil {
		return fmt.Errorf("put PendingDeal: %s", err)
	}
	return nil
}

func (s *store) deletePendingDeal(proposalCid cid.Cid) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if err := s.ds.Delete(makePendingDealKey(proposalCid)); err != nil {
		return fmt.Errorf("delete PendingDeal: %s", err)
	}
	return nil
}

func (s *store) getPendingDeals() ([]deals.StorageDealRecord, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	q := query.Query{Prefix: dsBaseStoragePending.String()}
	res, err := s.ds.Query(q)
	if err != nil {
		return nil, fmt.Errorf("executing query: %s", err)
	}
	defer func() {
		if err := res.Close(); err != nil {
			log.Errorf("closing query result: %s", err)
		}
	}()

	var ret []deals.StorageDealRecord
	for r := range res.Next() {
		if r.Error != nil {
			return nil, fmt.Errorf("iter next: %s", r.Error)
		}
		var dr deals.StorageDealRecord
		if err := json.Unmarshal(r.Value, &dr); err != nil {
			return nil, fmt.Errorf("unmarshaling query result: %s", err)
		}
		ret = append(ret, dr)
	}
	return ret, nil
}

func (s *store) putFinalDeal(dr deals.StorageDealRecord) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	err := s.ds.Delete(makePendingDealKey(dr.DealInfo.ProposalCid))
	if err != nil {
		return fmt.Errorf("deleting PendingDeal: %s", err)
	}
	buf, err := json.Marshal(dr)
	if err != nil {
		return fmt.Errorf("marshaling DealRecord: %s", err)
	}
	if err := s.ds.Put(makeFinalDealKey(dr.DealInfo.ProposalCid), buf); err != nil {
		return fmt.Errorf("put DealRecord: %s", err)
	}
	return nil
}

func (s *store) getFinalDeals() ([]deals.StorageDealRecord, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	q := query.Query{Prefix: dsBaseStorageFinal.String()}
	res, err := s.ds.Query(q)
	if err != nil {
		return nil, fmt.Errorf("executing query: %s", err)
	}
	defer func() {
		if err := res.Close(); err != nil {
			log.Errorf("closing query result: %s", err)
		}
	}()

	var ret []deals.StorageDealRecord
	for r := range res.Next() {
		if r.Error != nil {
			return nil, fmt.Errorf("iter next: %s", r.Error)
		}
		var dealRecord deals.StorageDealRecord
		if err := json.Unmarshal(r.Value, &dealRecord); err != nil {
			return nil, fmt.Errorf("unmarshaling query result: %s", err)
		}
		ret = append(ret, dealRecord)
	}
	return ret, nil
}

func (s *store) putRetrieval(rr deals.RetrievalDealRecord) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	buf, err := json.Marshal(rr)
	if err != nil {
		return fmt.Errorf("marshaling RetrievalRecord: %s", err)
	}
	if err := s.ds.Put(makeRetrievalKey(rr), buf); err != nil {
		return fmt.Errorf("put RetrievalRecord: %s", err)
	}
	return nil
}

func (s *store) getRetrievals() ([]deals.RetrievalDealRecord, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	q := query.Query{Prefix: dsBaseRetrieval.String()}
	res, err := s.ds.Query(q)
	if err != nil {
		return nil, fmt.Errorf("executing query: %s", err)
	}
	defer func() {
		if err := res.Close(); err != nil {
			log.Errorf("closing query result: %s", err)
		}
	}()

	var ret []deals.RetrievalDealRecord
	for r := range res.Next() {
		if r.Error != nil {
			return nil, fmt.Errorf("iter next: %s", r.Error)
		}
		var rr deals.RetrievalDealRecord
		if err := json.Unmarshal(r.Value, &rr); err != nil {
			return nil, fmt.Errorf("unmarshaling query result: %s", err)
		}
		ret = append(ret, rr)
	}
	return ret, nil
}

func makePendingDealKey(c cid.Cid) datastore.Key {
	return dsBaseStoragePending.ChildString(util.CidToString(c))
}

func makeFinalDealKey(c cid.Cid) datastore.Key {
	return dsBaseStorageFinal.ChildString(util.CidToString(c))
}

func makeRetrievalKey(rr deals.RetrievalDealRecord) datastore.Key {
	str := fmt.Sprintf("%v%v%v%v", rr.Time, rr.Addr, rr.DealInfo.Miner, util.CidToString(rr.DealInfo.RootCid))
	sum := md5.Sum([]byte(str))
	return dsBaseRetrieval.ChildString(fmt.Sprintf("%x", sum[:]))
}
