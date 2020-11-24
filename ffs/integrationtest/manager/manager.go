package manager

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api/apistruct"
	"github.com/ipfs/go-datastore"
	"github.com/stretchr/testify/require"
	"github.com/textileio/powergate/ffs"
	"github.com/textileio/powergate/ffs/api"
	"github.com/textileio/powergate/ffs/coreipfs"
	"github.com/textileio/powergate/ffs/filcold"
	"github.com/textileio/powergate/ffs/joblogger"
	"github.com/textileio/powergate/ffs/manager"
	"github.com/textileio/powergate/ffs/minerselector/fixed"
	"github.com/textileio/powergate/ffs/scheduler"
	"github.com/textileio/powergate/filchain"
	"github.com/textileio/powergate/lotus"
	"github.com/textileio/powergate/tests"
	"github.com/textileio/powergate/util"

	httpapi "github.com/ipfs/go-ipfs-http-client"
	dealsModule "github.com/textileio/powergate/deals/module"
	it "github.com/textileio/powergate/ffs/integrationtest"
	txndstr "github.com/textileio/powergate/txndstransform"
	walletModule "github.com/textileio/powergate/wallet/module"
)

const (
	iWalletBal int64 = 4000000000000000
)

// NewAPI returns a new set of components for FFS.
func NewAPI(t tests.TestingTWithCleanup, numMiners int) (*httpapi.HttpApi, *apistruct.FullNodeStruct, *api.API, func()) {
	ds := tests.NewTxMapDatastore()
	ipfs, ipfsMAddr := it.CreateIPFS(t)
	addr, clientBuilder, ms := NewDevnet(t, numMiners, ipfsMAddr)
	manager, closeManager := NewFFSManager(t, ds, clientBuilder, addr, ms, ipfs)
	auth, err := manager.Create(context.Background())
	require.NoError(t, err)
	time.Sleep(time.Second * 3) // Wait for funding txn to finish.
	fapi, err := manager.GetByAuthToken(auth.Token)
	require.NoError(t, err)
	client, cls, err := clientBuilder(context.Background())
	require.NoError(t, err)
	return ipfs, client, fapi, func() {
		err := fapi.Close()
		require.NoError(t, err)
		closeManager()
		cls()
	}
}

// NewDevnet creates a localnet.
func NewDevnet(t tests.TestingTWithCleanup, numMiners int, ipfsAddr string) (address.Address, lotus.ClientBuilder, ffs.MinerSelector) {
	client, addr, _ := tests.CreateLocalDevnetWithIPFS(t, numMiners, ipfsAddr, false)
	addrs := make([]string, numMiners)
	for i := 0; i < numMiners; i++ {
		addrs[i] = fmt.Sprintf("f0%d", 1000+i)
	}

	fixedMiners := make([]fixed.Miner, len(addrs))
	for i, a := range addrs {
		fixedMiners[i] = fixed.Miner{Addr: a, Country: "China", EpochPrice: 500000000}
	}
	ms := fixed.New(fixedMiners)
	return addr, client, ms
}

// NewFFSManager returns a new FFS manager.
func NewFFSManager(t require.TestingT, ds datastore.TxnDatastore, clientBuilder lotus.ClientBuilder, masterAddr address.Address, ms ffs.MinerSelector, ipfsClient *httpapi.HttpApi) (*manager.Manager, func()) {
	mg, _, err := NewCustomFFSManager(t, ds, clientBuilder, masterAddr, ms, ipfsClient, 0)
	return mg, err
}

// NewCustomFFSManager returns a new customized FFS manager.
func NewCustomFFSManager(t require.TestingT, ds datastore.TxnDatastore, cb lotus.ClientBuilder, masterAddr address.Address, ms ffs.MinerSelector, ipfsClient *httpapi.HttpApi, minimumPieceSize uint64) (*manager.Manager, *coreipfs.CoreIpfs, func()) {
	dm, err := dealsModule.New(txndstr.Wrap(ds, "deals"), cb, util.AvgBlockTime, time.Minute*10)
	require.NoError(t, err)

	fchain := filchain.New(cb)
	l := joblogger.New(txndstr.Wrap(ds, "ffs/joblogger"))
	lsm, err := lotus.NewSyncMonitor(cb)
	require.NoError(t, err)
	cl := filcold.New(ms, dm, ipfsClient, fchain, l, lsm, minimumPieceSize, 1)
	hl, err := coreipfs.New(ds, ipfsClient, l)
	require.NoError(t, err)
	sched, err := scheduler.New(txndstr.Wrap(ds, "ffs/scheduler"), l, hl, cl, 10, time.Minute*10, nil, scheduler.GCConfig{AutoGCInterval: 0})
	require.NoError(t, err)

	wm, err := walletModule.New(cb, masterAddr, *big.NewInt(iWalletBal), false, "")
	require.NoError(t, err)

	manager, err := manager.New(ds, wm, dm, sched, false, true)
	require.NoError(t, err)
	err = manager.SetDefaultStorageConfig(ffs.StorageConfig{
		Hot: ffs.HotConfig{
			Enabled:       true,
			AllowUnfreeze: false,
			Ipfs: ffs.IpfsConfig{
				AddTimeout: 10,
			},
		},
		Cold: ffs.ColdConfig{
			Enabled: true,
			Filecoin: ffs.FilConfig{
				ExcludedMiners:  nil,
				DealMinDuration: util.MinDealDuration,
				RepFactor:       1,
			},
		},
	})
	require.NoError(t, err)

	return manager, hl, func() {
		if err := manager.Close(); err != nil {
			t.Errorf("closing api: %s", err)
			t.FailNow()
		}
		if err := sched.Close(); err != nil {
			t.Errorf("closing scheduler: %s", err)
			t.FailNow()
		}
		if err := l.Close(); err != nil {
			t.Errorf("closing joblogger: %s", err)
			t.FailNow()
		}
	}
}
