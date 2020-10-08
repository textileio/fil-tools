package sr2

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api/apistruct"
	"github.com/filecoin-project/lotus/chain/types"
	logger "github.com/ipfs/go-log/v2"
	"github.com/textileio/powergate/ffs"
	"github.com/textileio/powergate/lotus"
)

var (
	log = logger.Logger("sr2-miner-selector")
)

// MinerSelector chooses miner under SR2 strategy.
type MinerSelector struct {
	url string
	cb  lotus.ClientBuilder
}

var _ ffs.MinerSelector = (*MinerSelector)(nil)

type minersBuckets struct {
	Buckets []bucket
}

type bucket struct {
	Amount         int
	MinerAddresses []string
}

// New returns a new SR2 miner selector.
func New(url string, cb lotus.ClientBuilder) (*MinerSelector, error) {
	ms := &MinerSelector{url: url, cb: cb}

	_, err := ms.getMiners()
	if err != nil {
		return nil, fmt.Errorf("verifying sr2 url content: %s", err)
	}

	return ms, nil
}

// GetMiners returns miners from SR2.
func (ms *MinerSelector) GetMiners(n int, f ffs.MinerSelectorFilter) ([]ffs.MinerProposal, error) {
	mb, err := ms.getMiners()
	if err != nil {
		return nil, fmt.Errorf("getting miners from url: %s", err)
	}

	c, cls, err := ms.cb()
	if err != nil {
		return nil, fmt.Errorf("creating lotus client: %s", err)
	}
	defer cls()

	rand.Seed(time.Now().UnixNano())
	var selected []ffs.MinerProposal
	for _, bucket := range mb.Buckets {
		miners := bucket.MinerAddresses
		rand.Shuffle(len(miners), func(i, j int) { miners[i], miners[j] = miners[j], miners[i] })

		// Stay safe.
		if bucket.Amount < 0 {
			bucket.Amount = 0
		}
		if bucket.Amount > len(miners) {
			bucket.Amount = len(miners)
		}
		var regionSelected int
		for i := 0; regionSelected < bucket.Amount && i < len(miners); i++ {
			sask, err := getMinerQueryAsk(c, miners[i])
			if err != nil {
				log.Warnf("sr2 miner query-ask errored: %s", miners[i], err)
				continue
			}
			if f.MaxPrice > 0 && sask > f.MaxPrice {
				log.Warnf("skipping miner %s with price % higher than max-price %s", miners[i], sask, f.MaxPrice)
				continue
			}
			selected = append(selected, ffs.MinerProposal{
				Addr:       miners[i],
				EpochPrice: sask,
			})
			regionSelected++
		}
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("no SR2 miners are available")
	}

	return selected, nil
}

// GetReplicationFactor returns the current replication factor of the
// remote configuration.
func (ms *MinerSelector) GetReplicationFactor() (int, error) {
	mb, err := ms.getMiners()
	if err != nil {
		return 0, fmt.Errorf("getting sr2 miners: %s", err)
	}
	var repFactor int
	for _, b := range mb.Buckets {
		repFactor += b.Amount
	}

	return repFactor, nil
}

func (ms *MinerSelector) getMiners() (minersBuckets, error) {
	r, err := http.DefaultClient.Get(ms.url)
	if err != nil {
		return minersBuckets{}, fmt.Errorf("getting miners list from url: %s", err)
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Warnf("closing request body from sr2 file: %s", err)
		}
	}()
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return minersBuckets{}, fmt.Errorf("reading body: %s", err)
	}
	var res minersBuckets
	if err := json.Unmarshal(content, &res); err != nil {
		return minersBuckets{}, fmt.Errorf("unmarshaling url contents: %s", err)
	}
	return res, nil
}

func getMinerQueryAsk(c *apistruct.FullNodeStruct, addrStr string) (uint64, error) {
	addr, err := address.NewFromString(addrStr)
	if err != nil {
		return 0, fmt.Errorf("miner address is invalid: %s", err)
	}
	ctx, cls := context.WithTimeout(context.Background(), time.Second*10)
	defer cls()
	mi, err := c.StateMinerInfo(ctx, addr, types.EmptyTSK)
	if err != nil {
		return 0, fmt.Errorf("getting miner %s info: %s", addr, err)
	}

	sask, err := c.ClientQueryAsk(ctx, *mi.PeerId, addr)
	if err != nil {
		return 0, fmt.Errorf("query asking: %s", err)
	}
	return sask.Price.Uint64(), nil
}
