package tradehistory

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

// maxJobDates bounds the explicit dates of a job.
const maxJobDates = 1000

// DefaultDayStatuses are fetched by jobs that select no statuses: days not
// fetched yet and failed days.
var DefaultDayStatuses = []consts.TSETradeDayStatus{consts.TSETradeDayMissing, consts.TSETradeDayFailed}

var dayStatusLabel = map[consts.TSETradeDayStatus]string{
	consts.TSETradeDayMissing: "دریافت‌نشده",
	consts.TSETradeDayFailed:  "خطادار",
	consts.TSETradeDayEmpty:   "بدون معامله",
	consts.TSETradeDayFetched:    "دریافت‌شده",
	consts.TSETradeDayMismatched: "دارای مغایرت",
}

// JobRequest queues a collection. Exactly one of TSEMarketIDs, AllMarkets and
// Gaps selects the markets.
type JobRequest struct {
	TSEMarketIDs []uint
	AllMarkets   bool
	// Gaps selects the markets matching a gaps page filter when the job is queued.
	Gaps *repositories.TradeGapFilter
	// Dates are fetched whatever their status; From, To and DayStatuses are then ignored.
	Dates       []time.Time
	From        *time.Time
	To          *time.Time
	DayStatuses []consts.TSETradeDayStatus
}

// CreateJob queues a collection for the tse-trade-history worker.
func (s *Service) CreateJob(ctx context.Context, req JobRequest) (*models.TSETradeHistoryJob, error) {
	selectors := 0
	for _, set := range []bool{len(req.TSEMarketIDs) > 0, req.AllMarkets, req.Gaps != nil} {
		if set {
			selectors++
		}
	}
	if selectors != 1 {
		return nil, fmt.Errorf("%w: select markets by ids, all_markets or gaps", ErrInvalidInput)
	}
	today := s.Today()
	job := &models.TSETradeHistoryJob{
		Status:       consts.TSETradeHistoryJobPending,
		AllMarkets:   req.AllMarkets,
		TSEMarketIDs: []uint{},
		Dates:        []string{},
		DayStatuses:  []consts.TSETradeDayStatus{},
	}

	if len(req.Dates) > maxJobDates {
		return nil, fmt.Errorf("%w: at most %d dates", ErrInvalidInput, maxJobDates)
	}
	for _, date := range req.Dates {
		date = dateOnly(date)
		if date.After(today) {
			return nil, fmt.Errorf("%w: %s is in the future", ErrInvalidInput, date.Format(dateLayout))
		}
		if !slices.Contains(job.Dates, date.Format(dateLayout)) {
			job.Dates = append(job.Dates, date.Format(dateLayout))
		}
	}
	if len(job.Dates) == 0 {
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
		for _, status := range req.DayStatuses {
			if !status.Valid() {
				return nil, fmt.Errorf("%w: day status %q", ErrInvalidInput, status)
			}
			if !slices.Contains(job.DayStatuses, status) {
				job.DayStatuses = append(job.DayStatuses, status)
			}
		}
		if len(job.DayStatuses) == 0 {
			job.DayStatuses = append(job.DayStatuses, DefaultDayStatuses...)
		}
	}

	var label string
	switch {
	case req.AllMarkets:
		label = "همه نمادهای فعال"
	case req.Gaps != nil:
		ids, err := s.statusRepo.GapMarketIDs(ctx, *req.Gaps)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return nil, fmt.Errorf("%w: no market matches the filter", ErrInvalidInput)
		}
		job.TSEMarketIDs = ids
		label = fmt.Sprintf("%d نماد", len(ids))
	default:
		for _, id := range req.TSEMarketIDs {
			if id != 0 && !slices.Contains(job.TSEMarketIDs, id) {
				job.TSEMarketIDs = append(job.TSEMarketIDs, id)
			}
		}
		markets, err := s.marketRepo.List(ctx, map[string]any{"id IN ?": job.TSEMarketIDs})
		if err != nil {
			return nil, err
		}
		if len(markets) != len(job.TSEMarketIDs) {
			return nil, fmt.Errorf("%w: %d of the markets were not found", ErrNotFound, len(job.TSEMarketIDs)-len(markets))
		}
		label = fmt.Sprintf("%d نماد", len(markets))
		if len(markets) == 1 {
			label = markets[0].Label
		}
	}
	job.Title = label + "، " + jobScope(job)
	if err := s.statusRepo.CreateJob(ctx, job); err != nil {
		return nil, err
	}
	s.logger.WithField("job_id", job.ID).WithField("title", job.Title).Info("queued trade history job")
	return job, nil
}

func jobScope(job *models.TSETradeHistoryJob) string {
	if len(job.Dates) > 0 {
		if len(job.Dates) == 1 {
			return "روز " + job.Dates[0]
		}
		return fmt.Sprintf("%d روز انتخابی", len(job.Dates))
	}
	labels := make([]string, len(job.DayStatuses))
	for i, status := range job.DayStatuses {
		labels[i] = dayStatusLabel[status]
	}
	scope := "روزهای " + strings.Join(labels, "، ")
	if job.FromDate != nil {
		scope += " از " + job.FromDate.Format(dateLayout)
	}
	if job.ToDate != nil {
		scope += " تا " + job.ToDate.Format(dateLayout)
	}
	return scope
}

func (s *Service) ListJobs(ctx context.Context, filter repositories.TradeHistoryJobFilter, page paginator.Params) ([]*models.TSETradeHistoryJob, int64, error) {
	return s.statusRepo.PageJobs(ctx, filter, page)
}

func (s *Service) GetJob(ctx context.Context, id uint) (*models.TSETradeHistoryJob, error) {
	job, err := s.statusRepo.GetJob(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: job %d", ErrNotFound, id)
	}
	return job, err
}

// CancelJob cancels a queued job, or asks the worker to stop a running job.
func (s *Service) CancelJob(ctx context.Context, id uint) (*models.TSETradeHistoryJob, error) {
	if _, err := s.GetJob(ctx, id); err != nil {
		return nil, err
	}
	active, err := s.statusRepo.CancelJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, fmt.Errorf("%w: job %d is already finished", ErrInvalidInput, id)
	}
	return s.GetJob(ctx, id)
}

// --- Worker ---

// EnqueueAllMarkets queues a job of every market with data collection enabled,
// unless one is already queued or running. It reports whether a job was queued.
func (s *Service) EnqueueAllMarkets(ctx context.Context) (bool, error) {
	active, err := s.statusRepo.HasActiveAllMarketsJob(ctx)
	if err != nil || active {
		return false, err
	}
	_, err = s.CreateJob(ctx, JobRequest{AllMarkets: true})
	return err == nil, err
}

// ClaimJob marks the oldest queued job as running by worker; nil when none is queued.
func (s *Service) ClaimJob(ctx context.Context, worker string) (*models.TSETradeHistoryJob, error) {
	return s.statusRepo.ClaimJob(ctx, worker)
}

// RequeueStaleJobs queues again the running jobs whose worker stopped sending
// heartbeats. next is when a job still running goes stale if its heartbeat
// stops; nil when no job is running.
func (s *Service) RequeueStaleJobs(ctx context.Context) (requeued int64, next *time.Time, err error) {
	after := s.cfg.StaleJobTimeout()
	if requeued, err = s.statusRepo.RequeueStaleJobs(ctx, time.Now().Add(-after)); err != nil {
		return 0, nil, err
	}
	oldest, err := s.statusRepo.OldestRunningHeartbeat(ctx)
	if err != nil || oldest == nil {
		return requeued, nil, err
	}
	stale := oldest.Add(after)
	return requeued, &stale, nil
}

// UpdateOwnedJob updates a job only while worker runs it; see
// TradeHistoryStatusRepository.UpdateOwnedJob.
func (s *Service) UpdateOwnedJob(ctx context.Context, id uint, worker string, columns map[string]any) (owned, cancelRequested bool, err error) {
	return s.statusRepo.UpdateOwnedJob(ctx, id, worker, columns)
}

// JobMarkets returns the markets of a job that can be fetched.
func (s *Service) JobMarkets(ctx context.Context, job *models.TSETradeHistoryJob) ([]*models.TSEMarket, error) {
	var markets []*models.TSEMarket
	var err error
	if job.AllMarkets {
		markets, err = s.marketRepo.ListDataCollection(ctx)
	} else {
		markets, err = s.marketRepo.List(ctx, map[string]any{"id IN ?": job.TSEMarketIDs})
	}
	if err != nil {
		return nil, err
	}
	result := make([]*models.TSEMarket, 0, len(markets))
	for _, market := range markets {
		if market.InsCode != "" {
			result = append(result, market)
		}
	}
	return result, nil
}

// PlanDays returns the trading days of market that job fetches, newest first.
func (s *Service) PlanDays(ctx context.Context, job *models.TSETradeHistoryJob, market *models.TSEMarket) ([]repositories.TradeDayRef, error) {
	dates := make([]time.Time, 0, len(job.Dates))
	for _, value := range job.Dates {
		date, err := ParseDate(value)
		if err != nil {
			return nil, err
		}
		dates = append(dates, date)
	}
	statuses := job.DayStatuses
	if len(statuses) == 0 {
		statuses = DefaultDayStatuses
	}
	return s.statusRepo.PlanDays(ctx, market.ID, dates, job.FromDate, job.ToDate, statuses)
}

// FindDay returns the trading day of a market on date; nil when it has no daily closing price that day.
func (s *Service) FindDay(ctx context.Context, tseMarketID uint, date time.Time) (*repositories.TradeDayRef, error) {
	return s.statusRepo.FindDay(ctx, tseMarketID, dateOnly(date))
}
