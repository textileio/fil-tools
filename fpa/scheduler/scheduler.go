package scheduler

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/textileio/fil-tools/fpa"
)

var (
	log = logging.Logger("fpa-scheduler")
)

// Scheduler receives actions to store a Cid in Hot and Cold layers. These actions are
// created as Jobs which have a lifecycle that can be watched by external actors.
// This Jobs are executed by delegating the work to the Hot and Cold layers configured for
// the scheduler.
type Scheduler struct {
	cold  fpa.ColdLayer
	hot   fpa.HotLayer
	store JobStore

	work chan struct{}

	ctx      context.Context
	cancel   context.CancelFunc
	finished chan struct{}
}

var _ fpa.Scheduler = (*Scheduler)(nil)

// New returns a new instance of Scheduler which uses JobStore as its backing repository for state,
// HotLayer for the hot layer, and ColdLayer for the cold layer.
func New(store JobStore, hot fpa.HotLayer, cold fpa.ColdLayer) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	sch := &Scheduler{
		store: store,
		hot:   hot,
		cold:  cold,

		work: make(chan struct{}, 1),

		ctx:      ctx,
		cancel:   cancel,
		finished: make(chan struct{}),
	}
	go sch.run()
	return sch
}

// Enqueue queues the specified CidConfig to be executed as a new Job. It returns
// the created JobID for further tracking of its state.
func (s *Scheduler) Enqueue(c fpa.CidConfig) (fpa.JobID, error) {
	jid := fpa.NewJobID()
	log.Infof("enqueuing %s", jid)
	j := fpa.Job{
		ID:     jid,
		Status: fpa.Queued,
		Config: c,
		CidInfo: fpa.CidInfo{
			ConfigID: c.ID,
			Cid:      c.Cid,
			Created:  time.Now(),
		},
	}
	if err := s.store.Put(j); err != nil {
		return fpa.EmptyJobID, fmt.Errorf("saving enqueued job: %s", err)
	}
	select {
	case s.work <- struct{}{}:
	default:
	}
	return jid, nil
}

// GetFromHot returns an io.Reader of the data from the hot layer.
// (TODO: in the future rate-limiting can be applied.)
func (s *Scheduler) GetFromHot(ctx context.Context, c cid.Cid) (io.Reader, error) {
	r, err := s.hot.Get(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("getting %s from hot layer: %s", c, err)
	}
	return r, nil
}

// Watch returns a channel to listen to Job status changes from a specified
// FastAPI instance.
func (s *Scheduler) Watch(iid fpa.InstanceID) <-chan fpa.Job {
	return s.store.Watch(iid)
}

// Unwatch unregisters a subscribing channel created by Watch().
func (s *Scheduler) Unwatch(ch <-chan fpa.Job) {
	s.store.Unwatch(ch)
}

// Close terminates the scheduler.
func (s *Scheduler) Close() error {
	s.cancel()
	<-s.finished
	return nil
}

func (s *Scheduler) run() {
	defer close(s.finished)
	for {
		select {
		case <-s.ctx.Done():
			log.Infof("terminating scheduler daemon")
			return
		case <-s.work:
			js, err := s.store.GetByStatus(fpa.Queued)
			if err != nil {
				log.Errorf("getting queued jobs: %s", err)
				continue
			}
			log.Infof("detected %d queued jobs", len(js))
			for _, j := range js {
				log.Infof("executing job %s", j.ID)
				if err := s.execute(s.ctx, &j); err != nil {
					log.Errorf("executing job %s: %s", j.ID, err)
					continue
				}
				log.Infof("job %s executed with final state %d and errcause %q", j.ID, j.Status, j.ErrCause)
				if err := s.store.Put(j); err != nil {
					log.Errorf("saving job %s: %s", j.ID, err)
				}
			}
		}
	}
}

func (s *Scheduler) execute(ctx context.Context, job *fpa.Job) error {
	cinfo := fpa.CidInfo{
		ConfigID: job.Config.ID,
		Cid:      job.Config.Cid,
		Created:  time.Now(),
	}
	var err error
	cinfo.Hot, err = s.hot.Pin(ctx, job.Config.Cid)
	if err != nil {
		job.Status = fpa.Failed
		job.ErrCause = err.Error()
		return nil
	}

	cinfo.Cold, err = s.cold.Store(ctx, job.Config.Cid, job.Config.Cold)
	if err != nil {
		job.Status = fpa.Failed
		job.ErrCause = err.Error()
		return nil
	}
	job.CidInfo = cinfo
	job.Status = fpa.Done
	return nil
}
