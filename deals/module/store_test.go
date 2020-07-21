package module

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/textileio/powergate/deals"
	"github.com/textileio/powergate/tests"
	"github.com/textileio/powergate/util"
)

func TestPutPendingDeal(t *testing.T) {
	s := newStore(tests.NewTxMapDatastore())

	c1, err := util.CidFromString("QmSnuWmxptJZdLJpKRarxBMS2Ju2oANVrgbr2xWbie9b2D")
	require.NoError(t, err)
	err = s.putPendingDeal(deals.StorageDealRecord{Addr: "a", Time: time.Now().Unix(), Pending: true, DealInfo: deals.StorageDealInfo{ProposalCid: c1}})
	require.NoError(t, err)

	c2, err := util.CidFromString("QmSnuWmxptJZdLJpKRarxBMS2Ju2oANVrgbr2xWbie9b2E")
	require.NoError(t, err)
	err = s.putPendingDeal(deals.StorageDealRecord{Addr: "b", Time: time.Now().Unix(), Pending: true, DealInfo: deals.StorageDealInfo{ProposalCid: c2}})
	require.NoError(t, err)

	c3, err := util.CidFromString("QmSnuWmxptJZdLJpKRarxBMS2Ju2oANVrgbr2xWbie9b2F")
	require.NoError(t, err)
	err = s.putPendingDeal(deals.StorageDealRecord{Addr: "c", Time: time.Now().Unix(), Pending: true, DealInfo: deals.StorageDealInfo{ProposalCid: c3}})
	require.NoError(t, err)
}

func TestGetPendingDeals(t *testing.T) {
	s := newStore(tests.NewTxMapDatastore())

	now := time.Now().Unix()

	c1, err := util.CidFromString("QmSnuWmxptJZdLJpKRarxBMS2Ju2oANVrgbr2xWbie9b2D")
	require.NoError(t, err)
	err = s.putPendingDeal(deals.StorageDealRecord{Addr: "a", Time: now, Pending: true, DealInfo: deals.StorageDealInfo{ProposalCid: c1}})
	require.NoError(t, err)

	c2, err := util.CidFromString("QmSnuWmxptJZdLJpKRarxBMS2Ju2oANVrgbr2xWbie9b2E")
	require.NoError(t, err)
	err = s.putPendingDeal(deals.StorageDealRecord{Addr: "b", Time: now + 1, Pending: true, DealInfo: deals.StorageDealInfo{ProposalCid: c2}})
	require.NoError(t, err)

	c3, err := util.CidFromString("QmSnuWmxptJZdLJpKRarxBMS2Ju2oANVrgbr2xWbie9b2F")
	require.NoError(t, err)
	err = s.putPendingDeal(deals.StorageDealRecord{Addr: "c", Time: now + 3, Pending: true, DealInfo: deals.StorageDealInfo{ProposalCid: c3}})
	require.NoError(t, err)

	res, err := s.getPendingDeals()
	require.NoError(t, err)
	require.Len(t, res, 3)
}

func TestDeletePendingDeal(t *testing.T) {
	s := newStore(tests.NewTxMapDatastore())

	c1, err := util.CidFromString("QmSnuWmxptJZdLJpKRarxBMS2Ju2oANVrgbr2xWbie9b2D")
	require.NoError(t, err)
	dr := deals.StorageDealRecord{Addr: "a", Time: time.Now().Unix(), Pending: true, DealInfo: deals.StorageDealInfo{ProposalCid: c1}}
	err = s.putPendingDeal(dr)
	require.NoError(t, err)

	res, err := s.getPendingDeals()
	require.NoError(t, err)
	require.Len(t, res, 1)

	err = s.deletePendingDeal(dr.DealInfo.ProposalCid)
	require.NoError(t, err)

	res, err = s.getPendingDeals()
	require.NoError(t, err)
	require.Empty(t, res)
}

func TestPutDealRecord(t *testing.T) {
	s := newStore(tests.NewTxMapDatastore())

	c1, err := util.CidFromString("QmSnuWmxptJZdLJpKRarxBMS2Ju2oANVrgbr2xWbie9b2D")
	require.NoError(t, err)
	pdr := deals.StorageDealRecord{Addr: "a", Time: time.Now().Unix(), Pending: true, DealInfo: deals.StorageDealInfo{ProposalCid: c1}}
	err = s.putPendingDeal(pdr)
	require.NoError(t, err)

	res, err := s.getPendingDeals()
	require.NoError(t, err)
	require.Len(t, res, 1)

	dr := deals.StorageDealRecord{Addr: "a", Time: time.Now().Unix(), Pending: false, DealInfo: deals.StorageDealInfo{ProposalCid: c1}}

	err = s.putFinalDeal(dr)
	require.NoError(t, err)

	res, err = s.getPendingDeals()
	require.NoError(t, err)
	require.Empty(t, res)
}

func TestGetDealRecords(t *testing.T) {
	s := newStore(tests.NewTxMapDatastore())

	now := time.Now().Unix()

	c1, err := util.CidFromString("QmSnuWmxptJZdLJpKRarxBMS2Ju2oANVrgbr2xWbie9b2D")
	require.NoError(t, err)
	dr1 := deals.StorageDealRecord{Addr: "a", Time: now, Pending: false, DealInfo: deals.StorageDealInfo{ProposalCid: c1}}
	err = s.putFinalDeal(dr1)
	require.NoError(t, err)

	c2, err := util.CidFromString("QmSnuWmxptJZdLJpKRarxBMS2Ju2oANVrgbr2xWbie9b2E")
	require.NoError(t, err)
	dr2 := deals.StorageDealRecord{Addr: "b", Time: now + 1, Pending: false, DealInfo: deals.StorageDealInfo{ProposalCid: c2}}
	err = s.putFinalDeal(dr2)
	require.NoError(t, err)

	c3, err := util.CidFromString("QmSnuWmxptJZdLJpKRarxBMS2Ju2oANVrgbr2xWbie9b2F")
	require.NoError(t, err)
	dr3 := deals.StorageDealRecord{Addr: "c", Time: now + 2, Pending: false, DealInfo: deals.StorageDealInfo{ProposalCid: c3}}
	err = s.putFinalDeal(dr3)
	require.NoError(t, err)

	res, err := s.getFinalDeals()
	require.NoError(t, err)
	require.Len(t, res, 3)
}

func TestPutRetrievalRecords(t *testing.T) {
	s := newStore(tests.NewTxMapDatastore())
	now := time.Now().Unix()
	c1, err := util.CidFromString("QmSnuWmxptJZdLJpKRarxBMS2Ju2oANVrgbr2xWbie9b2D")
	require.NoError(t, err)
	rr := deals.RetrievalDealRecord{Time: now, Addr: "from", DealInfo: deals.RetrievalDealInfo{RootCid: c1, Miner: "miner"}}
	err = s.putRetrieval(rr)
	require.NoError(t, err)
}

func TestGetRetrievalDeals(t *testing.T) {
	s := newStore(tests.NewTxMapDatastore())
	now := time.Now().Unix()

	c1, err := util.CidFromString("QmSnuWmxptJZdLJpKRarxBMS2Ju2oANVrgbr2xWbie9b2D")
	require.NoError(t, err)
	rr := deals.RetrievalDealRecord{Time: now, Addr: "from", DealInfo: deals.RetrievalDealInfo{RootCid: c1, Miner: "miner"}}
	err = s.putRetrieval(rr)
	require.NoError(t, err)

	require.NoError(t, err)
	rr = deals.RetrievalDealRecord{Time: now + 1, Addr: "from", DealInfo: deals.RetrievalDealInfo{RootCid: c1, Miner: "miner"}}
	err = s.putRetrieval(rr)
	require.NoError(t, err)

	require.NoError(t, err)
	rr = deals.RetrievalDealRecord{Time: now + 2, Addr: "from", DealInfo: deals.RetrievalDealInfo{RootCid: c1, Miner: "miner"}}
	err = s.putRetrieval(rr)
	require.NoError(t, err)

	res, err := s.getRetrievals()
	require.NoError(t, err)
	require.Len(t, res, 3)
}
