// Package tradehistory collects the trade history of TSE markets day by day.
// The trading days of a market are its daily closing price rows, stored by the
// tse-market-data-sync worker; the result of each trade history fetch and the
// totals of the received trades are recorded in the same row, so the two
// sources can be compared and missing, empty and failed days fetched again.
//
// The panel API uses the service directly for quick actions, such as fetching
// one day of one market. Long collections are queued as jobs that the
// tse-trade-history worker claims and runs with the same service; the service
// never calls the worker, which keeps the import graph acyclic:
//
//	api -> services/tradehistory <- worker
package tradehistory

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/adapters/stock/tse"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrNoAdapter    = errors.New("no active TSE provider")
	ErrNoStorage    = errors.New("trade history storage is not configured")
)

const dateLayout = "2006-01-02"

type Service struct {
	cfg        *configs.TSETradeHistory
	adapter    *tse.Adapter
	marketRepo *repositories.TSEMarketRepository
	statusRepo *repositories.TradeHistoryStatusRepository
	tradeRepo  *repositories.TradeHistoryRepository
	location   *time.Location
	logger     *logrus.Entry
}

func NewService(
	cfg *configs.TSETradeHistory,
	adapter *tse.Adapter,
	marketRepo *repositories.TSEMarketRepository,
	statusRepo *repositories.TradeHistoryStatusRepository,
	tradeRepo *repositories.TradeHistoryRepository,
) *Service {
	if cfg == nil {
		cfg = &configs.TSETradeHistory{}
	}
	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = 1
	}
	location, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		logrus.WithError(err).Warn("load Asia/Tehran timezone; using a fixed +03:30 offset")
		location = time.FixedZone("Asia/Tehran", 3*3600+1800)
	}
	return &Service{
		cfg:        cfg,
		adapter:    adapter,
		marketRepo: marketRepo,
		statusRepo: statusRepo,
		tradeRepo:  tradeRepo,
		location:   location,
		logger:     logrus.WithField("service", "tradehistory"),
	}
}

// Config returns the configuration the service runs with.
func (s *Service) Config() *configs.TSETradeHistory { return s.cfg }

// Today is the current date in Tehran, as UTC midnight like the stored days.
func (s *Service) Today() time.Time {
	return dateOnly(time.Now().In(s.location))
}

// dateOnly is the calendar date of t as UTC midnight.
func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// ParseDate parses a YYYY-MM-DD date.
func ParseDate(value string) (time.Time, error) {
	date, err := time.ParseInLocation(dateLayout, value, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: date %q must use YYYY-MM-DD", ErrInvalidInput, value)
	}
	return date, nil
}

// CollectOptions bounds the requests of a day.
type CollectOptions struct {
	// JobID is recorded with the day result.
	JobID *uint
	// MaxAttempts and RetryDelay default to the configuration.
	MaxAttempts int
	RetryDelay  time.Duration
}

// DayResult is the result of fetching one day.
type DayResult struct {
	Date   time.Time
	Status consts.TSETradeDayStatus
	// TradeCount counts the received trades, canceled included.
	TradeCount    int64
	CanceledCount int64
	Volume        int64
	NetVolume     int64
	NetValue      int64
	Attempts      int
	// Err is the fetch or store failure of a failed day.
	Err error
}

// CollectDay fetches the trades of market on a trading day, stores them in
// ClickHouse and records the result and the trade totals in the daily closing
// price row of the day. A failed fetch is recorded and returned in the result;
// the returned error is set only when ctx is done or the result could not be recorded.
func (s *Service) CollectDay(ctx context.Context, market *models.TSEMarket, day repositories.TradeDayRef, opts CollectOptions) (DayResult, error) {
	date := dateOnly(day.TradeDate.UTC())
	result := DayResult{Date: date}
	if s.adapter == nil {
		return result, ErrNoAdapter
	}
	if s.tradeRepo == nil || s.statusRepo == nil {
		return result, ErrNoStorage
	}
	maxAttempts, retryDelay := opts.MaxAttempts, opts.RetryDelay
	if maxAttempts < 1 {
		maxAttempts = s.cfg.MaxAttempts
	}
	if retryDelay <= 0 {
		retryDelay = s.cfg.RetryDelay
	}

	trades, attempts, err := s.fetch(ctx, market, date, maxAttempts, retryDelay)
	result.Attempts = attempts
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	switch {
	case err != nil:
		result.Status, result.Err = consts.TSETradeDayFailed, err
	case len(trades) == 0:
		result.Status = consts.TSETradeDayEmpty
	default:
		if err := s.saveTrades(ctx, market, date, trades); err != nil {
			if ctx.Err() != nil {
				return result, ctx.Err()
			}
			result.Status, result.Err = consts.TSETradeDayFailed, err
		} else {
			result.Status = consts.TSETradeDayFetched
			result.addTotals(trades)
		}
	}
	return result, s.saveResult(ctx, day.ID, result, opts.JobID)
}

// addTotals counts the trades and sums their volume and value.
func (r *DayResult) addTotals(trades []tse.TradeHistory) {
	for _, trade := range trades {
		quantity := int64(trade.Quantity)
		r.TradeCount++
		r.Volume += quantity
		if trade.Canceled != 0 {
			r.CanceledCount++
			continue
		}
		r.NetVolume += quantity
		r.NetValue += int64(trade.Price) * quantity
	}
}

func (s *Service) saveResult(ctx context.Context, id uint, result DayResult, jobID *uint) error {
	row := repositories.TradeDayResult{
		Status: result.Status, TradeCount: result.TradeCount, CanceledCount: result.CanceledCount,
		Volume: result.Volume, NetVolume: result.NetVolume, NetValue: result.NetValue,
		Attempts: result.Attempts, JobID: jobID, FetchedAt: time.Now(),
	}
	if result.Err != nil {
		row.Error = result.Err.Error()
	}
	if err := s.statusRepo.SaveDayResult(ctx, id, row); err != nil {
		return fmt.Errorf("record trade history of %s: %w", result.Date.Format(dateLayout), err)
	}
	return nil
}

func (s *Service) fetch(ctx context.Context, market *models.TSEMarket, date time.Time, maxAttempts int, retryDelay time.Duration) ([]tse.TradeHistory, int, error) {
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var trades []tse.TradeHistory
		trades, err = s.adapter.FetchTradeHistory(ctx, market.InsCode, date.Format("20060102"))
		if err == nil {
			return trades, attempt, nil
		}
		if ctx.Err() != nil || attempt == maxAttempts {
			return nil, attempt, err
		}
		s.logger.WithError(err).WithFields(logrus.Fields{
			"ins_code": market.InsCode, "date": date.Format(dateLayout), "attempt": attempt,
		}).Warn("retrying TSE trade history fetch")
		timer := time.NewTimer(retryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, attempt, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, maxAttempts, err
}

func (s *Service) saveTrades(ctx context.Context, market *models.TSEMarket, date time.Time, trades []tse.TradeHistory) error {
	version := uint64(time.Now().UnixNano())
	rows := make([]models.TSETradeHistory, len(trades))
	for index, trade := range trades {
		rows[index] = models.TSETradeHistory{
			MarketID:   uint64(market.MarketID),
			InsCode:    market.InsCode,
			TradeDate:  date.Format(dateLayout),
			TradeIndex: uint32(index),
			Canceled:   trade.Canceled != 0,
			TradeTime:  int64(trade.TradeTime),
			Price:      int64(trade.Price),
			Quantity:   int64(trade.Quantity),
			Version:    version,
		}
	}
	if err := s.tradeRepo.Insert(ctx, rows); err != nil {
		return fmt.Errorf("store trades: %w", err)
	}
	return nil
}

func (s *Service) getMarket(ctx context.Context, id uint) (*models.TSEMarket, error) {
	market, err := s.marketRepo.GetByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: TSE market %d", ErrNotFound, id)
	}
	return market, err
}
