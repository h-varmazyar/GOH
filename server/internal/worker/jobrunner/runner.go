// Package jobrunner runs the jobs of a job queue table on several instances.
// Jobs are queued in Postgres, by the panel or by the job-scheduler worker,
// and each queue is served by a worker that wraps a Runner, such as the
// tse-trade-history and candle-builder workers.
//
// A runner runs Options.Instances instances, each on its own goroutine. Every
// instance claims queued jobs on its own, so jobs run side by side; the claim
// is locked in Postgres, so instances of other processes never take the same
// job. A job is split into units, such as its markets; an instance with no job
// to claim helps a job that another instance of this process runs, taking its
// units one by one, so a large job is spread over every idle instance. A job
// interrupted by a shutdown is queued again and resumes on the next run.
//
// Idle instances do not poll the queue: they wait for a wake-up from
// Options.Listen, which reports Postgres notifications of the queue, and look
// at the queue only on a long fallback interval otherwise.
//
// The instance running a job stores a heartbeat with its progress. A job whose
// heartbeat stopped for Options.StaleAfter, such as the job of a killed
// process, is queued again by a goroutine that checks only while a job may go
// stale, see requeueStaleJobs. Progress is stored only while the job is still
// run by the instance, so an instance whose job was queued again stops it
// instead of running it alongside the instance that claimed it next.
package jobrunner

import (
	"context"
	"fmt"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/sirupsen/logrus"
)

const (
	// DefaultPollInterval is how often idle instances look at the queue when
	// no notification wakes them.
	DefaultPollInterval = 10 * time.Minute
	// DefaultStaleAfter queues again a running job without a heartbeat for this long.
	DefaultStaleAfter = 5 * time.Minute
	// FlushInterval is how often the progress and heartbeat of a job are
	// stored and a cancel request is checked.
	FlushInterval = 10 * time.Second
)

// Queue is the job table a runner serves.
type Queue interface {
	// Claim marks the oldest runnable job as run by worker; nil when none is runnable.
	Claim(ctx context.Context, worker string) (Job, error)
	// UpdateOwned updates a job only while worker runs it. owned is false when
	// the job was queued again or claimed by another worker.
	UpdateOwned(ctx context.Context, id uint, worker string, columns map[string]any) (owned, cancelRequested bool, err error)
	// RequeueStale queues again the running jobs whose heartbeat stopped. next
	// is when a job still running goes stale if its heartbeat stops; nil when
	// no job is running.
	RequeueStale(ctx context.Context) (requeued int64, next *time.Time, err error)
	// Notify wakes the runners of the queue in every process.
	Notify(ctx context.Context) error
}

// Job is a claimed job. Its units are run by one or more instances at once,
// so Run and Progress must be safe for concurrent use.
type Job interface {
	ID() uint
	Title() string
	// Plan prepares the job and returns its number of units; an error fails the job.
	Plan(ctx context.Context) (units int, err error)
	// Run runs one unit, 0 <= unit < units. It records its result in the
	// progress and logs its failures; a unit cut short by ctx is left unfinished.
	Run(ctx context.Context, unit int, logger *logrus.Entry)
	// Progress returns the progress columns of the job.
	Progress() map[string]any
}

type Options struct {
	// Name is the worker name used in logs and in the instance IDs.
	Name      string
	Instances int
	// PollInterval is how often idle instances look at the queue without a
	// wake-up; DefaultPollInterval when not set.
	PollInterval time.Duration
	// StaleAfter is DefaultStaleAfter when not set.
	StaleAfter time.Duration
	// Listen calls wake whenever a job may have become runnable, until ctx is
	// done; see pgnotify.Listener. Without it the instances only poll.
	Listen func(ctx context.Context, wake func())
	// OnFinish is called after a job finished, whatever its result, such as
	// to wake the queues whose jobs wait for it.
	OnFinish func(ctx context.Context, job Job, status consts.JobStatus)
}

type Runner struct {
	queue  Queue
	opts   Options
	id     string
	logger *logrus.Entry
	// requeue queues the stale jobs again; replaced in tests.
	requeue func(ctx context.Context) (requeued int64, next *time.Time, err error)
	// claimed gets a signal when an instance claims a job, which may go stale.
	claimed chan struct{}

	// mu guards active, the jobs run by instances of this process oldest
	// first, and wake, which is closed to wake idle instances.
	mu     sync.Mutex
	active []*activeJob
	wake   chan struct{}
}

func New(queue Queue, opts Options) *Runner {
	host, _ := os.Hostname()
	if opts.PollInterval <= 0 {
		opts.PollInterval = DefaultPollInterval
	}
	if opts.StaleAfter <= 0 {
		opts.StaleAfter = DefaultStaleAfter
	}
	r := &Runner{
		queue:   queue,
		opts:    opts,
		id:      fmt.Sprintf("%s-%d", host, os.Getpid()),
		logger:  logrus.WithField("worker", opts.Name),
		wake:    make(chan struct{}),
		claimed: make(chan struct{}, 1),
	}
	if queue != nil {
		r.requeue = queue.RequeueStale
	}
	return r
}

// Run runs the instances until ctx is done.
func (r *Runner) Run(ctx context.Context) error {
	instances := max(r.opts.Instances, 1)
	r.logger.WithFields(logrus.Fields{"worker_id": r.id, "instances": instances}).Info("starting job runner")
	var wg sync.WaitGroup
	wg.Go(func() { r.requeueStaleJobs(ctx) })
	if r.opts.Listen != nil {
		wg.Go(func() { r.opts.Listen(ctx, r.Wake) })
	}
	for index := range instances {
		wg.Go(func() { r.runInstance(ctx, index+1) })
	}
	wg.Wait()
	return ctx.Err()
}

// Wake makes the idle instances look at the queue now.
func (r *Runner) Wake() {
	r.mu.Lock()
	close(r.wake)
	r.wake = make(chan struct{})
	r.mu.Unlock()
}

// runInstance claims and runs jobs until ctx is done. With no job to claim, it
// helps a job that another instance runs, or waits for a wake-up.
func (r *Runner) runInstance(ctx context.Context, index int) {
	id := fmt.Sprintf("%s#%d", r.id, index)
	logger := r.logger.WithField("instance", id)
	for ctx.Err() == nil {
		// The wake channel is taken before looking at the queue, so a wake-up
		// sent while looking is not missed.
		r.mu.Lock()
		wake := r.wake
		r.mu.Unlock()

		job, err := r.queue.Claim(ctx, id)
		if err != nil && ctx.Err() == nil {
			logger.WithError(err).Error("claim job")
		}
		if job != nil {
			r.notifyClaimed()
			r.runJob(ctx, job, id, logger)
			continue
		}
		if r.helpActiveJob(logger) {
			continue
		}
		timer := time.NewTimer(r.opts.PollInterval)
		select {
		case <-ctx.Done():
		case <-wake:
		case <-timer.C:
		}
		timer.Stop()
	}
}

func (r *Runner) notifyClaimed() {
	select {
	case r.claimed <- struct{}{}:
	default:
	}
}

// requeueStaleJobs queues again the running jobs whose heartbeat stopped. A
// job goes stale StaleAfter after its last heartbeat at the earliest, so the
// database is checked only then: once at start, for the jobs of a killed
// process, and then when the oldest heartbeat of the running jobs, of every
// process, would go stale. With no running job it sleeps until an instance of
// this process claims one.
func (r *Runner) requeueStaleJobs(ctx context.Context) {
	after := r.opts.StaleAfter
	check := true
	// next is when to check again; zero while no job is running.
	var next time.Time
	for {
		if check {
			requeued, stale, err := r.requeue(ctx)
			switch {
			case err != nil:
				if ctx.Err() != nil {
					return
				}
				r.logger.WithError(err).Error("requeue stale jobs")
				next = time.Now().Add(after)
			case stale != nil:
				next = *stale
			default:
				next = time.Time{}
			}
			if requeued > 0 {
				r.logger.WithField("jobs", requeued).Warn("requeued jobs without heartbeat")
			}
		}
		var due <-chan time.Time
		if !next.IsZero() {
			due = time.After(time.Until(next))
		}
		select {
		case <-ctx.Done():
			return
		case <-due:
			check = true
		case <-r.claimed:
			// A job claimed now goes stale no sooner than after; an earlier
			// check that is due already covers it.
			check = false
			if next.IsZero() {
				next = time.Now().Add(after)
			}
		}
	}
}

// activeJob is a job run by an instance of this process. Its units are taken
// one by one by the instance that claimed it and by idle instances that help it.
type activeJob struct {
	job    Job
	ctx    context.Context
	units  chan int
	logger *logrus.Entry
	// helpers counts the instances helping the job; draining, guarded by the
	// runner mutex, stops new instances from joining once the units ran out.
	helpers  sync.WaitGroup
	draining bool
}

func newActiveJob(ctx context.Context, job Job, units int, logger *logrus.Entry) *activeJob {
	queue := make(chan int, units)
	for unit := range units {
		queue <- unit
	}
	close(queue)
	return &activeJob{job: job, ctx: ctx, units: queue, logger: logger}
}

// process runs the units of a with the calling instance and every idle
// instance that joins, and returns when all of them are done.
func (r *Runner) process(a *activeJob) {
	r.mu.Lock()
	r.active = append(r.active, a)
	r.mu.Unlock()
	// Idle instances are woken to help with the new job.
	r.Wake()

	r.drain(a, a.logger)

	r.mu.Lock()
	a.draining = true
	r.active = slices.DeleteFunc(r.active, func(item *activeJob) bool { return item == a })
	r.mu.Unlock()
	a.helpers.Wait()
}

// helpActiveJob joins the oldest job of this process with units left and runs
// units of it until none is left. It reports whether it helped.
func (r *Runner) helpActiveJob(logger *logrus.Entry) bool {
	r.mu.Lock()
	var target *activeJob
	for _, a := range r.active {
		if !a.draining && len(a.units) > 0 && a.ctx.Err() == nil {
			target = a
			break
		}
	}
	if target != nil {
		target.helpers.Add(1)
	}
	r.mu.Unlock()
	if target == nil {
		return false
	}
	defer target.helpers.Done()
	logger.WithField("job_id", target.job.ID()).Info("helping job")
	r.drain(target, logger.WithFields(logrus.Fields{"job_id": target.job.ID(), "title": target.job.Title()}))
	return true
}

// drain runs units of a until none is left or the job is stopped.
func (r *Runner) drain(a *activeJob, logger *logrus.Entry) {
	for unit := range a.units {
		if a.ctx.Err() != nil {
			return
		}
		a.job.Run(a.ctx, unit, logger)
	}
}

// columns are the progress columns of job with a new heartbeat.
func columns(job Job) map[string]any {
	columns := job.Progress()
	columns["heartbeat_at"] = time.Now()
	return columns
}

// runJob runs a job claimed by an instance; idle instances may help it.
func (r *Runner) runJob(ctx context.Context, job Job, worker string, instanceLogger *logrus.Entry) {
	logger := instanceLogger.WithFields(logrus.Fields{"job_id": job.ID(), "title": job.Title()})
	logger.Info("running job")
	started := time.Now()

	// Progress is stored with a context that outlives ctx, so a shutdown can
	// still queue the job again.
	store := context.WithoutCancel(ctx)
	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	units, err := job.Plan(jobCtx)
	if err != nil {
		if ctx.Err() != nil {
			r.requeueInterrupted(store, job, worker, logger)
			return
		}
		r.finish(store, job, worker, consts.JobFailed, fmt.Sprintf("plan job: %v", err))
		return
	}
	if owned, _, err := r.queue.UpdateOwned(store, job.ID(), worker, columns(job)); err != nil {
		logger.WithError(err).Warn("store job progress")
	} else if !owned {
		logger.Warn("job was taken over before it started")
		return
	}

	var canceled, lost atomic.Bool
	flushDone := make(chan struct{})
	go func() {
		defer close(flushDone)
		ticker := time.NewTicker(FlushInterval)
		defer ticker.Stop()
		for {
			select {
			case <-jobCtx.Done():
				return
			case <-ticker.C:
			}
			// A failed write keeps the job running; it is lost only if the
			// heartbeat stays out long enough for the job to be queued again.
			owned, cancelRequested, err := r.queue.UpdateOwned(store, job.ID(), worker, columns(job))
			switch {
			case err != nil:
				logger.WithError(err).Warn("store job progress")
			case !owned:
				logger.Warn("job was queued again or taken over; stopping it")
				lost.Store(true)
				cancel()
				return
			case cancelRequested:
				logger.Info("job cancel requested")
				canceled.Store(true)
				cancel()
				return
			}
		}
	}()

	r.process(newActiveJob(jobCtx, job, units, logger))
	cancel()
	<-flushDone

	switch {
	case lost.Load():
		// The job is left to the worker that runs it now.
		return
	case canceled.Load():
		r.finish(store, job, worker, consts.JobCanceled, "")
	case ctx.Err() != nil:
		r.requeueInterrupted(store, job, worker, logger)
		return
	default:
		r.finish(store, job, worker, consts.JobDone, "")
	}
	logger.WithFields(logrus.Fields(job.Progress())).WithField("duration", time.Since(started).Round(time.Second)).
		Info("job finished")
}

// requeueInterrupted queues a job stopped by a shutdown again; it resumes on
// the next run, here or in another process.
func (r *Runner) requeueInterrupted(ctx context.Context, job Job, worker string, logger *logrus.Entry) {
	update := columns(job)
	update["status"] = consts.JobPending
	if _, _, err := r.queue.UpdateOwned(ctx, job.ID(), worker, update); err != nil {
		logger.WithError(err).Error("queue interrupted job again")
		return
	}
	if err := r.queue.Notify(ctx); err != nil {
		logger.WithError(err).Warn("wake runners of interrupted job")
	}
	logger.Info("job interrupted; queued again")
}

// finish stores the result of a job, unless worker no longer runs it.
func (r *Runner) finish(ctx context.Context, job Job, worker string, status consts.JobStatus, message string) {
	update := columns(job)
	update["status"] = status
	update["error"] = message
	update["finished_at"] = time.Now()
	logger := r.logger.WithField("job_id", job.ID())
	owned, _, err := r.queue.UpdateOwned(ctx, job.ID(), worker, update)
	switch {
	case err != nil:
		logger.WithError(err).Error("store job result")
	case !owned:
		logger.Warn("job was taken over; its result is not stored")
		return
	}
	if message != "" {
		logger.WithField("error", message).Error("job failed")
	}
	if r.opts.OnFinish != nil {
		r.opts.OnFinish(ctx, job, status)
	}
}
