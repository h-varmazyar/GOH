package workers

import (
	"context"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/services/tradehistory"
	"github.com/h-varmazyar/gatec/internal/worker/jobrunner"
	"github.com/sirupsen/logrus"
)

const (
	tradeHistoryMaxAttempts = 3
	tradeHistoryRetryDelay  = time.Second * 30
)

// TSETradeHistory runs the trade history jobs queued from the panel or by the
// job-scheduler worker. Each job selects markets and days; the days of a market
// are its daily closing price rows. The result and trade totals of every day
// are recorded in its row, so missing, empty, failed and mismatched days are
// listed in the panel and can be queued again.
//
// The jobs are run by a jobrunner.Runner, with the markets of a job as its
// units: several instances, woken by Postgres notifications when a job is
// queued, share the markets of a job, and a job of a stopped or killed process
// is queued again. When a job finishes, the queues waiting for it, such as the
// candle jobs, are woken.
type TSETradeHistory struct {
	service *tradehistory.Service
	cfg     *configs.TSETradeHistory
	runner  *jobrunner.Runner
}

// NewTSETradeHistory creates the worker; listen reports Postgres
// notifications of the job queue, see pgnotify.Listener, and may be nil.
func NewTSETradeHistory(service *tradehistory.Service, cfg *configs.TSETradeHistory, listen func(ctx context.Context, wake func())) *TSETradeHistory {
	w := &TSETradeHistory{service: service, cfg: cfg}
	w.runner = jobrunner.New(tradeHistoryQueue{w}, jobrunner.Options{
		Name:         w.Name(),
		Instances:    cfg.Instances,
		PollInterval: cfg.PollInterval,
		StaleAfter:   cfg.StaleJobTimeout(),
		Listen:       listen,
		OnFinish: func(ctx context.Context, job jobrunner.Job, _ consts.JobStatus) {
			if err := service.NotifyJobFinished(ctx); err != nil {
				logrus.WithError(err).WithField("job_id", job.ID()).Warn("announce finished trade history job")
			}
		},
	})
	return w
}

func (w *TSETradeHistory) Name() string { return "tse-trade-history" }

func (w *TSETradeHistory) Stop() error { return nil }

func (w *TSETradeHistory) Start(ctx context.Context) error {
	if w.cfg.EnqueueAllOnStart {
		queued, err := w.service.EnqueueAllMarkets(ctx)
		if err != nil {
			logrus.WithError(err).Error("queue trade history job of all markets")
		} else if queued {
			logrus.Info("queued trade history job of all markets")
		}
	}
	return w.runner.Run(ctx)
}

// tradeHistoryQueue serves the trade history jobs to the runner.
type tradeHistoryQueue struct{ w *TSETradeHistory }

func (q tradeHistoryQueue) Claim(ctx context.Context, worker string) (jobrunner.Job, error) {
	job, err := q.w.service.ClaimJob(ctx, worker)
	if err != nil || job == nil {
		return nil, err
	}
	return &tradeHistoryJob{w: q.w, job: job, progress: newJobProgress()}, nil
}

func (q tradeHistoryQueue) UpdateOwned(ctx context.Context, id uint, worker string, columns map[string]any) (bool, bool, error) {
	return q.w.service.UpdateOwnedJob(ctx, id, worker, columns)
}

func (q tradeHistoryQueue) RequeueStale(ctx context.Context) (int64, *time.Time, error) {
	return q.w.service.RequeueStaleJobs(ctx)
}

func (q tradeHistoryQueue) Notify(ctx context.Context) error { return q.w.service.NotifyJobs(ctx) }

// tradeHistoryJob is a claimed trade history job; its units are its markets.
type tradeHistoryJob struct {
	w        *TSETradeHistory
	job      *models.TSETradeHistoryJob
	markets  []*models.TSEMarket
	progress *jobProgress
}

func (j *tradeHistoryJob) ID() uint      { return j.job.ID }
func (j *tradeHistoryJob) Title() string { return j.job.Title }

// Plan lists the markets of the job. Progress counts the current run; a
// resumed job plans only the days still left.
func (j *tradeHistoryJob) Plan(ctx context.Context) (int, error) {
	markets, err := j.w.service.JobMarkets(ctx, j.job)
	if err != nil {
		return 0, err
	}
	j.markets = markets
	j.progress.totalMarkets = len(markets)
	return len(markets), nil
}

func (j *tradeHistoryJob) Run(ctx context.Context, unit int, logger *logrus.Entry) {
	j.w.collectMarket(ctx, j.job, j.markets[unit], j.progress, logger)
}

func (j *tradeHistoryJob) Progress() map[string]any { return j.progress.columns() }

// jobProgress counts the days of the current run of a job.
type jobProgress struct {
	totalMarkets int
	doneMarkets  atomic.Int64
	planned      atomic.Int64
	processed    atomic.Int64
	fetched      atomic.Int64
	empty        atomic.Int64
	failed       atomic.Int64

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
		"total_markets":  p.totalMarkets,
		"done_markets":   p.doneMarkets.Load(),
		"planned_days":   p.planned.Load(),
		"processed_days": p.processed.Load(),
		"fetched_days":   p.fetched.Load(),
		"empty_days":     p.empty.Load(),
		"failed_days":    p.failed.Load(),
		"current_market": strings.Join(labels, "، "),
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
