package repositories

import (
	"context"
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/pkg/paginator"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CandleJobChannel wakes the candle-builder workers.
const CandleJobChannel = "candle_jobs"

// CandleJobRepository stores the candle jobs and reads the markets and
// timeframes they select.
type CandleJobRepository struct {
	db   *gorm.DB
	jobs *JobQueue[models.CandleJob]
}

func NewCandleJobRepository(db *gorm.DB) *CandleJobRepository {
	// A job waiting for a trade history job is runnable once that job has
	// finished, or was deleted.
	runnable := func(tx *gorm.DB) *gorm.DB {
		return tx.Where(`after_trade_history_job_id IS NULL OR NOT EXISTS (
			SELECT 1 FROM tse_trade_history_jobs t WHERE t.id = candle_jobs.after_trade_history_job_id AND t.status IN ?)`,
			consts.JobActive)
	}
	return &CandleJobRepository{db: db, jobs: NewJobQueue[models.CandleJob](db, CandleJobChannel, runnable)}
}

// --- Jobs ---

// CreateJob stores a queued job and wakes the workers.
func (r *CandleJobRepository) CreateJob(ctx context.Context, job *models.CandleJob) error {
	return r.jobs.Create(ctx, job)
}

func (r *CandleJobRepository) GetJob(ctx context.Context, id uint) (*models.CandleJob, error) {
	return r.jobs.Get(ctx, id)
}

func (r *CandleJobRepository) ClaimJob(ctx context.Context, worker string) (*models.CandleJob, error) {
	return r.jobs.Claim(ctx, worker)
}

func (r *CandleJobRepository) CancelJob(ctx context.Context, id uint) (bool, error) {
	return r.jobs.Cancel(ctx, id)
}

func (r *CandleJobRepository) RequeueStaleJobs(ctx context.Context, before time.Time) (int64, error) {
	return r.jobs.RequeueStale(ctx, before)
}

func (r *CandleJobRepository) OldestRunningHeartbeat(ctx context.Context) (*time.Time, error) {
	return r.jobs.OldestRunningHeartbeat(ctx)
}

func (r *CandleJobRepository) UpdateOwnedJob(ctx context.Context, id uint, worker string, columns map[string]any) (owned, cancelRequested bool, err error) {
	return r.jobs.UpdateOwned(ctx, id, worker, columns)
}

func (r *CandleJobRepository) NotifyJobs(ctx context.Context) error {
	return r.jobs.Notify(ctx)
}

// CandleJobFilter selects jobs in the panel; empty fields match everything.
type CandleJobFilter struct {
	Statuses []consts.JobStatus
	// MarketID selects the jobs of every market and the jobs of this market.
	MarketID uint
}

func (f CandleJobFilter) apply(tx *gorm.DB) *gorm.DB {
	if len(f.Statuses) > 0 {
		tx = tx.Where("status IN ?", f.Statuses)
	}
	if f.MarketID != 0 {
		tx = tx.Where("all_markets OR market_ids @> ?::jsonb", jsonArray(f.MarketID))
	}
	return tx
}

// PageJobs lists jobs, active and newest first.
func (r *CandleJobRepository) PageJobs(ctx context.Context, filter CandleJobFilter, page paginator.Params) ([]*models.CandleJob, int64, error) {
	var total int64
	if err := filter.apply(r.db.WithContext(ctx).Model(&models.CandleJob{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	jobs := make([]*models.CandleJob, 0)
	err := page.Apply(filter.apply(r.db.WithContext(ctx).Model(&models.CandleJob{}))).
		Order(clause.OrderBy{Expression: clause.Expr{SQL: "CASE status WHEN ? THEN 0 WHEN ? THEN 1 ELSE 2 END, id DESC",
			Vars: []any{consts.JobRunning, consts.JobPending}}}).
		Find(&jobs).Error
	return jobs, total, err
}

// TradeHistoryJobExists reports whether the trade history job id exists.
func (r *CandleJobRepository) TradeHistoryJobExists(ctx context.Context, id uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.TSETradeHistoryJob{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

// --- Markets ---

// TSEMarketSource is a market with the insCodes its trades are stored under.
type TSEMarketSource struct {
	MarketID uint
	Label    string
	InsCodes []string
}

// TSEMarketSources returns the markets of the provider with their insCodes,
// ordered by ID: the markets ids, or with all every market with an insCode
// whose data collection is enabled. Markets without insCodes are left out.
func (r *CandleJobRepository) TSEMarketSources(ctx context.Context, providerID uint, ids []uint, all bool) ([]TSEMarketSource, error) {
	tx := r.db.WithContext(ctx).Table("markets m").
		Joins("JOIN tse_markets t ON t.market_id = m.id").
		Select("m.id AS market_id, m.label, t.ins_code").
		Where("m.provider_id = ? AND m.deleted_at IS NULL AND t.ins_code <> ''", providerID)
	if all {
		tx = tx.Where("EXISTS (SELECT 1 FROM tse_markets e WHERE e.market_id = m.id AND e.data_collection_enabled)")
	} else {
		tx = tx.Where("m.id IN ?", ids)
	}
	var rows []struct {
		MarketID uint
		Label    string
		InsCode  string
	}
	if err := tx.Order("m.id, t.id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	sources := make([]TSEMarketSource, 0)
	for _, row := range rows {
		if n := len(sources); n > 0 && sources[n-1].MarketID == row.MarketID {
			sources[n-1].InsCodes = append(sources[n-1].InsCodes, row.InsCode)
			continue
		}
		sources = append(sources, TSEMarketSource{MarketID: row.MarketID, Label: row.Label, InsCodes: []string{row.InsCode}})
	}
	return sources, nil
}

// ProviderMarketIDs returns which of ids are markets of the provider.
func (r *CandleJobRepository) ProviderMarketIDs(ctx context.Context, providerID uint, ids []uint) ([]uint, error) {
	found := make([]uint, 0, len(ids))
	err := r.db.WithContext(ctx).Model(&models.Market{}).
		Where("provider_id = ? AND id IN ?", providerID, ids).Pluck("id", &found).Error
	return found, err
}

// MarketLabel returns the label of a market.
func (r *CandleJobRepository) MarketLabel(ctx context.Context, id uint) (string, error) {
	var label string
	err := r.db.WithContext(ctx).Model(&models.Market{}).Where("id = ?", id).Pluck("label", &label).Error
	return label, err
}
