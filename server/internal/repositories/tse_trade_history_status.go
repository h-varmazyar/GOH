package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/pkg/paginator"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const sqlDate = "2006-01-02"

// TradeHistoryStatusRepository records the trade history fetch results in the
// daily closing price rows, which are the trading days of each TSE market, and
// stores the queued trade history jobs.
type TradeHistoryStatusRepository struct{ db *gorm.DB }

func NewTradeHistoryStatusRepository(db *gorm.DB) *TradeHistoryStatusRepository {
	return &TradeHistoryStatusRepository{db: db}
}

// tradeDaysFrom joins the daily closing prices with their candles; candle times
// are UTC midnights of the trading days.
const tradeDaysFrom = "tse_daily_closing_prices p JOIN candles c ON c.id = p.candle_id"

const tradeDayDate = "(to_timestamp(c.time) AT TIME ZONE 'UTC')::date"

// mismatchedCondition selects fetched or empty days whose trade count or volume,
// canceled trades left out, differ from the closing price of the day.
const mismatchedCondition = `(p.history_status IN ('FETCHED', 'EMPTY') AND (
	p.history_trade_count - p.history_canceled_count <> p.trade_count OR p.history_net_volume <> p.volume::bigint))`

// statusCondition is the condition of the days with one of the statuses.
func statusCondition(statuses []consts.TSETradeDayStatus) string {
	conditions := make([]string, 0, len(statuses))
	for _, status := range statuses {
		switch status {
		case consts.TSETradeDayMissing:
			conditions = append(conditions, "p.history_status IS NULL")
		case consts.TSETradeDayMismatched:
			conditions = append(conditions, mismatchedCondition)
		case consts.TSETradeDayFetched, consts.TSETradeDayEmpty, consts.TSETradeDayFailed:
			conditions = append(conditions, fmt.Sprintf("p.history_status = '%s'", status))
		}
	}
	if len(conditions) == 0 {
		return "TRUE"
	}
	return "(" + strings.Join(conditions, " OR ") + ")"
}

// --- Days ---

// TradeDayRef is a trading day of a market: its daily closing price row.
type TradeDayRef struct {
	ID            uint
	TradeDate     time.Time
	HistoryStatus *consts.TSETradeDayStatus
}

// TradeDayResult is the result of a trade history fetch of a day.
type TradeDayResult struct {
	Status        consts.TSETradeDayStatus
	TradeCount    int64
	CanceledCount int64
	Volume        int64
	NetVolume     int64
	NetValue      int64
	Attempts      int
	Error         string
	JobID         *uint
	FetchedAt     time.Time
}

// SaveDayResult records a fetch result in the daily closing price row id. The
// attempts are added to those of earlier fetches.
func (r *TradeHistoryStatusRepository) SaveDayResult(ctx context.Context, id uint, result TradeDayResult) error {
	update := r.db.WithContext(ctx).Model(&models.TSEDailyClosingPrice{}).Where("id = ?", id).UpdateColumns(map[string]any{
		"history_status":         result.Status,
		"history_trade_count":    result.TradeCount,
		"history_canceled_count": result.CanceledCount,
		"history_volume":         result.Volume,
		"history_net_volume":     result.NetVolume,
		"history_net_value":      result.NetValue,
		"history_attempts":       gorm.Expr("history_attempts + ?", result.Attempts),
		"history_error":          result.Error,
		"history_job_id":         result.JobID,
		"history_fetched_at":     result.FetchedAt,
	})
	if update.Error == nil && update.RowsAffected == 0 {
		return fmt.Errorf("daily closing price %d not found", id)
	}
	return update.Error
}

// FindDay returns the trading day of the market on date, or nil when the market
// has no daily closing price on that day.
func (r *TradeHistoryStatusRepository) FindDay(ctx context.Context, tseMarketID uint, date time.Time) (*TradeDayRef, error) {
	days := make([]TradeDayRef, 0, 1)
	err := r.db.WithContext(ctx).Raw("SELECT p.id, "+tradeDayDate+" AS trade_date, p.history_status FROM "+tradeDaysFrom+
		" WHERE p.tse_market_id = ? AND c.time = ? LIMIT 1", tseMarketID, date.Unix()).Scan(&days).Error
	if err != nil || len(days) == 0 {
		return nil, err
	}
	return &days[0], nil
}

// PlanDays returns the trading days of the market to fetch, newest first: the
// days on dates when given, otherwise the days in [from, to] with one of the statuses.
func (r *TradeHistoryStatusRepository) PlanDays(ctx context.Context, tseMarketID uint, dates []time.Time, from, to *time.Time, statuses []consts.TSETradeDayStatus) ([]TradeDayRef, error) {
	tx := r.db.WithContext(ctx).Table(tradeDaysFrom).
		Select("p.id, "+tradeDayDate+" AS trade_date, p.history_status").
		Where("p.tse_market_id = ?", tseMarketID)
	if len(dates) > 0 {
		times := make([]int64, len(dates))
		for i, date := range dates {
			times[i] = date.Unix()
		}
		tx = tx.Where("c.time IN ?", times)
	} else {
		if from != nil {
			tx = tx.Where("c.time >= ?", from.Unix())
		}
		if to != nil {
			tx = tx.Where("c.time <= ?", to.Unix())
		}
		tx = tx.Where(statusCondition(statuses))
	}
	days := make([]TradeDayRef, 0)
	return days, tx.Order("c.time DESC").Scan(&days).Error
}

// TradeDayCounts counts the trading days of a market by trade history status.
type TradeDayCounts struct {
	TotalDays      int
	FetchedDays    int
	EmptyDays      int
	FailedDays     int
	MissingDays    int
	MismatchedDays int
	FirstDate      *time.Time
	LastDate       *time.Time
}

const dayCountColumns = `COUNT(*) AS total_days,
	COUNT(*) FILTER (WHERE p.history_status = 'FETCHED') AS fetched_days,
	COUNT(*) FILTER (WHERE p.history_status = 'EMPTY') AS empty_days,
	COUNT(*) FILTER (WHERE p.history_status = 'FAILED') AS failed_days,
	COUNT(*) FILTER (WHERE p.history_status IS NULL) AS missing_days,
	COUNT(*) FILTER (WHERE ` + mismatchedCondition + `) AS mismatched_days`

func (r *TradeHistoryStatusRepository) CountDays(ctx context.Context, tseMarketID uint) (TradeDayCounts, error) {
	var counts TradeDayCounts
	err := r.db.WithContext(ctx).Raw("SELECT "+dayCountColumns+`,
		MIN(`+tradeDayDate+`) AS first_date, MAX(`+tradeDayDate+`) AS last_date
		FROM `+tradeDaysFrom+" WHERE p.tse_market_id = ?", tseMarketID).Scan(&counts).Error
	return counts, err
}

// TradeDayFilter selects trading days of a market; empty fields match everything.
type TradeDayFilter struct {
	Status consts.TSETradeDayStatus
	From   *time.Time
	To     *time.Time
}

// TradeDay is a trading day of a market with its closing price figures and the
// figures of its fetched trade history.
type TradeDay struct {
	ID                   uint
	TradeDate            time.Time
	TradeCount           int64
	Volume               float64
	Value                int64
	HistoryStatus        *consts.TSETradeDayStatus
	HistoryTradeCount    int64
	HistoryCanceledCount int64
	HistoryVolume        int64
	HistoryNetVolume     int64
	HistoryNetValue      int64
	HistoryAttempts      int
	HistoryError         string
	HistoryJobID         *uint
	HistoryFetchedAt     *time.Time
	Mismatched           bool
}

// PageDays lists the trading days of a market, newest first.
func (r *TradeHistoryStatusRepository) PageDays(ctx context.Context, tseMarketID uint, filter TradeDayFilter, page paginator.Params) ([]TradeDay, int64, error) {
	query := func() *gorm.DB {
		tx := r.db.WithContext(ctx).Table(tradeDaysFrom).Where("p.tse_market_id = ?", tseMarketID)
		if filter.Status != "" {
			tx = tx.Where(statusCondition([]consts.TSETradeDayStatus{filter.Status}))
		}
		if filter.From != nil {
			tx = tx.Where("c.time >= ?", filter.From.Unix())
		}
		if filter.To != nil {
			tx = tx.Where("c.time <= ?", filter.To.Unix())
		}
		return tx
	}
	var total int64
	if err := query().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	days := make([]TradeDay, 0)
	err := page.Apply(query()).Select("p.id, " + tradeDayDate + ` AS trade_date, p.trade_count, p.volume, p.value,
		p.history_status, p.history_trade_count, p.history_canceled_count, p.history_volume, p.history_net_volume,
		p.history_net_value, p.history_attempts, p.history_error, p.history_job_id, p.history_fetched_at,
		` + mismatchedCondition + " AS mismatched").Order("c.time DESC").Scan(&days).Error
	return days, total, err
}

// --- Gaps ---

// TradeGapProblem selects markets by what is wrong with their trade history.
type TradeGapProblem string

const (
	// TradeGapIncomplete: missing or failed days.
	TradeGapIncomplete TradeGapProblem = ""
	TradeGapMissing    TradeGapProblem = "missing"
	TradeGapFailed     TradeGapProblem = "failed"
	TradeGapEmpty      TradeGapProblem = "empty"
	TradeGapMismatched TradeGapProblem = "mismatched"
	// TradeGapAny: missing, failed, empty or mismatched days.
	TradeGapAny TradeGapProblem = "any"
	// TradeGapAll lists every market with trading days.
	TradeGapAll TradeGapProblem = "all"
)

// TradeGapFilter selects markets on the trade history gaps page.
type TradeGapFilter struct {
	Query                 string
	Problem               TradeGapProblem
	Delisted              *bool
	DataCollectionEnabled *bool
}

// TradeGap is a market with the counts of its trading days.
type TradeGap struct {
	TSEMarketID           uint
	MarketID              uint
	InsCode               string
	Label                 string
	Name                  string
	Delisted              bool
	DataCollectionEnabled bool
	TradeDayCounts
	LastFetchedAt *time.Time
}

// gapQuery builds the query of markets with their day counts. Markets without
// daily closing prices have no trading days and are left out.
func gapQuery(filter TradeGapFilter) (string, map[string]any) {
	conditions := []string{"TRUE"}
	args := map[string]any{}
	if query := strings.TrimSpace(filter.Query); query != "" {
		conditions = append(conditions, "(t.label ILIKE @like OR t.name ILIKE @like OR t.ins_code = @query)")
		args["like"], args["query"] = "%"+query+"%", query
	}
	if filter.Delisted != nil {
		conditions = append(conditions, "t.delisted = @delisted")
		args["delisted"] = *filter.Delisted
	}
	if filter.DataCollectionEnabled != nil {
		conditions = append(conditions, "t.data_collection_enabled = @collect")
		args["collect"] = *filter.DataCollectionEnabled
	}
	problem := map[TradeGapProblem]string{
		TradeGapIncomplete: "(missing_days > 0 OR failed_days > 0)",
		TradeGapMissing:    "missing_days > 0",
		TradeGapFailed:     "failed_days > 0",
		TradeGapEmpty:      "empty_days > 0",
		TradeGapMismatched: "mismatched_days > 0",
		TradeGapAny:        "(missing_days > 0 OR failed_days > 0 OR empty_days > 0 OR mismatched_days > 0)",
		TradeGapAll:        "TRUE",
	}[filter.Problem]
	if problem == "" {
		problem = "TRUE"
	}
	query := `
		SELECT * FROM (
			SELECT t.id AS tse_market_id, t.market_id, t.ins_code, t.label, t.name, t.delisted, t.data_collection_enabled, days.*
			FROM tse_markets t
			JOIN (
				-- Joining candles for the dates of every market is slow, so the
				-- dates are read for the listed markets only; see gapDates.
				SELECT p.tse_market_id AS day_market_id, ` + dayCountColumns + `,
					MAX(p.history_fetched_at) AS last_fetched_at
				FROM tse_daily_closing_prices p
				GROUP BY p.tse_market_id
			) days ON days.day_market_id = t.id
			WHERE ` + strings.Join(conditions, " AND ") + `
		) gaps WHERE ` + problem
	return query, args
}

// raw runs query with named args; an empty args map would be bound as a
// positional argument, so it is left out.
func (r *TradeHistoryStatusRepository) raw(ctx context.Context, query string, args map[string]any) *gorm.DB {
	if len(args) == 0 {
		return r.db.WithContext(ctx).Raw(query)
	}
	return r.db.WithContext(ctx).Raw(query, args)
}

// PageGaps lists markets with their day counts, the most incomplete first. The
// number of matching markets is TradeGapTotals.Markets.
func (r *TradeHistoryStatusRepository) PageGaps(ctx context.Context, filter TradeGapFilter, page paginator.Params) ([]TradeGap, error) {
	query, args := gapQuery(filter)
	args["limit"], args["offset"] = page.Limit(), page.Offset()
	gaps := make([]TradeGap, 0)
	err := r.raw(ctx, query+` ORDER BY missing_days + failed_days DESC, mismatched_days DESC, empty_days DESC,
		label ASC, tse_market_id ASC LIMIT @limit OFFSET @offset`, args).Scan(&gaps).Error
	if err != nil || len(gaps) == 0 {
		return gaps, err
	}
	return gaps, r.gapDates(ctx, gaps)
}

// gapDates sets the first and last trading days of the markets.
func (r *TradeHistoryStatusRepository) gapDates(ctx context.Context, gaps []TradeGap) error {
	ids := make([]uint, len(gaps))
	for i, gap := range gaps {
		ids[i] = gap.TSEMarketID
	}
	rows := make([]struct {
		TSEMarketID uint
		FirstDate   *time.Time
		LastDate    *time.Time
	}, 0, len(gaps))
	err := r.db.WithContext(ctx).Table(tradeDaysFrom).
		Select("p.tse_market_id, MIN("+tradeDayDate+") AS first_date, MAX("+tradeDayDate+") AS last_date").
		Where("p.tse_market_id IN ?", ids).Group("p.tse_market_id").Scan(&rows).Error
	if err != nil {
		return err
	}
	for _, row := range rows {
		for i := range gaps {
			if gaps[i].TSEMarketID == row.TSEMarketID {
				gaps[i].FirstDate, gaps[i].LastDate = row.FirstDate, row.LastDate
			}
		}
	}
	return nil
}

// GapMarketIDs returns the IDs of every market that matches filter.
func (r *TradeHistoryStatusRepository) GapMarketIDs(ctx context.Context, filter TradeGapFilter) ([]uint, error) {
	query, args := gapQuery(filter)
	ids := make([]uint, 0)
	err := r.raw(ctx, "SELECT tse_market_id FROM ("+query+") selected ORDER BY tse_market_id", args).Scan(&ids).Error
	return ids, err
}

// TradeGapTotals sums the day counts of the markets that match filter.
type TradeGapTotals struct {
	Markets        int64
	TotalDays      int64
	FetchedDays    int64
	EmptyDays      int64
	FailedDays     int64
	MissingDays    int64
	MismatchedDays int64
}

func (r *TradeHistoryStatusRepository) GapTotals(ctx context.Context, filter TradeGapFilter) (TradeGapTotals, error) {
	var totals TradeGapTotals
	query, args := gapQuery(filter)
	err := r.raw(ctx, `SELECT COUNT(*) AS markets,
		COALESCE(SUM(total_days), 0) AS total_days, COALESCE(SUM(fetched_days), 0) AS fetched_days,
		COALESCE(SUM(empty_days), 0) AS empty_days, COALESCE(SUM(failed_days), 0) AS failed_days,
		COALESCE(SUM(missing_days), 0) AS missing_days, COALESCE(SUM(mismatched_days), 0) AS mismatched_days
		FROM (`+query+`) totals`, args).Scan(&totals).Error
	return totals, err
}

// CountWithoutDays counts the markets that match filter but have no daily
// closing prices, so they have no trading days to fetch yet.
func (r *TradeHistoryStatusRepository) CountWithoutDays(ctx context.Context, filter TradeGapFilter) (int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.TSEMarket{}).
		Where("NOT EXISTS (SELECT 1 FROM tse_daily_closing_prices p WHERE p.tse_market_id = tse_markets.id)")
	if filter.Delisted != nil {
		tx = tx.Where("delisted = ?", *filter.Delisted)
	}
	if filter.DataCollectionEnabled != nil {
		tx = tx.Where("data_collection_enabled = ?", *filter.DataCollectionEnabled)
	}
	var count int64
	return count, tx.Count(&count).Error
}

// --- Jobs ---

func (r *TradeHistoryStatusRepository) CreateJob(ctx context.Context, job *models.TSETradeHistoryJob) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *TradeHistoryStatusRepository) GetJob(ctx context.Context, id uint) (*models.TSETradeHistoryJob, error) {
	job := new(models.TSETradeHistoryJob)
	return job, r.db.WithContext(ctx).First(job, id).Error
}

// TradeHistoryJobFilter selects jobs in the panel; empty fields match everything.
type TradeHistoryJobFilter struct {
	Statuses []consts.TSETradeHistoryJobStatus
	// TSEMarketID selects the jobs of every market and the jobs of this market.
	TSEMarketID uint
}

func (f TradeHistoryJobFilter) apply(tx *gorm.DB) *gorm.DB {
	if len(f.Statuses) > 0 {
		tx = tx.Where("status IN ?", f.Statuses)
	}
	if f.TSEMarketID != 0 {
		tx = tx.Where("all_markets OR tse_market_ids @> ?::jsonb", jsonArray(f.TSEMarketID))
	}
	return tx
}

// PageJobs lists jobs, active and newest first.
func (r *TradeHistoryStatusRepository) PageJobs(ctx context.Context, filter TradeHistoryJobFilter, page paginator.Params) ([]*models.TSETradeHistoryJob, int64, error) {
	var total int64
	if err := filter.apply(r.db.WithContext(ctx).Model(&models.TSETradeHistoryJob{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	jobs := make([]*models.TSETradeHistoryJob, 0)
	err := page.Apply(filter.apply(r.db.WithContext(ctx).Model(&models.TSETradeHistoryJob{}))).
		// Order ignores a plain clause.Expr, so the expression is wrapped in OrderBy.
		Order(clause.OrderBy{Expression: clause.Expr{SQL: "CASE status WHEN ? THEN 0 WHEN ? THEN 1 ELSE 2 END, id DESC",
			Vars: []any{consts.TSETradeHistoryJobRunning, consts.TSETradeHistoryJobPending}}}).
		Find(&jobs).Error
	return jobs, total, err
}

// HasActiveAllMarketsJob reports whether a job of every market is queued or running.
func (r *TradeHistoryStatusRepository) HasActiveAllMarketsJob(ctx context.Context) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.TSETradeHistoryJob{}).
		Where("all_markets AND status IN ?", consts.TSETradeHistoryJobActive).Count(&count).Error
	return count > 0, err
}

// ClaimJob marks the oldest pending job as running by worker and returns it,
// or nil when no job is pending. Workers never claim the same job.
func (r *TradeHistoryStatusRepository) ClaimJob(ctx context.Context, worker string) (*models.TSETradeHistoryJob, error) {
	var claimed *models.TSETradeHistoryJob
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		job := new(models.TSETradeHistoryJob)
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ?", consts.TSETradeHistoryJobPending).Order("id ASC").Limit(1).Find(job).Error
		if err != nil || job.ID == 0 {
			return err
		}
		now := time.Now()
		columns := map[string]any{
			"status": consts.TSETradeHistoryJobRunning, "worker": worker, "heartbeat_at": now, "updated_at": now, "error": "",
		}
		if job.StartedAt == nil {
			columns["started_at"] = now
		}
		if err := tx.Model(job).UpdateColumns(columns).Error; err != nil {
			return err
		}
		claimed = job
		return nil
	})
	return claimed, err
}

// CancelJob cancels a pending job, or asks the worker of a running job to stop
// it. It reports whether the job was active.
func (r *TradeHistoryStatusRepository) CancelJob(ctx context.Context, id uint) (bool, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.TSETradeHistoryJob{}).
		Where("id = ? AND status = ?", id, consts.TSETradeHistoryJobPending).
		UpdateColumns(map[string]any{"status": consts.TSETradeHistoryJobCanceled, "finished_at": now, "updated_at": now})
	if result.Error != nil || result.RowsAffected > 0 {
		return result.RowsAffected > 0, result.Error
	}
	result = r.db.WithContext(ctx).Model(&models.TSETradeHistoryJob{}).
		Where("id = ? AND status = ?", id, consts.TSETradeHistoryJobRunning).
		UpdateColumns(map[string]any{"cancel_requested": true, "updated_at": now})
	return result.RowsAffected > 0, result.Error
}

// RequeueStaleJobs queues again the running jobs without a heartbeat since before.
func (r *TradeHistoryStatusRepository) RequeueStaleJobs(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.TSETradeHistoryJob{}).
		Where("status = ? AND (heartbeat_at IS NULL OR heartbeat_at < ?)", consts.TSETradeHistoryJobRunning, before).
		UpdateColumns(map[string]any{"status": consts.TSETradeHistoryJobPending, "updated_at": time.Now()})
	return result.RowsAffected, result.Error
}

// OldestRunningHeartbeat returns the oldest heartbeat of the running jobs; nil
// when no job is running.
func (r *TradeHistoryStatusRepository) OldestRunningHeartbeat(ctx context.Context) (*time.Time, error) {
	var oldest sql.NullTime
	err := r.db.WithContext(ctx).Model(&models.TSETradeHistoryJob{}).
		Where("status = ?", consts.TSETradeHistoryJobRunning).
		Select("MIN(heartbeat_at)").Scan(&oldest).Error
	if err != nil || !oldest.Valid {
		return nil, err
	}
	return &oldest.Time, nil
}

// UpdateOwnedJob updates a job only while worker runs it. owned is false when
// the job was queued again or claimed by another worker; cancelRequested
// reports a cancel request of the panel.
func (r *TradeHistoryStatusRepository) UpdateOwnedJob(ctx context.Context, id uint, worker string, columns map[string]any) (owned, cancelRequested bool, err error) {
	columns["updated_at"] = time.Now()
	var rows []models.TSETradeHistoryJob
	err = r.db.WithContext(ctx).Model(&rows).
		Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}, {Name: "cancel_requested"}}}).
		Where("id = ? AND worker = ? AND status = ?", id, worker, consts.TSETradeHistoryJobRunning).
		UpdateColumns(columns).Error
	if err != nil || len(rows) == 0 {
		return false, false, err
	}
	return true, rows[0].CancelRequested, nil
}
