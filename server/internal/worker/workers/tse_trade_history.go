package workers

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/services/tradehistory"
	"github.com/sirupsen/logrus"
)

const (
	tradeHistoryMaxAttempts = 3
	tradeHistoryRetryDelay  = time.Second * 30

	// tradeHistoryFlushInterval is how often job progress and heartbeat are stored
	// and a cancel request is checked.
	tradeHistoryFlushInterval = 10 * time.Second
)

// TSETradeHistory runs the trade history jobs queued from the panel. Each job
// selects markets and days; the days of a market are its daily closing price
// rows. The result and trade totals of every day are recorded in its row, so
// missing, empty, failed and mismatched days are listed in the panel and can be
// queued again.
//
// The worker runs cfg.Instances instances, each on its own goroutine. Every
// instance claims queued jobs on its own, so jobs run side by side; the claim
// is locked in Postgres, so instances of other processes never take the same
// job. An instance with no job to claim helps a job that another instance of
// this process runs, taking its markets one by one, so a large job is spread
// over every idle instance. A job interrupted by a shutdown is queued again and
// resumes with the days that are still not fetched.
//
// The instance running a job stores a heartbeat with its progress. A job whose
// heartbeat stopped for cfg.StaleJobAfter, such as the job of a killed process,
// is queued again by a goroutine that checks only while a job may go stale, see
// requeueStaleJobs. Progress is stored only while the job is still run by the
// instance, so an instance whose job was queued again stops it instead of
// running it alongside the instance that claimed it next.
type TSETradeHistory struct {
	service *tradehistory.Service
	cfg     *configs.TSETradeHistory
	id      string
	logger  *logrus.Entry
	// collect fetches the planned days of one market of a job; replaced in tests.
	collect func(ctx context.Context, job *models.TSETradeHistoryJob, market *models.TSEMarket, progress *jobProgress, logger *logrus.Entry)
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

func NewTSETradeHistory(service *tradehistory.Service, cfg *configs.TSETradeHistory) *TSETradeHistory {
	host, _ := os.Hostname()
	w := &TSETradeHistory{
		service: service,
		cfg:     cfg,
		id:      fmt.Sprintf("%s-%d", host, os.Getpid()),
		logger:  logrus.WithField("worker", "tse-trade-history"),
		wake:    make(chan struct{}),
		claimed: make(chan struct{}, 1),
	}
	w.collect = w.collectMarket
	w.requeue = service.RequeueStaleJobs
	return w
}

func (w *TSETradeHistory) Name() string { return "tse-trade-history" }

func (w *TSETradeHistory) Stop() error { return nil }

func (w *TSETradeHistory) Start(ctx context.Context) error {
	instances := max(w.cfg.Instances, 1)
	w.logger.WithFields(logrus.Fields{"worker_id": w.id, "instances": instances}).Info("starting TSE trade history job runner")
	if w.cfg.EnqueueAllOnStart {
		queued, err := w.service.EnqueueAllMarkets(ctx)
		if err != nil {
			w.logger.WithError(err).Error("queue trade history job of all markets")
		} else if queued {
			w.logger.Info("queued trade history job of all markets")
		}
	}
	var wg sync.WaitGroup
	wg.Go(func() { w.requeueStaleJobs(ctx) })
	for index := range instances {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.runInstance(ctx, index+1)
		}()
	}
	wg.Wait()
	return ctx.Err()
}

// runInstance claims and runs jobs until ctx is done. With no job to claim, it
// helps a job that another instance runs, or waits for the next poll.
func (w *TSETradeHistory) runInstance(ctx context.Context, index int) {
	id := fmt.Sprintf("%s#%d", w.id, index)
	logger := w.logger.WithField("instance", id)
	poll := w.cfg.PollInterval
	if poll <= 0 {
		poll = 15 * time.Second
	}
	for ctx.Err() == nil {
		job, err := w.service.ClaimJob(ctx, id)
		if err != nil && ctx.Err() == nil {
			logger.WithError(err).Error("claim trade history job")
		}
		if job != nil {
			w.notifyClaimed()
			w.runJob(ctx, job, id, logger)
			continue
		}
		if w.helpActiveJob(logger) {
			continue
		}
		w.mu.Lock()
		wake := w.wake
		w.mu.Unlock()
		select {
		case <-ctx.Done():
		case <-wake:
		case <-time.After(poll):
		}
	}
}

func (w *TSETradeHistory) notifyClaimed() {
	select {
	case w.claimed <- struct{}{}:
	default:
	}
}

// requeueStaleJobs queues again the running jobs whose heartbeat stopped. A
// job goes stale StaleJobAfter after its last heartbeat at the earliest, so the
// database is checked only then: once at start, for the jobs of a killed
// process, and then when the oldest heartbeat of the running jobs, of every
// process, would go stale. With no running job it sleeps until an instance of
// this process claims one.
func (w *TSETradeHistory) requeueStaleJobs(ctx context.Context) {
	after := w.cfg.StaleJobTimeout()
	check := true
	// next is when to check again; zero while no job is running.
	var next time.Time
	for {
		if check {
			requeued, stale, err := w.requeue(ctx)
			switch {
			case err != nil:
				if ctx.Err() != nil {
					return
				}
				w.logger.WithError(err).Error("requeue stale trade history jobs")
				next = time.Now().Add(after)
			case stale != nil:
				next = *stale
			default:
				next = time.Time{}
			}
			if requeued > 0 {
				w.logger.WithField("jobs", requeued).Warn("requeued trade history jobs without heartbeat")
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
		case <-w.claimed:
			// A job claimed now goes stale no sooner than after; an earlier
			// check that is due already covers it.
			check = false
			if next.IsZero() {
				next = time.Now().Add(after)
			}
		}
	}
}

// activeJob is a job run by an instance of this process. Its markets are taken
// one by one by the instance that claimed it and by idle instances that help it.
type activeJob struct {
	job      *models.TSETradeHistoryJob
	ctx      context.Context
	markets  chan *models.TSEMarket
	progress *jobProgress
	logger   *logrus.Entry
	// helpers counts the instances helping the job; draining, guarded by the
	// worker mutex, stops new instances from joining once the markets ran out.
	helpers  sync.WaitGroup
	draining bool
}

func newActiveJob(ctx context.Context, job *models.TSETradeHistoryJob, markets []*models.TSEMarket, logger *logrus.Entry) *activeJob {
	queue := make(chan *models.TSEMarket, len(markets))
	for _, market := range markets {
		queue <- market
	}
	close(queue)
	return &activeJob{job: job, ctx: ctx, markets: queue, progress: newJobProgress(), logger: logger}
}

// process fetches the markets of a with the calling instance and every idle
// instance that joins, and returns when all of them are done.
func (w *TSETradeHistory) process(a *activeJob) {
	w.mu.Lock()
	w.active = append(w.active, a)
	// Idle instances are woken to help with the new job.
	close(w.wake)
	w.wake = make(chan struct{})
	w.mu.Unlock()

	w.drain(a, a.logger)

	w.mu.Lock()
	a.draining = true
	w.active = slices.DeleteFunc(w.active, func(item *activeJob) bool { return item == a })
	w.mu.Unlock()
	a.helpers.Wait()
}

// helpActiveJob joins the oldest job of this process with markets left and
// fetches markets of it until none is left. It reports whether it helped.
func (w *TSETradeHistory) helpActiveJob(logger *logrus.Entry) bool {
	w.mu.Lock()
	var target *activeJob
	for _, a := range w.active {
		if !a.draining && len(a.markets) > 0 && a.ctx.Err() == nil {
			target = a
			break
		}
	}
	if target != nil {
		target.helpers.Add(1)
	}
	w.mu.Unlock()
	if target == nil {
		return false
	}
	defer target.helpers.Done()
	logger.WithField("job_id", target.job.ID).Info("helping trade history job")
	w.drain(target, logger.WithFields(logrus.Fields{"job_id": target.job.ID, "title": target.job.Title}))
	return true
}

// drain fetches markets of a until none is left or the job is stopped.
func (w *TSETradeHistory) drain(a *activeJob, logger *logrus.Entry) {
	for market := range a.markets {
		if a.ctx.Err() != nil {
			return
		}
		w.collect(a.ctx, a.job, market, a.progress, logger)
	}
}

// jobProgress counts the days of the current run of a job.
type jobProgress struct {
	doneMarkets atomic.Int64
	planned     atomic.Int64
	processed   atomic.Int64
	fetched     atomic.Int64
	empty       atomic.Int64
	failed      atomic.Int64

	mu sync.Mutex
	// current are the labels of the markets being fetched, by market ID.
	current map[uint]string
}

func newJobProgress() *jobProgress {
	return &jobProgress{current: make(map[uint]string)}
}

func (p *jobProgress) start(market *models.TSEMarket) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current[market.ID] = market.Label
}

func (p *jobProgress) finish(market *models.TSEMarket) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.current, market.ID)
}

func (p *jobProgress) columns() map[string]any {
	p.mu.Lock()
	labels := make([]string, 0, len(p.current))
	for _, label := range p.current {
		labels = append(labels, label)
	}
	p.mu.Unlock()
	slices.Sort(labels)
	return map[string]any{
		"done_markets":   p.doneMarkets.Load(),
		"planned_days":   p.planned.Load(),
		"processed_days": p.processed.Load(),
		"fetched_days":   p.fetched.Load(),
		"empty_days":     p.empty.Load(),
		"failed_days":    p.failed.Load(),
		"current_market": strings.Join(labels, "، "),
		"heartbeat_at":   time.Now(),
	}
}

// runJob runs a job claimed by an instance; idle instances may help it.
func (w *TSETradeHistory) runJob(ctx context.Context, job *models.TSETradeHistoryJob, worker string, instanceLogger *logrus.Entry) {
	logger := instanceLogger.WithFields(logrus.Fields{"job_id": job.ID, "title": job.Title})
	logger.Info("running trade history job")
	started := time.Now()

	// Progress is stored with a context that outlives ctx, so a shutdown can
	// still queue the job again.
	store := context.WithoutCancel(ctx)
	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	markets, err := w.service.JobMarkets(jobCtx, job)
	if err != nil {
		w.finish(store, job, worker, consts.TSETradeHistoryJobFailed, nil, fmt.Sprintf("list markets: %v", err))
		return
	}
	active := newActiveJob(jobCtx, job, markets, logger)
	progress := active.progress
	// Progress counts the current run; a resumed job plans only the days still left.
	reset := progress.columns()
	reset["total_markets"] = len(markets)
	if owned, _, err := w.service.UpdateOwnedJob(store, job.ID, worker, reset); err != nil {
		logger.WithError(err).Warn("store trade history job progress")
	} else if !owned {
		logger.Warn("trade history job was taken over before it started")
		return
	}

	var canceled, lost atomic.Bool
	flushDone := make(chan struct{})
	go func() {
		defer close(flushDone)
		ticker := time.NewTicker(tradeHistoryFlushInterval)
		defer ticker.Stop()
		for {
			select {
			case <-jobCtx.Done():
				return
			case <-ticker.C:
			}
			// A failed write keeps the job running; it is lost only if the
			// heartbeat stays out long enough for the job to be queued again.
			owned, cancelRequested, err := w.service.UpdateOwnedJob(store, job.ID, worker, progress.columns())
			switch {
			case err != nil:
				logger.WithError(err).Warn("store trade history job progress")
			case !owned:
				logger.Warn("trade history job was queued again or taken over; stopping it")
				lost.Store(true)
				cancel()
				return
			case cancelRequested:
				logger.Info("trade history job cancel requested")
				canceled.Store(true)
				cancel()
				return
			}
		}
	}()

	w.process(active)
	cancel()
	<-flushDone

	switch {
	case lost.Load():
		// The job is left to the worker that runs it now.
		return
	case canceled.Load():
		w.finish(store, job, worker, consts.TSETradeHistoryJobCanceled, progress, "")
	case ctx.Err() != nil:
		// The worker is stopping; the job resumes on the next run.
		columns := progress.columns()
		columns["status"] = consts.TSETradeHistoryJobPending
		if _, _, err := w.service.UpdateOwnedJob(store, job.ID, worker, columns); err != nil {
			logger.WithError(err).Error("queue interrupted trade history job again")
		}
		logger.Info("trade history job interrupted; queued again")
		return
	default:
		w.finish(store, job, worker, consts.TSETradeHistoryJobDone, progress, "")
	}
	logger.WithFields(logrus.Fields{
		"markets":  progress.doneMarkets.Load(),
		"days":     progress.processed.Load(),
		"fetched":  progress.fetched.Load(),
		"empty":    progress.empty.Load(),
		"failed":   progress.failed.Load(),
		"duration": time.Since(started).Round(time.Second),
	}).Info("trade history job finished")
}

// finish stores the result of a job, unless worker no longer runs it.
func (w *TSETradeHistory) finish(ctx context.Context, job *models.TSETradeHistoryJob, worker string, status consts.TSETradeHistoryJobStatus, progress *jobProgress, message string) {
	columns := map[string]any{}
	if progress != nil {
		columns = progress.columns()
	}
	columns["status"] = status
	columns["error"] = message
	columns["finished_at"] = time.Now()
	if owned, _, err := w.service.UpdateOwnedJob(ctx, job.ID, worker, columns); err != nil {
		w.logger.WithError(err).WithField("job_id", job.ID).Error("store trade history job result")
	} else if !owned {
		w.logger.WithField("job_id", job.ID).Warn("trade history job was taken over; its result is not stored")
	}
	if message != "" {
		w.logger.WithField("job_id", job.ID).WithField("error", message).Error("trade history job failed")
	}
}

// collectMarket fetches the planned days of one market of a job.
func (w *TSETradeHistory) collectMarket(ctx context.Context, job *models.TSETradeHistoryJob, market *models.TSEMarket, progress *jobProgress, logger *logrus.Entry) {
	logger = logger.WithFields(logrus.Fields{"tse_market_id": market.ID, "market": market.Label, "ins_code": market.InsCode})
	days, err := w.service.PlanDays(ctx, job, market)
	if err != nil {
		if ctx.Err() == nil {
			logger.WithError(err).Error("plan trade history days")
		}
		return
	}
	progress.planned.Add(int64(len(days)))
	if len(days) == 0 {
		progress.doneMarkets.Add(1)
		return
	}
	progress.start(market)
	defer progress.finish(market)
	logger.WithField("days", len(days)).Info("fetching trade history of market")

	jobID := job.ID
	opts := tradehistory.CollectOptions{JobID: &jobID, MaxAttempts: w.cfg.MaxAttempts, RetryDelay: w.cfg.RetryDelay}
	var fetched, empty, failed int
	for _, day := range days {
		date := day.TradeDate.UTC().Format("2006-01-02")
		result, err := w.service.CollectDay(ctx, market, day, opts)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			logger.WithError(err).WithField("date", date).Error("collect trade history day")
			continue
		}
		progress.processed.Add(1)
		switch result.Status {
		case consts.TSETradeDayFetched:
			fetched++
			progress.fetched.Add(1)
		case consts.TSETradeDayEmpty:
			empty++
			progress.empty.Add(1)
		case consts.TSETradeDayFailed:
			failed++
			progress.failed.Add(1)
			logger.WithError(result.Err).WithField("date", date).Warn("trade history day failed")
		}
	}
	if ctx.Err() == nil {
		progress.doneMarkets.Add(1)
	}
	logger.WithFields(logrus.Fields{"fetched": fetched, "empty": empty, "failed": failed}).Info("fetched trade history of market")
}
