package workers

import (
	"context"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/internal/services/candles"
	"github.com/h-varmazyar/gatec/internal/worker/jobrunner"
	"github.com/sirupsen/logrus"
)

// CandleBuilder runs the candle jobs queued from the panel or by the
// job-scheduler worker: it builds the candles of the markets of a job in its
// timeframes from the stored trades, in ClickHouse. Building a range again
// replaces its candles, so the open candle of a week is completed by the next
// build.
//
// The jobs are run by a jobrunner.Runner, with the markets of a job as its
// units, like the tse-trade-history worker. The worker does not poll: a
// Postgres notification wakes it when a job is queued, and when a trade
// history job finishes, since candle jobs may wait for one.
//
// CandlePolling streams live crypto candles from exchange sockets into
// Postgres; this worker aggregates stored trades in batches, so the two share
// no code. Building candles continuously only needs jobs queued more often,
// such as by a scheduler, or a job of the day queued when its trades arrive.
type CandleBuilder struct {
	service *candles.Service
	cfg     *configs.CandleBuilder
	runner  *jobrunner.Runner
}

// NewCandleBuilder creates the worker; listen reports Postgres notifications
// of the candle and trade history job queues and may be nil.
func NewCandleBuilder(service *candles.Service, cfg *configs.CandleBuilder, listen func(ctx context.Context, wake func())) *CandleBuilder {
	w := &CandleBuilder{service: service, cfg: cfg}
	w.runner = jobrunner.New(candleQueue{service}, jobrunner.Options{
		Name:         w.Name(),
		Instances:    cfg.Instances,
		PollInterval: cfg.PollInterval,
		StaleAfter:   cfg.StaleJobTimeout(),
		Listen:       listen,
	})
	return w
}

func (w *CandleBuilder) Name() string { return "candle-builder" }

func (w *CandleBuilder) Stop() error { return nil }

func (w *CandleBuilder) Start(ctx context.Context) error {
	// The jobs name timeframes by ID, so the supported timeframes of TSE must exist.
	if provider, err := w.service.TSEProvider(ctx); err != nil {
		logrus.WithError(err).Warn("candle timeframes of TSE were not checked")
	} else if _, err := w.service.EnsureTimeframes(ctx, provider.ID); err != nil {
		logrus.WithError(err).Error("create candle timeframes of TSE")
	}
	return w.runner.Run(ctx)
}

// candleQueue serves the candle jobs to the runner.
type candleQueue struct{ service *candles.Service }

func (q candleQueue) Claim(ctx context.Context, worker string) (jobrunner.Job, error) {
	job, err := q.service.ClaimJob(ctx, worker)
	if err != nil || job == nil {
		return nil, err
	}
	return &candleJob{service: q.service, job: job, current: make(map[uint]string)}, nil
}

func (q candleQueue) UpdateOwned(ctx context.Context, id uint, worker string, columns map[string]any) (bool, bool, error) {
	return q.service.UpdateOwnedJob(ctx, id, worker, columns)
}

func (q candleQueue) RequeueStale(ctx context.Context) (int64, *time.Time, error) {
	return q.service.RequeueStaleJobs(ctx)
}

func (q candleQueue) Notify(ctx context.Context) error { return q.service.NotifyJobs(ctx) }

// candleJob is a claimed candle job; its units are its markets. Progress
// counts the current run: a resumed job builds every market again, which
// replaces the candles built before.
type candleJob struct {
	service    *candles.Service
	job        *models.CandleJob
	timeframes []repositories.CandleTimeframe
	markets    []repositories.TSEMarketSource

	done, empty, failed, built atomic.Int64

	mu sync.Mutex
	// current are the labels of the markets being built, by market ID.
	current map[uint]string
}

func (j *candleJob) ID() uint      { return j.job.ID }
func (j *candleJob) Title() string { return j.job.Title }

func (j *candleJob) Plan(ctx context.Context) (int, error) {
	timeframes, err := j.service.JobTimeframes(ctx, j.job)
	if err != nil {
		return 0, err
	}
	markets, err := j.service.JobMarkets(ctx, j.job)
	if err != nil {
		return 0, err
	}
	j.timeframes, j.markets = timeframes, markets
	return len(markets), nil
}

func (j *candleJob) Run(ctx context.Context, unit int, logger *logrus.Entry) {
	market := j.markets[unit]
	logger = logger.WithFields(logrus.Fields{"market_id": market.MarketID, "market": market.Label})
	j.mu.Lock()
	j.current[market.MarketID] = market.Label
	j.mu.Unlock()
	defer func() {
		j.mu.Lock()
		delete(j.current, market.MarketID)
		j.mu.Unlock()
	}()

	result, err := j.service.BuildMarket(ctx, j.job, j.timeframes, market)
	j.built.Add(result.Candles)
	switch {
	case ctx.Err() != nil:
		return
	case err != nil:
		j.failed.Add(1)
		logger.WithError(err).Error("build candles of market")
	case result.Empty:
		j.empty.Add(1)
	default:
		logger.WithFields(logrus.Fields{"candles": result.Candles, "chunks": result.Chunks}).Debug("built candles of market")
	}
	j.done.Add(1)
}

func (j *candleJob) Progress() map[string]any {
	j.mu.Lock()
	labels := make([]string, 0, len(j.current))
	for _, label := range j.current {
		labels = append(labels, label)
	}
	j.mu.Unlock()
	slices.Sort(labels)
	return map[string]any{
		"total_markets":  len(j.markets),
		"done_markets":   j.done.Load(),
		"empty_markets":  j.empty.Load(),
		"failed_markets": j.failed.Load(),
		"candles":        j.built.Load(),
		"current_market": strings.Join(labels, "، "),
	}
}
