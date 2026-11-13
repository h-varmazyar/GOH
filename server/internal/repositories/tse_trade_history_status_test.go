package repositories

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/pkg/paginator"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var errRollback = errors.New("rollback")

// withTestSchema runs fn in a transaction on a throwaway schema of the Postgres
// database in GATEC_TEST_POSTGRES_DSN; the transaction is always rolled back.
func withTestSchema(t *testing.T, fn func(tx *gorm.DB)) {
	dsn := os.Getenv("GATEC_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("GATEC_TEST_POSTGRES_DSN is not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		// The throwaway schema has only the tables the queries use.
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open Postgres: %v", err)
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		for _, statement := range []string{
			"CREATE SCHEMA trade_history_test",
			"SET LOCAL search_path TO trade_history_test",
			`CREATE TABLE tse_markets (id bigserial PRIMARY KEY, market_id bigint, ins_code text, label text, name text,
				delisted boolean DEFAULT false, data_collection_enabled boolean DEFAULT true)`,
			"CREATE TABLE candles (id bigserial PRIMARY KEY, market_id bigint, time bigint)",
			`CREATE TABLE tse_daily_closing_prices (id bigserial PRIMARY KEY, market_id bigint, tse_market_id bigint, candle_id bigint,
				trade_count bigint, volume double precision, value bigint)`,
		} {
			if err := tx.Exec(statement).Error; err != nil {
				t.Fatalf("%s: %v", statement, err)
			}
		}
		// The trade history columns are added the way the migration adds them to the existing table.
		if err := tx.AutoMigrate(&models.TSEDailyClosingPrice{}, &models.TSETradeHistoryJob{}); err != nil {
			t.Fatalf("migrate: %v", err)
		}
		fn(tx)
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatalf("transaction: %v", err)
	}
}

func date(value string) time.Time {
	parsed, _ := time.ParseInLocation(sqlDate, value, time.UTC)
	return parsed
}

func TestTradeHistoryDaysAndGaps(t *testing.T) {
	withTestSchema(t, func(tx *gorm.DB) {
		ctx := context.Background()
		repo := NewTradeHistoryStatusRepository(tx)
		mustExec := func(query string, args ...any) {
			if err := tx.Exec(query, args...).Error; err != nil {
				t.Fatalf("%s: %v", query, err)
			}
		}
		// Market 1 has four trading days; market 2 has none.
		mustExec("INSERT INTO tse_markets (id, market_id, ins_code, label, name) VALUES (1, 10, '111', 'AAA', 'A co'), (2, 20, '222', 'BBB', 'B co')")
		for i, day := range []string{"2024-01-01", "2024-01-02", "2024-01-03", "2024-01-06"} {
			mustExec("INSERT INTO candles (id, market_id, time) VALUES (?, 10, ?)", i+1, date(day).Unix())
			mustExec("INSERT INTO tse_daily_closing_prices (id, market_id, tse_market_id, candle_id, trade_count, volume, value) VALUES (?, 10, 1, ?, 10, 1000, 5000)", i+1, i+1)
		}

		all, err := repo.PlanDays(ctx, 1, nil, nil, nil, []consts.TSETradeDayStatus{consts.TSETradeDayMissing})
		if err != nil || len(all) != 4 || all[0].TradeDate.UTC().Format(sqlDate) != "2024-01-06" {
			t.Fatalf("plan missing days = %+v, %v", all, err)
		}
		save := func(id uint, result TradeDayResult) {
			result.FetchedAt = time.Now()
			if err := repo.SaveDayResult(ctx, id, result); err != nil {
				t.Fatalf("save day: %v", err)
			}
		}
		// Day 1 matches its closing price once a canceled trade is left out.
		save(1, TradeDayResult{Status: consts.TSETradeDayFetched, TradeCount: 11, CanceledCount: 1, Volume: 1100, NetVolume: 1000, Attempts: 1})
		// Day 2 has fewer trades than its closing price.
		save(2, TradeDayResult{Status: consts.TSETradeDayFetched, TradeCount: 9, Volume: 900, NetVolume: 900, Attempts: 1})
		save(3, TradeDayResult{Status: consts.TSETradeDayFailed, Error: "boom", Attempts: 2})
		save(3, TradeDayResult{Status: consts.TSETradeDayFailed, Error: "boom again", Attempts: 1})
		if err := repo.SaveDayResult(ctx, 99, TradeDayResult{Status: consts.TSETradeDayEmpty}); err == nil {
			t.Fatal("saving a result of an unknown day should fail")
		}

		counts, err := repo.CountDays(ctx, 1)
		if err != nil {
			t.Fatalf("count days: %v", err)
		}
		if counts.TotalDays != 4 || counts.FetchedDays != 2 || counts.FailedDays != 1 || counts.MissingDays != 1 || counts.MismatchedDays != 1 ||
			counts.FirstDate.UTC().Format(sqlDate) != "2024-01-01" || counts.LastDate.UTC().Format(sqlDate) != "2024-01-06" {
			t.Fatalf("counts = %+v", counts)
		}

		days, total, err := repo.PageDays(ctx, 1, TradeDayFilter{}, paginator.New(1, 10))
		if err != nil || total != 4 || len(days) != 4 {
			t.Fatalf("page days = %d, %v", total, err)
		}
		if day := days[1]; day.TradeDate.UTC().Format(sqlDate) != "2024-01-03" || day.HistoryAttempts != 3 || day.HistoryError != "boom again" {
			t.Fatalf("failed day = %+v", day)
		}
		if day := days[3]; !(day.HistoryNetVolume == 1000 && day.HistoryVolume == 1100 && !day.Mismatched) {
			t.Fatalf("matching day = %+v", day)
		}
		mismatched, total, err := repo.PageDays(ctx, 1, TradeDayFilter{Status: consts.TSETradeDayMismatched}, paginator.New(1, 10))
		if err != nil || total != 1 || mismatched[0].ID != 2 || !mismatched[0].Mismatched {
			t.Fatalf("mismatched days = %+v, %d, %v", mismatched, total, err)
		}
		from := date("2024-01-02")
		if _, total, _ := repo.PageDays(ctx, 1, TradeDayFilter{From: &from}, paginator.New(1, 10)); total != 3 {
			t.Fatalf("days from %s = %d, want 3", from.Format(sqlDate), total)
		}

		retry, err := repo.PlanDays(ctx, 1, nil, nil, nil, []consts.TSETradeDayStatus{consts.TSETradeDayMissing, consts.TSETradeDayFailed})
		if err != nil || len(retry) != 2 || retry[0].ID != 4 || retry[1].ID != 3 {
			t.Fatalf("plan missing and failed days = %+v, %v", retry, err)
		}
		picked, err := repo.PlanDays(ctx, 1, []time.Time{date("2024-01-02"), date("2024-01-05")}, nil, nil, nil)
		if err != nil || len(picked) != 1 || picked[0].ID != 2 {
			t.Fatalf("plan picked days = %+v, %v", picked, err)
		}
		found, err := repo.FindDay(ctx, 1, date("2024-01-02"))
		if err != nil || found == nil || found.ID != 2 || found.HistoryStatus == nil || *found.HistoryStatus != consts.TSETradeDayFetched {
			t.Fatalf("find day = %+v, %v", found, err)
		}
		if found, err := repo.FindDay(ctx, 1, date("2024-01-05")); err != nil || found != nil {
			t.Fatalf("find a day without closing price = %+v, %v", found, err)
		}

		gaps, err := repo.PageGaps(ctx, TradeGapFilter{}, paginator.New(1, 10))
		if err != nil || len(gaps) != 1 || gaps[0].TSEMarketID != 1 || gaps[0].MissingDays != 1 || gaps[0].MismatchedDays != 1 ||
			gaps[0].LastFetchedAt == nil || gaps[0].FirstDate == nil || gaps[0].LastDate.UTC().Format(sqlDate) != "2024-01-06" {
			t.Fatalf("page gaps = %+v, %v", gaps, err)
		}
		totals, err := repo.GapTotals(ctx, TradeGapFilter{Problem: TradeGapAll})
		if err != nil || totals.Markets != 1 || totals.TotalDays != 4 || totals.MismatchedDays != 1 {
			t.Fatalf("totals = %+v, %v", totals, err)
		}
		ids, err := repo.GapMarketIDs(ctx, TradeGapFilter{Problem: TradeGapMismatched})
		if err != nil || len(ids) != 1 || ids[0] != 1 {
			t.Fatalf("mismatched market ids = %v, %v", ids, err)
		}
		if ids, _ := repo.GapMarketIDs(ctx, TradeGapFilter{Problem: TradeGapEmpty}); len(ids) != 0 {
			t.Fatalf("empty market ids = %v, want none", ids)
		}
		withoutDays, err := repo.CountWithoutDays(ctx, TradeGapFilter{})
		if err != nil || withoutDays != 1 {
			t.Fatalf("without days = %d, %v", withoutDays, err)
		}
	})
}

func TestTradeHistoryJobs(t *testing.T) {
	withTestSchema(t, func(tx *gorm.DB) {
		ctx := context.Background()
		repo := NewTradeHistoryStatusRepository(tx)
		first := &models.TSETradeHistoryJob{Status: consts.TSETradeHistoryJobPending, TSEMarketIDs: []uint{3, 7}, DayStatuses: []consts.TSETradeDayStatus{consts.TSETradeDayMissing}}
		second := &models.TSETradeHistoryJob{Status: consts.TSETradeHistoryJobPending, AllMarkets: true}
		for _, job := range []*models.TSETradeHistoryJob{first, second} {
			if err := repo.CreateJob(ctx, job); err != nil {
				t.Fatalf("create job: %v", err)
			}
		}
		claimed, err := repo.ClaimJob(ctx, "test")
		if err != nil || claimed == nil || claimed.ID != first.ID {
			t.Fatalf("claim = %+v, %v", claimed, err)
		}
		jobs, total, err := repo.PageJobs(ctx, TradeHistoryJobFilter{TSEMarketID: 7, Statuses: consts.TSETradeHistoryJobActive}, paginator.New(1, 10))
		if err != nil || total != 2 || jobs[0].Status != consts.TSETradeHistoryJobRunning {
			t.Fatalf("jobs of market 7 = %d, %v", total, err)
		}
		if _, total, _ := repo.PageJobs(ctx, TradeHistoryJobFilter{TSEMarketID: 5}, paginator.New(1, 10)); total != 1 {
			t.Fatalf("jobs of market 5 = %d, want the all markets job only", total)
		}
		if active, err := repo.HasActiveAllMarketsJob(ctx); err != nil || !active {
			t.Fatalf("active all markets job = %t, %v", active, err)
		}
		// A pending job is canceled at once; a running one is asked to stop.
		if ok, err := repo.CancelJob(ctx, second.ID); err != nil || !ok {
			t.Fatalf("cancel pending = %t, %v", ok, err)
		}
		if job, _ := repo.GetJob(ctx, second.ID); job.Status != consts.TSETradeHistoryJobCanceled {
			t.Fatalf("canceled job status = %s", job.Status)
		}
		if ok, err := repo.CancelJob(ctx, first.ID); err != nil || !ok {
			t.Fatalf("cancel running = %t, %v", ok, err)
		}
		if job, _ := repo.GetJob(ctx, first.ID); job.Status != consts.TSETradeHistoryJobRunning || !job.CancelRequested {
			t.Fatalf("running job after cancel = %s, %t", job.Status, job.CancelRequested)
		}
		if oldest, err := repo.OldestRunningHeartbeat(ctx); err != nil || oldest == nil {
			t.Fatalf("oldest heartbeat = %v, %v", oldest, err)
		}
		// Only the worker running the job stores its progress; the cancel request is returned.
		if owned, cancelRequested, err := repo.UpdateOwnedJob(ctx, first.ID, "test", map[string]any{"done_markets": 3}); err != nil || !owned || !cancelRequested {
			t.Fatalf("update by owner = %t, %t, %v", owned, cancelRequested, err)
		}
		if owned, _, err := repo.UpdateOwnedJob(ctx, first.ID, "other", map[string]any{"done_markets": 9}); err != nil || owned {
			t.Fatalf("update by other worker = %t, %v", owned, err)
		}
		if job, _ := repo.GetJob(ctx, first.ID); job.DoneMarkets != 3 {
			t.Fatalf("done markets = %d, want 3", job.DoneMarkets)
		}
		if requeued, err := repo.RequeueStaleJobs(ctx, time.Now().Add(time.Minute)); err != nil || requeued != 1 {
			t.Fatalf("requeued = %d, %v", requeued, err)
		}
		if oldest, err := repo.OldestRunningHeartbeat(ctx); err != nil || oldest != nil {
			t.Fatalf("oldest heartbeat with no running job = %v, %v", oldest, err)
		}
		// The worker whose job was queued again no longer owns it.
		if owned, _, err := repo.UpdateOwnedJob(ctx, first.ID, "test", map[string]any{"done_markets": 4}); err != nil || owned {
			t.Fatalf("update of requeued job = %t, %v", owned, err)
		}
		if claimed, err := repo.ClaimJob(ctx, "test"); err != nil || claimed == nil || claimed.ID != first.ID {
			t.Fatalf("claim requeued = %+v, %v", claimed, err)
		}
		if claimed, err := repo.ClaimJob(ctx, "test"); err != nil || claimed != nil {
			t.Fatalf("claim with no pending job = %+v, %v", claimed, err)
		}
	})
}
