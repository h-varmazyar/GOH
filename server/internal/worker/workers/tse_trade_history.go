package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/adapters/stock/tse"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/sirupsen/logrus"
)

const (
	tradeHistoryMaxAttempts = 3
	tradeHistoryRetryDelay  = time.Second * 30
)

type TSETradeHistory struct {
	adapter     *tse.Adapter
	marketRepo  *repositories.TSEMarketRepository
	historyRepo *repositories.TradeHistoryRepository
	cfg         *configs.TSETradeHistory
	logger      *logrus.Entry
}

func NewTSETradeHistory(
	adapter *tse.Adapter,
	cfg *configs.TSETradeHistory,
	marketRepo *repositories.TSEMarketRepository,
	historyRepo *repositories.TradeHistoryRepository,
) *TSETradeHistory {
	return &TSETradeHistory{
		adapter:     adapter,
		marketRepo:  marketRepo,
		historyRepo: historyRepo,
		cfg:         cfg,
		logger:      logrus.WithField("worker", "tse-trade-history"),
	}
}

func (w *TSETradeHistory) Name() string { return "tse-trade-history" }

func (w *TSETradeHistory) Stop() error { return nil }

func (w *TSETradeHistory) Start(ctx context.Context) error {
	logrus.Info("starting tse trade history collector")
	markets, err := w.marketRepo.ListDataCollection(ctx)
	if err != nil {
		return fmt.Errorf("list TSE markets: %w", err)
	}
	now := time.Now()
	for _, market := range markets {
		if market.InsCode == "" || market.IPODate == nil || market.IPODate.IsZero() {
			continue
		}
		for t := *market.IPODate; t.Before(now); t = t.AddDate(0, 0, 1) {
			date := t.Format("20060102")
			var history []tse.TradeHistory
			for attempt := 1; attempt <= tradeHistoryMaxAttempts; attempt++ {
				history, err = w.adapter.FetchTradeHistory(ctx, market.InsCode, date)
				if err == nil {
					break
				}
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if attempt == tradeHistoryMaxAttempts {
					return fmt.Errorf(
						"fetch TSE trade history for market %s on %s after %d attempts: %w",
						market.InsCode,
						date,
						attempt,
						err,
					)
				}

				w.logger.WithError(err).WithFields(logrus.Fields{
					"market":  market.InsCode,
					"date":    date,
					"attempt": attempt,
				}).Warn("retrying TSE trade history fetch")

				timer := time.NewTimer(tradeHistoryRetryDelay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return ctx.Err()
				case <-timer.C:
				}
			}
			if len(history) > 0 {
				rows := make([]models.TSETradeHistory, len(history))
				for index, trade := range history {
					rows[index] = models.TSETradeHistory{
						MarketID:   uint64(market.MarketID),
						InsCode:    market.InsCode,
						TradeDate:  t.Format("2006-01-02"),
						TradeIndex: uint32(index),
						Canceled:   trade.Canceled,
						TradeTime:  int64(trade.TradeTime),
						Price:      int64(trade.Price),
						Quantity:   int64(trade.Quantity),
						Version:    uint64(time.Now().UnixNano()),
					}
				}
				if err := w.historyRepo.Insert(ctx, rows); err != nil {
					return fmt.Errorf("save TSE trade history for market %s on %s: %w", market.InsCode, date, err)
				}
			}
		}
	}
	return nil
}
