package filcold

import (
	"context"
	"fmt"
	"io"

	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/ipfs/go-car"
	"github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"
	logger "github.com/ipfs/go-log/v2"
	"github.com/textileio/fil-tools/deals"
	"github.com/textileio/fil-tools/fpa"
)

var (
	log = logger.Logger("fpa-filcold")
)

// FilCold is a fpa.ColdLayer implementation that stores data in Filecoin.
type FilCold struct {
	ms  fpa.MinerSelector
	dm  *deals.Module
	dag format.DAGService
}

var _ fpa.ColdLayer = (*FilCold)(nil)

// New returns a new FilCold instance
func New(ms fpa.MinerSelector, dm *deals.Module, dag format.DAGService) *FilCold {
	return &FilCold{
		ms:  ms,
		dm:  dm,
		dag: dag,
	}
}

// Store stores a Cid in Filecoin considering the configuration provided. The Cid is retrieved using
// the DAGService registered on instance creation. Currently, a default configuration is used.
// (TODO: ColdConfig will enable more configurations in the future)
func (fc *FilCold) Store(ctx context.Context, c cid.Cid, conf fpa.ColdConfig) (fpa.ColdInfo, error) {
	var ci fpa.ColdInfo
	ci.Filecoin = fpa.FilInfo{
		Duration: uint64(1000),
	}
	config, err := makeStorageConfig(ctx, fc.ms)
	if err != nil {
		return ci, fmt.Errorf("selecting miners to make the deal: %s", err)
	}
	r := ipldToFileTransform(ctx, fc.dag, c)
	log.Infof("storing deals in filecoin...")
	dataCid, res, err := fc.dm.Store(ctx, conf.Filecoin.WalletAddr, r, config, ci.Filecoin.Duration)
	if err != nil {
		return ci, fmt.Errorf("storing deals in deal manager: %s", err)
	}
	ci.Filecoin.PayloadCID = dataCid

	if ci.Filecoin.Proposals, err = fc.waitForDeals(ctx, res); err != nil {
		return ci, fmt.Errorf("waiting for deals to finish: %s", err)
	}
	return ci, nil
}

func (fc *FilCold) waitForDeals(ctx context.Context, res []deals.StoreResult) ([]fpa.FilStorage, error) {
	notDone := make(map[cid.Cid]struct{})
	var propcids []cid.Cid
	var filstrg []fpa.FilStorage
	for _, d := range res {
		if d.Success {
			filstrg = append(filstrg, fpa.FilStorage{
				ProposalCid: d.ProposalCid,
				Failed:      !d.Success,
			})
			propcids = append(propcids, d.ProposalCid)
			notDone[d.ProposalCid] = struct{}{}
		}
	}

	log.Infof("watching deals unfold...")
	chDi, err := fc.dm.Watch(ctx, propcids)
	if err != nil {
		return nil, err
	}
	for di := range chDi {
		// ToDo: check state coverage, changes return since deals can fail
		if di.StateID == storagemarket.StorageDealActive {
			delete(notDone, di.ProposalCid)
		}
		if len(notDone) == 0 {
			break
		}
	}
	log.Infof("done")
	return filstrg, nil
}

func ipldToFileTransform(ctx context.Context, dag format.DAGService, c cid.Cid) io.Reader {
	r, w := io.Pipe()
	go func() {
		if err := car.WriteCar(ctx, dag, []cid.Cid{c}, w); err != nil {
			w.CloseWithError(err)
		}
		w.Close()
	}()
	return r
}

func makeStorageConfig(ctx context.Context, ms fpa.MinerSelector) ([]deals.StorageDealConfig, error) {
	mps, err := ms.GetTopMiners(1) // ToDo: hardcoded 1 will change when config will be added to instance/method-call
	if err != nil {
		return nil, err
	}
	res := make([]deals.StorageDealConfig, len(mps))
	for i, m := range mps {
		res[i] = deals.StorageDealConfig{Miner: m.Addr, EpochPrice: m.EpochPrice}
	}
	return res, nil
}
