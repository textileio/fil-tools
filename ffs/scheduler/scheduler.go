package scheduler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	"github.com/textileio/powergate/ffs"
	"github.com/textileio/powergate/ffs/scheduler/internal/astore"
	"github.com/textileio/powergate/ffs/scheduler/internal/cistore"
	"github.com/textileio/powergate/ffs/scheduler/internal/jstore"
	txndstr "github.com/textileio/powergate/txndstransform"
	"github.com/textileio/powergate/util"
)

const (
	maxParallelExecutions = 50
)

var (
	log = logging.Logger("ffs-scheduler")

	// ErrNotFound is returned when an item isn't found on a Store.
	ErrNotFound = errors.New("item not found")
)

// Scheduler receives actions to store a Cid in Hot and Cold layers. These actions are
// created as Jobs which have a lifecycle that can be watched by external actors.
// This Jobs are executed by delegating the work to the Hot and Cold layers configured for
// the scheduler.
type Scheduler struct {
	cs  ffs.ColdStorage
	hs  ffs.HotStorage
	js  *jstore.Store
	as  *astore.Store
	cis *cistore.Store
	l   ffs.CidLogger

	rateLim            chan struct{}
	evaluateQueuedWork chan struct{}

	ctx      context.Context
	cancel   context.CancelFunc
	finished chan struct{}
}

// New returns a new instance of Scheduler which uses JobStore as its backing repository for state,
// HotStorage for the hot layer, and ColdStorage for the cold layer.
func New(ds datastore.TxnDatastore, l ffs.CidLogger, hs ffs.HotStorage, cs ffs.ColdStorage) (*Scheduler, error) {
	js, err := jstore.New(txndstr.Wrap(ds, "jstore"))
	if err != nil {
		return nil, fmt.Errorf("loading scheduler jobstore: %s", err)
	}
	as := astore.New(txndstr.Wrap(ds, "astore"))
	cis := cistore.New(txndstr.Wrap(ds, "cistore"))
	ctx, cancel := context.WithCancel(context.Background())
	sch := &Scheduler{
		cs:  cs,
		hs:  hs,
		js:  js,
		as:  as,
		cis: cis,
		l:   l,

		rateLim:            make(chan struct{}, maxParallelExecutions),
		evaluateQueuedWork: make(chan struct{}, 1),

		ctx:      ctx,
		cancel:   cancel,
		finished: make(chan struct{}),
	}
	go sch.run()
	return sch, nil
}

// PushConfig queues the specified CidConfig to be executed as a new Job. It returns
// the created JobID for further tracking of its state.
func (s *Scheduler) PushConfig(iid ffs.APIID, cfg ffs.CidConfig) (ffs.JobID, error) {
	return s.push(iid, cfg, cid.Undef)
}

// PushReplace queues a new CidConfig to be executed as a new Job, replacing an oldCid that will be
// untrack in the Scheduler (i.e: deal renewals, repairing).
func (s *Scheduler) PushReplace(iid ffs.APIID, cfg ffs.CidConfig, oldCid cid.Cid) (ffs.JobID, error) {
	if !oldCid.Defined() {
		return ffs.EmptyJobID, fmt.Errorf("cid can't be undefined")
	}
	return s.push(iid, cfg, oldCid)
}

func (s *Scheduler) push(iid ffs.APIID, cfg ffs.CidConfig, oldCid cid.Cid) (ffs.JobID, error) {
	if !cfg.Cid.Defined() {
		return ffs.EmptyJobID, fmt.Errorf("cid can't be undefined")
	}
	if iid == ffs.EmptyInstanceID {
		return ffs.EmptyJobID, fmt.Errorf("invalid Action ID")
	}
	if err := cfg.Validate(); err != nil {
		return ffs.EmptyJobID, fmt.Errorf("validating cid config: %s", err)
	}
	jid := ffs.NewJobID()
	j := ffs.Job{
		ID:     jid,
		APIID:  iid,
		Cid:    cfg.Cid,
		Status: ffs.Queued,
	}

	if err := s.js.Enqueue(j); err != nil {
		return ffs.EmptyJobID, fmt.Errorf("enqueuing job: %s", err)
	}
	ctx := context.WithValue(context.Background(), ffs.CtxKeyJid, jid)
	s.l.Log(ctx, cfg.Cid, "Pushing new configuration...")

	aa := astore.Action{
		APIID:       iid,
		Cfg:         cfg,
		ReplacedCid: oldCid,
	}
	if err := s.as.Put(j.ID, aa); err != nil {
		return ffs.EmptyJobID, fmt.Errorf("saving new config in store: %s", err)
	}

	if oldCid.Defined() {
		if err := s.Untrack(oldCid); err != nil {
			return ffs.EmptyJobID, fmt.Errorf("untracking replaced cid: %s", err)
		}
	}
	select {
	case s.evaluateQueuedWork <- struct{}{}:
	default:
	}

	s.l.Log(ctx, cfg.Cid, "Configuration saved successfully")
	return jid, nil
}

// Untrack untracks a Cid for renewal and repair background crons.
func (s *Scheduler) Untrack(c cid.Cid) error {
	if err := s.as.Remove(c); err != nil {
		return fmt.Errorf("removing cid from action store: %s", err)
	}
	return nil
}

// GetCidInfo returns the current storage state of a Cid. Returns ErrNotFound
// if there isn't information for a Cid.
func (s *Scheduler) GetCidInfo(c cid.Cid) (ffs.CidInfo, error) {
	info, err := s.cis.Get(c)
	if err == cistore.ErrNotFound {
		return ffs.CidInfo{}, ErrNotFound
	}
	if err != nil {
		return ffs.CidInfo{}, fmt.Errorf("getting CidInfo from store: %s", err)
	}
	return info, nil
}

// GetCidFromHot returns an io.Reader of the data from the hot layer.
func (s *Scheduler) GetCidFromHot(ctx context.Context, c cid.Cid) (io.Reader, error) {
	r, err := s.hs.Get(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("getting %s from hot layer: %s", c, err)
	}
	return r, nil
}

// GetJob the current state of a Job.
func (s *Scheduler) GetJob(jid ffs.JobID) (ffs.Job, error) {
	j, err := s.js.Get(jid)
	if err != nil {
		if err == jstore.ErrNotFound {
			return ffs.Job{}, ErrNotFound
		}
		return ffs.Job{}, fmt.Errorf("get Job from store: %s", err)
	}
	return j, nil
}

// WatchJobs returns a channel to listen to Job status changes from a specified
// API instance. It immediately pushes the current Job state to the channel.
func (s *Scheduler) WatchJobs(ctx context.Context, c chan<- ffs.Job, iid ffs.APIID) error {
	return s.js.Watch(ctx, c, iid)
}

// WatchLogs writes to a channel all new logs for Cids. The context should be
// canceled when wanting to stop receiving updates to the channel.
func (s *Scheduler) WatchLogs(ctx context.Context, c chan<- ffs.LogEntry) error {
	return s.l.Watch(ctx, c)
}

// GetLogs returns history logs of a Cid.
func (s *Scheduler) GetLogs(ctx context.Context, c cid.Cid) ([]ffs.LogEntry, error) {
	lgs, err := s.l.Get(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("getting logs: %s", err)
	}
	return lgs, nil
}

// Close terminates the scheduler.
func (s *Scheduler) Close() error {
	s.cancel()
	<-s.finished
	if err := s.js.Close(); err != nil {
		return fmt.Errorf("closing jobstore: %s", err)
	}
	return nil
}

func (s *Scheduler) run() {
	defer close(s.finished)
	if err := s.resumeStartedDeals(); err != nil {
		log.Errorf("resuming started deals: %s", err)
		return
	}
	for {
		select {
		case <-s.ctx.Done():
			log.Infof("terminating scheduler daemon")
			return
		case <-time.After(util.AvgBlockTime):
			log.Debug("running renewal checks...")
			s.execRenewCron(s.ctx)
			log.Debug("renewal cron done")
			log.Debug("running repair checks...")
			s.execRepairCron(s.ctx)
			log.Debug("repair cron done")
		case <-s.evaluateQueuedWork:
			log.Debug("evaluating Job queue execution...")
			s.executeQueuedJobs(s.ctx)
			log.Debug("evaluation Job queue execution...")
		}
	}
}

func (s *Scheduler) resumeStartedDeals() error {
	ejids := s.js.GetExecutingJobs()
	// No need for rate limit since "Executing" # of jobs are already rate limited on creation.
	var wg sync.WaitGroup
	for _, jid := range ejids {
		if s.ctx.Err() != nil {
			break
		}
		j, err := s.js.Get(jid)
		if err != nil {
			return fmt.Errorf("getting resumed queued job: %s", err)
		}
		wg.Add(1)
		go func(j ffs.Job) {
			defer wg.Done()
			// We re-execute the pipeline as if was dequeued.
			// Both hot and cold storage can detect resumed job execution.
			s.executeQueuedJob(j)
		}(j)
	}
	wg.Wait()
	return nil
}

func (s *Scheduler) execRepairCron(ctx context.Context) {
	as, err := s.as.GetRepairable()
	if err != nil {
		log.Errorf("getting repairable cid configs from store: %s", err)
	}
	for _, a := range as {
		log.Debugf("scheduling deal repair for Cid %s", a.Cfg.Cid)
		if err := s.scheduleRepairJob(ctx, a); err != nil {
			log.Errorf("repair of %s: %s", a.Cfg.Cid, err)
		}
		log.Debugf("scheduling repair done")
	}
}

func (s *Scheduler) scheduleRepairJob(ctx context.Context, a astore.Action) error {
	s.l.Log(ctx, a.Cfg.Cid, "Scheduling deal repair...")
	a.Cfg.Repairable = false
	jid, err := s.push(a.APIID, a.Cfg, cid.Undef)
	if err != nil {
		return fmt.Errorf("scheduling repair job: %s", err)
	}
	s.l.Log(ctx, a.Cfg.Cid, "Job %s was queued for repair evaluation.", jid)
	return nil
}

func (s *Scheduler) execRenewCron(ctx context.Context) {
	as, err := s.as.GetRenewable()
	if err != nil {
		log.Errorf("getting renweable cid configs from store: %s", err)
	}
	for _, a := range as {
		log.Debugf("evaluating deal renewal for Cid %s", a.Cfg.Cid)
		if err := s.evaluateRenewal(ctx, a); err != nil {
			log.Errorf("renweal of %s: %s", a.Cfg.Cid, err)
		}
		log.Debugf("deal renewal done")
	}
}

func (s *Scheduler) evaluateRenewal(ctx context.Context, a astore.Action) error {
	inf, err := s.getRefreshedInfo(ctx, a.Cfg.Cid)
	if err == ErrNotFound {
		log.Infof("skip renewal evaluation for %s since Cid isn't stored yet", a.Cfg.Cid)
		return nil
	}
	if err != nil {
		return fmt.Errorf("getting cid info from store: %s", err)
	}
	s.l.Log(ctx, a.Cfg.Cid, "Evaluating deal renweal...")

	var errors []ffs.DealError
	inf.Cold.Filecoin, errors, err = s.cs.EnsureRenewals(ctx, a.Cfg.Cid, inf.Cold.Filecoin, a.Cfg.Cold.Filecoin)
	for _, e := range errors {
		log.Warnf("renew deal error: %s %s %s", e.ProposalCid, e.Miner, e.Message)
	}
	if err != nil {
		return fmt.Errorf("evaluating renewal in cold-storage: %s", err)
	}
	if err := s.cis.Put(inf); err != nil {
		return fmt.Errorf("saving new cid info in store: %s", err)
	}

	s.l.Log(ctx, a.Cfg.Cid, "Deal renewal evaluated successfully")
	return nil
}

func (s *Scheduler) executeQueuedJobs(ctx context.Context) {
	var err error
	var j *ffs.Job

forLoop:
	for {
		select {
		case <-ctx.Done():
			break forLoop
		case s.rateLim <- struct{}{}:
			// We have a slot, try pushing queued things.
		default:
			// If the execution pipeline is full, we can't
			// add new things as Executing.
			break forLoop
		}

		j, err = s.js.Dequeue()
		if err != nil {
			log.Errorf("getting queued jobs: %s", err)
			<-s.rateLim
			return
		}
		if j == nil {
			// Make the slot available again.
			<-s.rateLim
			break
		}
		go func(j ffs.Job) {
			s.executeQueuedJob(j)
			<-s.rateLim

			// Signal that there's a new open slot, and
			// that other blocked Jobs with the same Cid
			// can be executed.
			select {
			case s.evaluateQueuedWork <- struct{}{}:
			default:
			}
		}(*j)
	}
}

func (s *Scheduler) executeQueuedJob(j ffs.Job) {
	a, err := s.as.Get(j.ID)
	if err != nil {
		log.Errorf("getting push config action data from store: %s", err)
		return
	}

	ctx := context.WithValue(s.ctx, ffs.CtxKeyJid, j.ID)
	s.l.Log(ctx, a.Cfg.Cid, "Executing job %s...", j.ID)
	info, dealErrors, err := s.execute(ctx, a, j)
	if err != nil {
		log.Errorf("executing job %s: %s", j.ID, err)
		if err := s.js.Finalize(j.ID, ffs.Failed, err, dealErrors); err != nil {
			log.Errorf("changing job to failed: %s", err)
		}
		s.l.Log(ctx, a.Cfg.Cid, "Job %s execution failed.", j.ID)
		return
	}
	if err := s.cis.Put(info); err != nil {
		log.Errorf("saving cid info to store: %s", err)
	}
	if err := s.js.Finalize(j.ID, ffs.Success, nil, dealErrors); err != nil {
		log.Errorf("changing job to success: %s", err)
	}
	s.l.Log(ctx, a.Cfg.Cid, "Job %s execution finished successfully.", j.ID)
}

func (s *Scheduler) execute(ctx context.Context, a astore.Action, job ffs.Job) (ffs.CidInfo, []ffs.DealError, error) {
	ci, err := s.getRefreshedInfo(ctx, a.Cfg.Cid)
	if err != nil {
		return ffs.CidInfo{}, nil, fmt.Errorf("getting current cid info from store: %s", err)
	}

	s.l.Log(ctx, a.Cfg.Cid, "Ensuring Hot-Storage satisfies the configuration...")
	hot, err := s.executeHotStorage(ctx, ci, a.Cfg.Hot, a.Cfg.Cold.Filecoin.Addr, a.ReplacedCid)
	if err != nil {
		s.l.Log(ctx, a.Cfg.Cid, "Hot-Storage excution failed.")
		return ffs.CidInfo{}, nil, fmt.Errorf("executing hot-storage config: %s", err)
	}
	s.l.Log(ctx, a.Cfg.Cid, "Hot-Storage execution ran successfully.")

	s.l.Log(ctx, a.Cfg.Cid, "Ensuring Cold-Storage satisfies the configuration...")
	cold, errors, err := s.executeColdStorage(ctx, ci, a.Cfg.Cold)
	if err != nil {
		s.l.Log(ctx, a.Cfg.Cid, "Cold-Storage execution failed.")
		return ffs.CidInfo{}, errors, fmt.Errorf("executing cold-storage config: %s", err)
	}
	s.l.Log(ctx, a.Cfg.Cid, "Cold-Storage execution ran successfully.")

	return ffs.CidInfo{
		JobID:   job.ID,
		Cid:     a.Cfg.Cid,
		Hot:     hot,
		Cold:    cold,
		Created: time.Now(),
	}, errors, nil
}

func (s *Scheduler) executeHotStorage(ctx context.Context, curr ffs.CidInfo, cfg ffs.HotConfig, waddr string, replaceCid cid.Cid) (ffs.HotInfo, error) {
	if cfg.Enabled == curr.Hot.Enabled {
		s.l.Log(ctx, curr.Cid, "Current Cid state is healthy in Hot-Storage.")
		return curr.Hot, nil
	}

	if !cfg.Enabled {
		if err := s.hs.Remove(ctx, curr.Cid); err != nil {
			return ffs.HotInfo{}, fmt.Errorf("removing from hot storage: %s", err)
		}
		s.l.Log(ctx, curr.Cid, "Cid successfully removed from Hot-Storage.")
		return ffs.HotInfo{Enabled: false}, nil
	}

	sctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(cfg.Ipfs.AddTimeout))
	defer cancel()

	var size int
	var err error
	if !replaceCid.Defined() {
		size, err = s.hs.Store(sctx, curr.Cid)
	} else {
		s.l.Log(ctx, curr.Cid, "Replace of previous pin %s", replaceCid)
		size, err = s.hs.Replace(sctx, replaceCid, curr.Cid)
	}
	if err != nil {
		s.l.Log(ctx, curr.Cid, "Direct fetching from IPFS wasn't possible.")
		if !cfg.AllowUnfreeze || len(curr.Cold.Filecoin.Proposals) == 0 {
			s.l.Log(ctx, curr.Cid, "Unfreeze is disabled or active Filecoin deals are unavailable.")
			return ffs.HotInfo{}, fmt.Errorf("pinning cid in hot storage: %s", err)
		}
		s.l.Log(ctx, curr.Cid, "Unfreezing from Filecoin...")
		if err := s.cs.Fetch(ctx, curr.Cold.Filecoin.DataCid, waddr); err != nil {
			return ffs.HotInfo{}, fmt.Errorf("unfreezing from Cold Storage: %s", err)
		}
		s.l.Log(ctx, curr.Cid, "Unfrozen successfully, saving in Hot-Storage...")
		size, err = s.hs.Store(ctx, curr.Cold.Filecoin.DataCid)
		if err != nil {
			return ffs.HotInfo{}, fmt.Errorf("pinning unfrozen cid: %s", err)
		}
	}
	return ffs.HotInfo{
		Enabled: true,
		Size:    size,
		Ipfs: ffs.IpfsHotInfo{
			Created: time.Now(),
		},
	}, nil
}

func (s *Scheduler) getRefreshedInfo(ctx context.Context, c cid.Cid) (ffs.CidInfo, error) {
	var err error
	ci, err := s.cis.Get(c)
	if err != nil {
		if err != cistore.ErrNotFound {
			return ffs.CidInfo{}, ErrNotFound
		}
		return ffs.CidInfo{Cid: c}, nil // Default value has both storages disabled
	}

	ci.Hot, err = s.getRefreshedHotInfo(ctx, c, ci.Hot)
	if err != nil {
		return ffs.CidInfo{}, fmt.Errorf("getting refreshed hot info: %s", err)
	}

	ci.Cold, err = s.getRefreshedColdInfo(ctx, ci.Cold)
	if err != nil {
		return ffs.CidInfo{}, fmt.Errorf("getting refreshed cold info: %s", err)
	}

	return ci, nil
}

func (s *Scheduler) getRefreshedHotInfo(ctx context.Context, c cid.Cid, curr ffs.HotInfo) (ffs.HotInfo, error) {
	var err error
	curr.Enabled, err = s.hs.IsStored(ctx, c)
	if err != nil {
		return ffs.HotInfo{}, err
	}
	return curr, nil
}

func (s *Scheduler) getRefreshedColdInfo(ctx context.Context, curr ffs.ColdInfo) (ffs.ColdInfo, error) {
	activeDeals := make([]ffs.FilStorage, 0, len(curr.Filecoin.Proposals))
	for _, fp := range curr.Filecoin.Proposals {
		active, err := s.cs.IsFilDealActive(ctx, fp.ProposalCid)
		if err != nil {
			return ffs.ColdInfo{}, fmt.Errorf("getting deal state of proposal %s: %s", fp.ProposalCid, err)
		}
		if active {
			activeDeals = append(activeDeals, fp)
		}
	}
	curr.Filecoin.Proposals = activeDeals
	return curr, nil
}

func (s *Scheduler) executeColdStorage(ctx context.Context, curr ffs.CidInfo, cfg ffs.ColdConfig) (ffs.ColdInfo, []ffs.DealError, error) {
	if !cfg.Enabled {
		s.l.Log(ctx, curr.Cid, "Cold-Storage was disabled, Filecoin deals will eventually expire.")
		return curr.Cold, nil, nil
	}

	sds, err := s.js.GetStartedDeals(curr.Cid)
	if err != nil {
		return ffs.ColdInfo{}, nil, fmt.Errorf("checking for started deals: %s", err)
	}
	var allErrors []ffs.DealError
	if len(sds) > 0 {
		s.l.Log(ctx, curr.Cid, "Resuming %d dettached executing deals...", len(sds))
		fsg, failedDeals, err := s.cs.WaitForDeals(ctx, curr.Cid, sds)
		if err != nil {
			return ffs.ColdInfo{}, nil, fmt.Errorf("finish tracking reattached deals: %s", err)
		}
		s.l.Log(ctx, curr.Cid, "A total of %d resumed deals finished successfully", len(fsg))
		allErrors = append(allErrors, failedDeals...)
		// Append the resumed and confirmed deals to the current active proposals
		curr.Cold.Filecoin.Proposals = append(fsg, curr.Cold.Filecoin.Proposals...)
	}

	if cfg.Filecoin.RepFactor-len(curr.Cold.Filecoin.Proposals) <= 0 {
		s.l.Log(ctx, curr.Cid, "The current replication factor is equal or higher than desired, avoiding making new deals.")
		return curr.Cold, nil, nil
	}

	deltaFilConfig := createDeltaFilConfig(cfg, curr.Cold.Filecoin)
	s.l.Log(ctx, curr.Cid, "Current replication factor is lower than desired, making %d new deals...", deltaFilConfig.RepFactor)
	startedProposals, rejectedProposals, size, err := s.cs.Store(ctx, curr.Cid, deltaFilConfig)
	if err != nil {
		return ffs.ColdInfo{}, rejectedProposals, err
	}
	allErrors = append(allErrors, rejectedProposals...)
	if err := s.js.AddStartedDeals(curr.Cid, startedProposals); err != nil {
		return ffs.ColdInfo{}, rejectedProposals, err
	}
	okDeals, failedDeals, err := s.cs.WaitForDeals(ctx, curr.Cid, startedProposals)
	allErrors = append(allErrors, failedDeals...)
	if err != nil {
		return ffs.ColdInfo{}, allErrors, fmt.Errorf("watching deals unfold: %s", err)
	}
	if err := s.js.RemoveStartedDeals(curr.Cid); err != nil {
		return ffs.ColdInfo{}, allErrors, fmt.Errorf("removing temporal started deals storage: %s", err)
	}

	return ffs.ColdInfo{Filecoin: ffs.FilInfo{
		DataCid:   curr.Cid,
		Size:      size,
		Proposals: append(okDeals, curr.Cold.Filecoin.Proposals...), // Append to any existing other proposals
	}}, allErrors, nil
}

func createDeltaFilConfig(cfg ffs.ColdConfig, curr ffs.FilInfo) ffs.FilConfig {
	res := cfg.Filecoin
	res.RepFactor = cfg.Filecoin.RepFactor - len(curr.Proposals)
	for _, p := range curr.Proposals {
		res.ExcludedMiners = append(res.ExcludedMiners, p.Miner)
	}
	return res
}
