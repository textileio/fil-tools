package api

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/textileio/powergate/ffs"
	"github.com/textileio/powergate/util"
)

var (
	dsInstanceConfig       = datastore.NewKey("instanceconfig")
	dsBaseCidStorageConfig = datastore.NewKey("cidstorageconfig")
	dsBaseRetrievalRequest = datastore.NewKey("retrievalrequest")
)

type instanceStore struct {
	lock sync.Mutex
	ds   datastore.Datastore
}

func newInstanceStore(ds datastore.Datastore) *instanceStore {
	return &instanceStore{
		ds: ds,
	}
}

// putInstanceConfig saves general instance configuration configuration, such as
// default wallet address, default storage config, etc.
func (s *instanceStore) putInstanceConfig(c InstanceConfig) error {
	buf, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling instance config to datastore: %s", err)
	}
	if err := s.ds.Put(dsInstanceConfig, buf); err != nil {
		return fmt.Errorf("saving to datastore: %s", err)
	}
	return nil
}

// getInstanceConfig returns general instance configurations, such as default wallet address,
// default storage config, etc.
func (s *instanceStore) getInstanceConfig() (InstanceConfig, error) {
	buf, err := s.ds.Get(dsInstanceConfig)
	if err != nil {
		if err == datastore.ErrNotFound {
			return InstanceConfig{}, ErrNotFound
		}
		return InstanceConfig{}, fmt.Errorf("getting instance config from datastore: %s", err)
	}
	var c InstanceConfig
	if err := json.Unmarshal(buf, &c); err != nil {
		return InstanceConfig{}, fmt.Errorf("unmarshaling config from datastore: %s", err)
	}
	return c, nil
}

func (s *instanceStore) putStorageConfig(c cid.Cid, sc ffs.StorageConfig) error {
	if !c.Defined() {
		return fmt.Errorf("cid can't be undefined")
	}
	buf, err := json.Marshal(sc)
	if err != nil {
		return fmt.Errorf("marshaling cid config to datastore: %s", err)
	}
	if err := s.ds.Put(makeStorageConfigKey(c), buf); err != nil {
		return fmt.Errorf("saving cid config to datastore: %s", err)
	}
	return nil
}

func (s *instanceStore) removeStorageConfig(c cid.Cid) error {
	if !c.Defined() {
		return fmt.Errorf("cid can't be undefined")
	}
	if err := s.ds.Delete(makeStorageConfigKey(c)); err != nil {
		return fmt.Errorf("removing from datastore: %s", err)
	}
	return nil
}

func (s *instanceStore) getStorageConfigs(cids ...cid.Cid) (map[cid.Cid]ffs.StorageConfig, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	rawRes := make(map[cid.Cid][]byte)
	for _, cid := range cids {
		rawRes[cid] = nil
	}

	if len(cids) == 1 {
		// just getting a single value, do an explicit query for it
		buf, err := s.ds.Get(makeStorageConfigKey(cids[0]))
		if err != nil {
			if err == datastore.ErrNotFound {
				return nil, ErrNotFound
			}
			return nil, err
		}
		rawRes[cids[0]] = buf
	} else {
		// getting many or all values, so we have to query everything
		q := query.Query{
			Prefix: dsBaseCidStorageConfig.String(),
		}
		res, err := s.ds.Query(q)
		if err != nil {
			return nil, err
		}
		defer func() {
			if err := res.Close(); err != nil {
				log.Errorf("closing query result: %s", err)
			}
		}()
		for r := range res.Next() {
			if r.Error != nil {
				return nil, fmt.Errorf("iter next: %s", r.Error)
			}
			strCid := datastore.RawKey(r.Key).Name()
			c, err := util.CidFromString(strCid)
			if err != nil {
				return nil, fmt.Errorf("decoding cid: %s", err)
			}
			if len(rawRes) > 0 {
				// we have a filter, check it
				if _, ok := rawRes[c]; ok {
					rawRes[c] = r.Value
				}
			} else {
				// no filter, include everything
				rawRes[c] = r.Value
			}
		}
	}

	res := make(map[cid.Cid]ffs.StorageConfig, len(rawRes))
	for cid, buf := range rawRes {
		if buf == nil {
			// one of the provided filter cids wasn't found
			return nil, ErrNotFound
		}
		var conf ffs.StorageConfig
		if err := json.Unmarshal(buf, &conf); err != nil {
			return nil, fmt.Errorf("unmarshaling cid config from datastore: %s", err)
		}
		res[cid] = conf
	}

	return res, nil
}

func makeStorageConfigKey(c cid.Cid) datastore.Key {
	return dsBaseCidStorageConfig.ChildString(util.CidToString(c))
}

type retrievalRequest struct {
	ID            ffs.RetrievalID
	PayloadCid    cid.Cid
	PieceCid      cid.Cid
	Selector      string
	Miners        []string
	WalletAddress string
	MaxPrice      uint64
	JID           ffs.JobID
	CreatedAt     time.Time
}

func (s *instanceStore) putRetrievalRequest(rID ffs.RetrievalID, pyCid, piCid cid.Cid, selector string, miners []string, walletAddress string, maxPrice uint64, jid ffs.JobID) (retrievalRequest, error) {
	rr := retrievalRequest{
		ID:            rID,
		PayloadCid:    pyCid,
		PieceCid:      piCid,
		Selector:      selector,
		Miners:        miners,
		WalletAddress: walletAddress,
		MaxPrice:      maxPrice,
		JID:           jid,
		CreatedAt:     time.Now(),
	}
	buf, err := json.Marshal(rr)
	if err != nil {
		return retrievalRequest{}, fmt.Errorf("marshaling retrieval request for datastore: %s", err)
	}
	if err := s.ds.Put(makeRetrievalRequestKey(rID), buf); err != nil {
		return retrievalRequest{}, fmt.Errorf("saving cid config to datastore: %s", err)
	}
	return rr, nil
}

func (s *instanceStore) getRetrievalRequest(rid ffs.RetrievalID) (retrievalRequest, error) {
	buf, err := s.ds.Get(makeRetrievalRequestKey(rid))
	if err != nil {
		if err == datastore.ErrNotFound {
			return retrievalRequest{}, ErrNotFound
		}
		return retrievalRequest{}, fmt.Errorf("getting retrieval request from datastore: %s", err)
	}
	var rr retrievalRequest
	if err := json.Unmarshal(buf, &rr); err != nil {
		return retrievalRequest{}, fmt.Errorf("unmarshaling retrieval request from datastore: %s", err)
	}
	return rr, nil
}

func makeRetrievalRequestKey(rid ffs.RetrievalID) datastore.Key {
	return dsBaseRetrievalRequest.ChildString(rid.String())
}
