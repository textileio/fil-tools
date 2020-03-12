package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/textileio/powergate/ffs"
)

var (
	defaultWalletType = "bls"

	log = logging.Logger("ffs-api")
)

var (
	// ErrAlreaadyPinned returned when trying to push an initial config
	// for storing a Cid.
	ErrAlreadyPinned = errors.New("cid already pinned")
	// ErNotStored returned when there isn't CidInfo for a Cid.
	ErrNotStored = errors.New("cid isn't stored")
)

// Instance is an Api instance, which owns a Lotus Address and allows to
// Store and Retrieve Cids from Hot and Cold layers.
type Instance struct {
	store ConfigStore
	wm    ffs.WalletManager

	sched   ffs.Scheduler
	chSched <-chan ffs.Job

	lock     sync.Mutex
	config   Config
	watchers []watcher
	ctx      context.Context
	cancel   context.CancelFunc
	finished chan struct{}
}

type watcher struct {
	jobIDs []ffs.JobID
	ch     chan ffs.Job
}

// New returns a new Api instance.
func New(ctx context.Context, iid ffs.InstanceID, confstore ConfigStore, sch ffs.Scheduler, wm ffs.WalletManager, dc ffs.DefaultCidConfig) (*Instance, error) {
	if err := dc.Validate(); err != nil {
		return nil, fmt.Errorf("default cid config is invalid: %s", err)
	}
	addr, err := wm.NewWallet(ctx, defaultWalletType)
	if err != nil {
		return nil, fmt.Errorf("creating new wallet addr: %s", err)
	}
	config := Config{
		ID:               iid,
		WalletAddr:       addr,
		DefaultCidConfig: dc,
	}
	ctx, cancel := context.WithCancel(context.Background())
	i := new(iid, confstore, wm, config, sch, ctx, cancel)
	if err := i.store.PutInstanceConfig(config); err != nil {
		return nil, fmt.Errorf("saving new instance %s: %s", i.config.ID, err)
	}
	return i, nil
}

// Load loads a saved Api instance from its ConfigStore.
func Load(iid ffs.InstanceID, confstore ConfigStore, sched ffs.Scheduler, wm ffs.WalletManager) (*Instance, error) {
	config, err := confstore.GetInstanceConfig()
	if err != nil {
		return nil, fmt.Errorf("loading instance: %s", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	return new(iid, confstore, wm, *config, sched, ctx, cancel), nil
}

func new(iid ffs.InstanceID, confstore ConfigStore, wm ffs.WalletManager, config Config, sch ffs.Scheduler,
	ctx context.Context, cancel context.CancelFunc) *Instance {
	i := &Instance{
		store:   confstore,
		wm:      wm,
		config:  config,
		sched:   sch,
		chSched: sch.Watch(iid),

		cancel:   cancel,
		ctx:      ctx,
		finished: make(chan struct{}),
	}
	go i.watchJobs()
	return i
}

// ID returns the InstanceID of the instance.
func (i *Instance) ID() ffs.InstanceID {
	return i.config.ID
}

// WalletAddr returns the Lotus wallet address of the instance.
func (i *Instance) WalletAddr() string {
	return i.config.WalletAddr
}

// GetDefaultCidConfig returns the default instance Cid config, prepared for a Cid.
func (i *Instance) GetDefaultCidConfig(c cid.Cid) ffs.CidConfig {
	i.lock.Lock()
	defer i.lock.Unlock()
	return newDefaultAddCidConfig(c, i.config.DefaultCidConfig).Config
}

func (i *Instance) SetDefaultCidConfig(c ffs.DefaultCidConfig) error {
	i.lock.Lock()
	defer i.lock.Unlock()
	if err := c.Validate(); err != nil {
		return fmt.Errorf("default cid config is invalid: %s", err)
	}
	i.config.DefaultCidConfig = c
	return nil
}

// Show returns the information about a stored Cid. If no information is available,
// since the Cid was never stored, it returns ErrNotStore.
func (i *Instance) Show(c cid.Cid) (ffs.CidInfo, error) {
	inf, err := i.store.GetCidInfo(c)
	if err == ErrCidInfoNotFound {
		return inf, ErrNotStored
	}
	if err != nil {
		return inf, fmt.Errorf("getting cid information: %s", err)
	}
	return inf, nil
}

// Info returns instance information
func (i *Instance) Info(ctx context.Context) (InstanceInfo, error) {
	pins, err := i.store.GetCids()
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("getting pins from instance: %s", err)
	}
	balance, err := i.wm.Balance(ctx, i.config.WalletAddr)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("getting balance of %s: %s", i.config.WalletAddr, err)
	}
	return InstanceInfo{
		ID:               i.config.ID,
		DefaultCidConfig: i.config.DefaultCidConfig,
		Wallet: WalletInfo{
			Address: i.config.WalletAddr,
			Balance: balance,
		},
		Pins: pins,
	}, nil
}

// Watch subscribes to Job status changes. If jids is empty, it subscribes to
// all Job status changes corresonding to the instance. If jids is not empty,
// it immediately sends current state of those Jobs. If empty, it doesn't.
func (i *Instance) Watch(jids ...ffs.JobID) (<-chan ffs.Job, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	log.Info("registering watcher")

	var jobs []ffs.Job
	for _, jid := range jids {
		j, err := i.sched.GetJob(jid)
		if err != nil {
			return nil, fmt.Errorf("getting current job state: %s", err)
		}
		jobs = append(jobs, j)
	}

	ch := make(chan ffs.Job, 1)
	i.watchers = append(i.watchers, watcher{jobIDs: jids, ch: ch})
	for _, j := range jobs {
		select {
		case ch <- j:
		default:
			log.Warnf("dropped notifying current job state on slow receiver on %s", i.config.ID)
		}
	}

	return ch, nil
}

// Unwatch unregisters a ch returned by Watch to stop receiving updates.
func (i *Instance) Unwatch(ch <-chan ffs.Job) {
	i.lock.Lock()
	defer i.lock.Unlock()
	for j, w := range i.watchers {
		if w.ch == ch {
			close(w.ch)
			i.watchers[j] = i.watchers[len(i.watchers)-1]
			i.watchers = i.watchers[:len(i.watchers)-1]
			return
		}
	}
}

// AddCid push a new default configuration for the Cid in the Hot and
// Cold layer. (TODO: Soon the configuration will be received as a param,
// to allow different strategies in the Hot and Cold layer. Now a second AddCid
// will error with ErrAlreadyPinned. This might change depending if changing config
// for an existing Cid will use this same API, or another. In any case sounds safer to
// consider some option to specify we want to add a config without overriding some
// existing one.)
func (i *Instance) AddCid(c cid.Cid, opts ...AddCidOption) (ffs.JobID, error) {
	i.lock.Lock()
	defer i.lock.Unlock()

	addConfig := newDefaultAddCidConfig(c, i.config.DefaultCidConfig)
	for _, opt := range opts {
		if err := opt(&addConfig); err != nil {
			return ffs.EmptyJobID, fmt.Errorf("config option: %s", err)
		}
	}

	if !addConfig.OverrideConfig {
		_, err := i.store.GetCidConfig(c)
		if err == nil {
			return ffs.EmptyJobID, ErrAlreadyPinned
		}
		if err != ErrConfigNotFound {
			return ffs.EmptyJobID, fmt.Errorf("getting cid config: %s", err)
		}
	}

	if err := addConfig.Config.Validate(); err != nil {
		return ffs.EmptyJobID, err
	}

	addConf := ffs.AddAction{
		InstanceID: i.config.ID,
		ID:         ffs.NewCidConfigID(),
		Config:     addConfig.Config,
		Meta: ffs.AddMeta{
			WalletAddr: i.config.WalletAddr,
		},
	}
	log.Infof("adding cid %s to scheduler queue", c)
	jid, err := i.sched.EnqueueCid(addConf)
	if err != nil {
		return ffs.EmptyJobID, fmt.Errorf("scheduling cid %s: %s", c, err)
	}
	if err := i.store.PutCidConfig(addConf.Config); err != nil {
		return ffs.EmptyJobID, fmt.Errorf("saving new config for cid %s: %s", c, err)
	}
	return jid, nil
}

// Get returns an io.Reader for reading a stored Cid from the Hot Storage.
// (TODO: Scheduler.GetFromHot might have to return an error if we want to rate-limit
// hot layer retrievals)
func (i *Instance) Get(ctx context.Context, c cid.Cid) (io.Reader, error) {
	if !c.Defined() {
		return nil, fmt.Errorf("cid is undefined")
	}
	conf, err := i.store.GetCidConfig(c)
	if err != nil {
		return nil, fmt.Errorf("getting cid config: %s", err)
	}
	if !conf.Hot.Enabled {
		return nil, ffs.ErrHotStorageDisabled
	}
	r, err := i.sched.GetCidFromHot(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("getting from hot layer %s: %s", c, err)
	}
	return r, nil
}

// Close terminates the running Instance.
func (i *Instance) Close() error {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.cancel()
	<-i.finished
	i.sched.Unwatch(i.chSched)
	return nil
}

func (i *Instance) watchJobs() {
	defer close(i.finished)
	for {
		select {
		case <-i.ctx.Done():
			log.Infof("terminating job watching in %s", i.config.ID)
			return
		case j, ok := <-i.chSched:
			if !ok {
				panic("scheduler closed the watching channel")
			}
			log.Info("received notification from jobstore")
			if err := i.store.PutCidInfo(j.CidInfo); err != nil {
				log.Errorf("saving cid info %s: %s", j.CidInfo.Cid, err)
				continue
			}
			i.lock.Lock()
			log.Infof("notifying %d subscribed watchers", len(i.watchers))
			for k, w := range i.watchers {
				shouldNotify := len(w.jobIDs) == 0
				for _, jid := range w.jobIDs {
					if jid == j.ID {
						shouldNotify = true
						break
					}
				}
				log.Infof("evaluating watcher %d, shouldNotify %s", k, shouldNotify)
				if shouldNotify {
					select {
					case w.ch <- j:
						log.Info("notifying watcher")
					default:
						log.Warnf("skipping slow api watcher")
					}
				}
			}
			i.lock.Unlock()
		}
	}
}
