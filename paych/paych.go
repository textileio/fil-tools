package paych

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/apistruct"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin/paych"
	"github.com/ipfs/go-cid"
	"github.com/textileio/powergate/ffs"
)

// Module provides access to the paych api
type Module struct {
	api *apistruct.FullNodeStruct
}

var _ ffs.PaychManager = (*Module)(nil)

// New creates a new paych module
func New(api *apistruct.FullNodeStruct) (*Module, error) {
	return &Module{
		api: api,
	}, nil
}

// List lists all payment channels involving the specified addresses
func (m *Module) List(ctx context.Context, addrs ...address.Address) ([]ffs.PaychInfo, error) {
	filter := make(map[string]struct{}, len(addrs))
	for _, addr := range addrs {
		filter[addr.String()] = struct{}{}
	}

	allAddrs, err := m.api.PaychList(ctx)
	if err != nil {
		return nil, err
	}

	chans := make([]<-chan statusResult, len(allAddrs))
	for i, addr := range allAddrs {
		chans[i] = m.paychStatus(ctx, addr)
	}

	resultsCh := make(chan statusResult, len(chans))
	for _, c := range chans {
		go func(ch <-chan statusResult) {
			res := <-ch
			resultsCh <- res
		}(c)
	}

	results := make([]statusResult, len(chans))
	for i := 0; i < len(chans); i++ {
		results[i] = <-resultsCh
	}

	var final []ffs.PaychInfo
	for _, result := range results {
		if result.err != nil {
			// ToDo: do we want to fail if there was an error checking
			// even one status that may not even be in our filter?
			continue
		}
		_, addrInFilter := filter[result.addr.String()]
		_, ctlAddrInFilter := filter[result.status.ControlAddr.String()]
		if addrInFilter || ctlAddrInFilter {
			var dir ffs.PaychDir
			switch result.status.Direction {
			case api.PCHUndef:
				dir = ffs.PaychDirUndef
			case api.PCHInbound:
				dir = ffs.PaychDirInbound
			case api.PCHOutbound:
				dir = ffs.PaychDirOutbound
			default:
				return nil, fmt.Errorf("unknown pay channel direction %v", result.status.Direction)
			}

			info := ffs.PaychInfo{
				CtlAddr:   result.status.ControlAddr,
				Addr:      result.addr,
				Direction: dir,
			}

			final = append(final, info)
		}
	}

	return final, nil
}

// Create creates a new payment channel
func (m *Module) Create(ctx context.Context, from address.Address, to address.Address, amount uint64) (ffs.PaychInfo, cid.Cid, error) {
	a := types.NewInt(amount)
	info, err := m.api.PaychGet(ctx, from, to, a)
	if err != nil {
		return ffs.PaychInfo{}, cid.Undef, err
	}
	// ToDo: verify these addresses and direction make sense
	res := ffs.PaychInfo{
		CtlAddr:   from,
		Addr:      info.Channel,
		Direction: ffs.PaychDirOutbound,
	}
	return res, info.ChannelMessage, nil
}

// Redeem redeems a payment channel
func (m *Module) Redeem(ctx context.Context, ch address.Address) error {
	return nil
}

// CreateVoucher creates a pay channel voucher
func (m *Module) CreateVoucher(ctx context.Context, addr address.Address, amt uint64, opts ...ffs.CreateVoucherOption) (*paych.SignedVoucher, error) {
	return nil, nil
}

type statusResult struct {
	addr   address.Address
	status *api.PaychStatus
	err    error
}

func (m *Module) paychStatus(ctx context.Context, addr address.Address) <-chan statusResult {
	c := make(chan statusResult)
	go func() {
		defer close(c)
		status, err := m.api.PaychStatus(ctx, addr)
		c <- statusResult{addr: addr, status: status, err: err}
	}()
	return c
}
