package workers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/internal/services/candles"
	"github.com/h-varmazyar/gatec/internal/services/tradehistory"
	"github.com/sirupsen/logrus"
)

// dailyJobsSchedule names the daily run in the scheduled_runs table.
const dailyJobsSchedule = "daily-jobs"

// jobSchedulerRetry is how long a failed run waits before it is made again.
const jobSchedulerRetry = 5 * time.Minute

// JobScheduler queues the daily jobs of the job workers once a day, so the
// job workers themselves only wait for jobs: it queues a trade history job of
// the missing and failed days of the last days, and a candle job of the same
// days that waits for the trade history job to finish.
//
// The run of a day is recorded in Postgres before its jobs are queued, so it is
// made once whatever the number of scheduler processes. With CatchUp, a run
// whose time passed while no scheduler was running is made at start.
type JobScheduler struct {
	cfg          *configs.JobScheduler
	tradeHistory *tradehistory.Service
	candles      *candles.Service
	runs         *repositories.ScheduledRunRepository
	location     *time.Location
	logger       *logrus.Entry
}

func NewJobScheduler(cfg *configs.JobScheduler, tradeHistory *tradehistory.Service, candleService *candles.Service, runs *repositories.ScheduledRunRepository) *JobScheduler {
	location, err := time.LoadLocation(repositories.TehranTimezone)
	if err != nil {
		location = time.FixedZone(repositories.TehranTimezone, 3*3600+1800)
	}
	return &JobScheduler{
		cfg:          cfg,
		tradeHistory: tradeHistory,
		candles:      candleService,
		runs:         runs,
		location:     location,
		logger:       logrus.WithField("worker", "job-scheduler"),
	}
}

func (w *JobScheduler) Name() string { return "job-scheduler" }

func (w *JobScheduler) Stop() error { return nil }

func (w *JobScheduler) Start(ctx context.Context) error {
	now := time.Now().In(w.location)
	if w.cfg.CatchUp {
		today := time.Date(now.Year(), now.Month(), now.Day(), w.cfg.ScheduleHour, w.cfg.ScheduleMin, 0, 0, w.location)
		if !now.Before(today) {
			w.run(ctx, today)
		}
	}
	for ctx.Err() == nil {
		next := nextDailyRun(time.Now(), w.location, w.cfg.ScheduleHour, w.cfg.ScheduleMin)
		w.logger.WithField("next_run", next.Format(time.RFC3339)).Info("scheduled next daily jobs")
		if !sleep(ctx, time.Until(next)) {
			break
		}
		w.run(ctx, next)
	}
	return ctx.Err()
}

// run makes the run of the day of at, again after a failure until it succeeds
// or the next run is due.
func (w *JobScheduler) run(ctx context.Context, at time.Time) {
	date := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC)
	logger := w.logger.WithField("date", date.Format(time.DateOnly))
	deadline := at.AddDate(0, 0, 1)
	for ctx.Err() == nil {
		detail, err := w.queueDailyJobs(ctx, date)
		if err == nil {
			if detail != "" {
				logger.WithField("jobs", detail).Info("queued daily jobs")
			}
			return
		}
		logger.WithError(err).Error("queue daily jobs")
		if time.Now().Add(jobSchedulerRetry).After(deadline) || !sleep(ctx, jobSchedulerRetry) {
			return
		}
	}
}

// queueDailyJobs queues the jobs of date unless they were queued already, and
// describes what it queued; empty when the run was already made.
func (w *JobScheduler) queueDailyJobs(ctx context.Context, date time.Time) (string, error) {
	claimed, err := w.runs.ClaimScheduledRun(ctx, dailyJobsSchedule, date)
	if err != nil || !claimed {
		return "", err
	}
	detail, err := w.createJobs(ctx, date)
	if err != nil {
		// The run is made again; jobs already queued may be queued twice,
		// which only repeats work that finds nothing left to do.
		if releaseErr := w.runs.ReleaseScheduledRun(context.WithoutCancel(ctx), dailyJobsSchedule, date); releaseErr != nil {
			w.logger.WithError(releaseErr).Error("release failed daily run")
		}
		return "", err
	}
	if detail == "" {
		detail = "nothing to queue"
	}
	if err := w.runs.FinishScheduledRun(ctx, dailyJobsSchedule, date, detail); err != nil {
		w.logger.WithError(err).Warn("store daily run detail")
	}
	return detail, nil
}

func (w *JobScheduler) createJobs(ctx context.Context, date time.Time) (string, error) {
	var queued []string
	var tradeHistoryJob *uint
	if days := w.cfg.TradeHistoryLookbackDays; days > 0 {
		from := date.AddDate(0, 0, 1-days)
		job, err := w.tradeHistory.CreateJob(ctx, tradehistory.JobRequest{
			AllMarkets: true, From: &from, To: &date, DayStatuses: tradehistory.DefaultDayStatuses,
		})
		if err != nil {
			return "", fmt.Errorf("queue trade history job: %w", err)
		}
		tradeHistoryJob = &job.ID
		queued = append(queued, fmt.Sprintf("trade history job %d", job.ID))
	}
	if days := w.cfg.CandleLookbackDays; days > 0 {
		provider, err := w.candles.TSEProvider(ctx)
		if err != nil {
			return "", err
		}
		timeframes, err := w.candles.TimeframesByLabel(ctx, provider.ID, w.cfg.CandleTimeframes)
		if err != nil {
			return "", err
		}
		from := date.AddDate(0, 0, 1-days)
		job, err := w.candles.CreateJob(ctx, candles.JobRequest{
			ProviderID: provider.ID, AllMarkets: true, TimeframeIDs: timeframes,
			From: &from, To: &date, AfterTradeHistoryJobID: tradeHistoryJob,
		})
		if err != nil {
			return "", fmt.Errorf("queue candle job: %w", err)
		}
		queued = append(queued, fmt.Sprintf("candle job %d", job.ID))
	}
	return strings.Join(queued, ", "), nil
}

// sleep waits for d; false when ctx was done first.
func sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
