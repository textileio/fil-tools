package module

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/textileio/powergate/deals"
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

func (s *store) putPendingDeal(dr deals.DealRecord) error {
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

func (s *store) getPendingDeals() ([]deals.DealRecord, error) {
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

	var ret []deals.DealRecord
	for r := range res.Next() {
		var dr deals.DealRecord
		if err := json.Unmarshal(r.Value, &dr); err != nil {
			return nil, fmt.Errorf("unmarshaling query result: %s", err)
		}
		ret = append(ret, dr)
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i].Time < ret[j].Time })
	return ret, nil
}

func (s *store) putFinalDeal(dr deals.DealRecord) error {
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

func (s *store) getFinalDeals() ([]deals.DealRecord, error) {
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

	var ret []deals.DealRecord
	for r := range res.Next() {
		var dealRecord deals.DealRecord
		if err := json.Unmarshal(r.Value, &dealRecord); err != nil {
			return nil, fmt.Errorf("unmarshaling query result: %s", err)
		}
		ret = append(ret, dealRecord)
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].DealInfo.ActivationEpoch < ret[j].DealInfo.ActivationEpoch {
			return true
		}
		if ret[i].DealInfo.ActivationEpoch > ret[j].DealInfo.ActivationEpoch {
			return false
		}
		return ret[i].Time < ret[j].Time
	})
	return ret, nil
}

func (s *store) putRetrieval(rr deals.RetrievalRecord) error {
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

func (s *store) getRetrievals() ([]deals.RetrievalRecord, error) {
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

	var ret []deals.RetrievalRecord
	for r := range res.Next() {
		var rr deals.RetrievalRecord
		if err := json.Unmarshal(r.Value, &rr); err != nil {
			return nil, fmt.Errorf("unmarshaling query result: %s", err)
		}
		ret = append(ret, rr)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Time < ret[j].Time
	})
	return ret, nil
}

func makePendingDealKey(c cid.Cid) datastore.Key {
	return dsBaseStoragePending.ChildString(c.String())
}

func makeFinalDealKey(c cid.Cid) datastore.Key {
	return dsBaseStorageFinal.ChildString(c.String())
}

func makeRetrievalKey(rr deals.RetrievalRecord) datastore.Key {
	str := fmt.Sprintf("%v%v%v%v", rr.Time, rr.Addr, rr.RetrievalInfo.Miner, rr.RetrievalInfo.PieceCID.String())
	sum := md5.Sum([]byte(str))
	return dsBaseRetrieval.ChildString(string(sum[:]))
}
