package candles

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/pkg/paginator"
	"gorm.io/gorm"
)

// maxJobMarkets bounds the markets listed in a job.
const maxJobMarkets = 5000

// JobRequest queues a candle build. Exactly one of MarketIDs and AllMarkets
// selects the markets; every ID must be a row that belongs to the provider.
type JobRequest struct {
	ProviderID uint
	MarketIDs  []uint
	AllMarkets bool
	// TimeframeIDs default to every supported timeframe of the provider.
	TimeframeIDs []uint
	// From and To bound the trading days to build; nil is unbounded. A single
	// day, such as the last trading day, is From = To.
	From *time.Time
	To   *time.Time
	// AfterTradeHistoryJobID holds the job until that trade history job has finished.
	AfterTradeHistoryJobID *uint
}

// CreateJob checks a request against the database and queues it for the
// candle-builder worker.
func (s *Service) CreateJob(ctx context.Context, req JobRequest) (*models.CandleJob, error) {
	provider, err := s.Provider(ctx, req.ProviderID)
	if err != nil {
		return nil, err
	}
	if (len(req.MarketIDs) > 0) == req.AllMarkets {
		return nil, fmt.Errorf("%w: select markets by market_ids or all_markets", ErrInvalidInput)
	}
	job := &models.CandleJob{
		Status:       consts.JobPending,
		ProviderID:   provider.ID,
		AllMarkets:   req.AllMarkets,
		MarketIDs:    []uint{},
		TimeframeIDs: []uint{},
	}

	for _, id := range req.MarketIDs {
		if id != 0 && !slices.Contains(job.MarketIDs, id) {
			job.MarketIDs = append(job.MarketIDs, id)
		}
	}
	if len(job.MarketIDs) > maxJobMarkets {
		return nil, fmt.Errorf("%w: at most %d markets; use all_markets", ErrInvalidInput, maxJobMarkets)
	}
	if !req.AllMarkets {
		found, err := s.jobRepo.ProviderMarketIDs(ctx, provider.ID, job.MarketIDs)
		if err != nil {
			return nil, err
		}
		if missing := without(job.MarketIDs, found); len(missing) > 0 {
			return nil, fmt.Errorf("%w: markets %v are not markets of provider %d", ErrNotFound, missing, provider.ID)
		}
	}

	timeframes, err := s.Timeframes(ctx, provider.ID)
	if err != nil {
		return nil, err
	}
	if len(timeframes) == 0 {
		return nil, fmt.Errorf("%w: provider %d has no supported timeframe", ErrInvalidInput, provider.ID)
	}
	labels := make([]string, 0, len(timeframes))
	for _, timeframe := range timeframes {
		if len(req.TimeframeIDs) == 0 || slices.Contains(req.TimeframeIDs, timeframe.ID) {
			job.TimeframeIDs = append(job.TimeframeIDs, timeframe.ID)
			labels = append(labels, timeframe.Label)
		}
	}
	if missing := without(req.TimeframeIDs, job.TimeframeIDs); len(missing) > 0 {
		return nil, fmt.Errorf("%w: timeframes %v are not supported timeframes of provider %d", ErrNotFound, missing, provider.ID)
	}

	today := s.Today()
	if req.From != nil {
		from := dateOnly(*req.From)
		job.FromDate = &from
	}
	if req.To != nil {
		to := dateOnly(*req.To)
		job.ToDate = &to
	}
	if job.FromDate != nil && job.ToDate != nil && job.ToDate.Before(*job.FromDate) {
		return nil, fmt.Errorf("%w: from is after to", ErrInvalidInput)
	}
	if job.FromDate != nil && job.FromDate.After(today) {
		return nil, fmt.Errorf("%w: from is in the future", ErrInvalidInput)
	}

	if req.AfterTradeHistoryJobID != nil {
		exists, err := s.jobRepo.TradeHistoryJobExists(ctx, *req.AfterTradeHistoryJobID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("%w: trade history job %d", ErrNotFound, *req.AfterTradeHistoryJobID)
		}
		id := *req.AfterTradeHistoryJobID
		job.AfterTradeHistoryJobID = &id
	}

	marketLabel := "همه نمادها"
	switch {
	case len(job.MarketIDs) == 1:
		if marketLabel, err = s.jobRepo.MarketLabel(ctx, job.MarketIDs[0]); err != nil {
			return nil, err
		}
	case !job.AllMarkets:
		marketLabel = fmt.Sprintf("%d نماد", len(job.MarketIDs))
	}
	job.Title = marketLabel + "، " + strings.Join(labels, "/") + "، " + jobScope(job)
	if err := s.jobRepo.CreateJob(ctx, job); err != nil {
		return nil, err
	}
	s.logger.WithField("job_id", job.ID).WithField("title", job.Title).Info("queued candle job")
	return job, nil
}

// without returns the items of values missing from present.
func without(values, present []uint) []uint {
	missing := make([]uint, 0)
	for _, value := range values {
		if !slices.Contains(present, value) {
			missing = append(missing, value)
		}
	}
	return missing
}

func jobScope(job *models.CandleJob) string {
	switch {
	case job.FromDate == nil && job.ToDate == nil:
		return "همه روزها"
	case job.FromDate != nil && job.ToDate != nil && job.FromDate.Equal(*job.ToDate):
		return "روز " + job.FromDate.Format(dateLayout)
	}
	scope := "روزهای"
	if job.FromDate != nil {
		scope += " از " + job.FromDate.Format(dateLayout)
	}
	if job.ToDate != nil {
		scope += " تا " + job.ToDate.Format(dateLayout)
	}
	return scope
}

func (s *Service) ListJobs(ctx context.Context, filter repositories.CandleJobFilter, page paginator.Params) ([]*models.CandleJob, int64, error) {
	return s.jobRepo.PageJobs(ctx, filter, page)
}

func (s *Service) GetJob(ctx context.Context, id uint) (*models.CandleJob, error) {
	job, err := s.jobRepo.GetJob(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: job %d", ErrNotFound, id)
	}
	return job, err
}

// CancelJob cancels a queued job, or asks the worker to stop a running job.
func (s *Service) CancelJob(ctx context.Context, id uint) (*models.CandleJob, error) {
	if _, err := s.GetJob(ctx, id); err != nil {
		return nil, err
	}
	active, err := s.jobRepo.CancelJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, fmt.Errorf("%w: job %d is already finished", ErrInvalidInput, id)
	}
	return s.GetJob(ctx, id)
}

// --- Worker ---

// ClaimJob marks the oldest runnable job as running by worker; nil when none is runnable.
func (s *Service) ClaimJob(ctx context.Context, worker string) (*models.CandleJob, error) {
	return s.jobRepo.ClaimJob(ctx, worker)
}

// UpdateOwnedJob updates a job only while worker runs it.
func (s *Service) UpdateOwnedJob(ctx context.Context, id uint, worker string, columns map[string]any) (owned, cancelRequested bool, err error) {
	return s.jobRepo.UpdateOwnedJob(ctx, id, worker, columns)
}

// RequeueStaleJobs queues again the running jobs whose worker stopped sending
// heartbeats. next is when a job still running goes stale if its heartbeat
// stops; nil when no job is running.
func (s *Service) RequeueStaleJobs(ctx context.Context) (requeued int64, next *time.Time, err error) {
	after := s.cfg.StaleJobTimeout()
	if requeued, err = s.jobRepo.RequeueStaleJobs(ctx, time.Now().Add(-after)); err != nil {
		return 0, nil, err
	}
	oldest, err := s.jobRepo.OldestRunningHeartbeat(ctx)
	if err != nil || oldest == nil {
		return requeued, nil, err
	}
	stale := oldest.Add(after)
	return requeued, &stale, nil
}

// NotifyJobs wakes the candle-builder workers of every process.
func (s *Service) NotifyJobs(ctx context.Context) error { return s.jobRepo.NotifyJobs(ctx) }

// JobTimeframes returns the timeframes of a job, checked again against the
// provider when the job starts.
func (s *Service) JobTimeframes(ctx context.Context, job *models.CandleJob) ([]repositories.CandleTimeframe, error) {
	timeframes, err := s.Timeframes(ctx, job.ProviderID)
	if err != nil {
		return nil, err
	}
	result := make([]repositories.CandleTimeframe, 0, len(job.TimeframeIDs))
	for _, timeframe := range timeframes {
		if slices.Contains(job.TimeframeIDs, timeframe.ID) {
			result = append(result, repositories.CandleTimeframe{ID: timeframe.ID, Duration: timeframe.Duration})
		}
	}
	if len(result) != len(job.TimeframeIDs) {
		return nil, fmt.Errorf("timeframes %v are no longer supported timeframes of provider %d", job.TimeframeIDs, job.ProviderID)
	}
	return result, nil
}

// JobMarkets returns the markets of a job with their insCodes.
func (s *Service) JobMarkets(ctx context.Context, job *models.CandleJob) ([]repositories.TSEMarketSource, error) {
	if _, err := s.Provider(ctx, job.ProviderID); err != nil {
		return nil, err
	}
	return s.jobRepo.TSEMarketSources(ctx, job.ProviderID, job.MarketIDs, job.AllMarkets)
}
