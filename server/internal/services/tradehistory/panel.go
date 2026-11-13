package tradehistory

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/pkg/paginator"
)

// panelFetchTimeout bounds a day fetched from the panel; the adapter otherwise
// retries a failing TSETMC for minutes.
const panelFetchTimeout = 45 * time.Second

// MarketSummary counts the trading days of a market by trade history status.
type MarketSummary struct {
	Market *models.TSEMarket
	repositories.TradeDayCounts
	ActiveJobs []*models.TSETradeHistoryJob
}

func (s *Service) MarketSummary(ctx context.Context, tseMarketID uint) (*MarketSummary, error) {
	market, err := s.getMarket(ctx, tseMarketID)
	if err != nil {
		return nil, err
	}
	summary := &MarketSummary{Market: market}
	if summary.TradeDayCounts, err = s.statusRepo.CountDays(ctx, market.ID); err != nil {
		return nil, err
	}
	summary.ActiveJobs, _, err = s.statusRepo.PageJobs(ctx, repositories.TradeHistoryJobFilter{
		Statuses: consts.TSETradeHistoryJobActive, TSEMarketID: market.ID,
	}, paginator.New(1, 20))
	return summary, err
}

// ListDays lists the trading days of a market, newest first, with their closing
// price and trade history figures.
func (s *Service) ListDays(ctx context.Context, tseMarketID uint, filter repositories.TradeDayFilter, page paginator.Params) ([]repositories.TradeDay, int64, error) {
	if filter.Status != "" && !filter.Status.Valid() {
		return nil, 0, fmt.Errorf("%w: status %q", ErrInvalidInput, filter.Status)
	}
	if _, err := s.getMarket(ctx, tseMarketID); err != nil {
		return nil, 0, err
	}
	return s.statusRepo.PageDays(ctx, tseMarketID, filter, page)
}

// FetchDay fetches the trade history of one trading day of a market right away, for the panel.
func (s *Service) FetchDay(ctx context.Context, tseMarketID uint, date time.Time) (DayResult, error) {
	date = dateOnly(date)
	result := DayResult{Date: date}
	market, err := s.getMarket(ctx, tseMarketID)
	if err != nil {
		return result, err
	}
	if market.InsCode == "" {
		return result, fmt.Errorf("%w: the market has no insCode", ErrInvalidInput)
	}
	day, err := s.statusRepo.FindDay(ctx, market.ID, date)
	if err != nil {
		return result, err
	}
	if day == nil {
		return result, fmt.Errorf("%w: no daily closing price of %s on %s; the tse-market-data-sync worker stores the trading days",
			ErrNotFound, market.Label, date.Format(dateLayout))
	}
	// The result is recorded even if the client leaves before the fetch ends.
	fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), panelFetchTimeout)
	defer cancel()
	result, err = s.CollectDay(fetchCtx, market, *day, CollectOptions{MaxAttempts: 1})
	if errors.Is(err, context.DeadlineExceeded) {
		// The fetch timed out before a result was recorded; record it as failed.
		result.Status, result.Err = consts.TSETradeDayFailed, fmt.Errorf("fetch timed out after %s", panelFetchTimeout)
		result.Attempts = max(result.Attempts, 1)
		err = s.saveResult(context.WithoutCancel(ctx), day.ID, result, nil)
	}
	return result, err
}

// GapsPage is a page of markets with the counts of their trading days.
type GapsPage struct {
	Gaps   []repositories.TradeGap
	Total  int64
	Totals repositories.TradeGapTotals
	// WithoutDays counts the markets without daily closing prices, which have no
	// trading days to fetch yet.
	WithoutDays int64
}

func (s *Service) ListGaps(ctx context.Context, filter repositories.TradeGapFilter, page paginator.Params) (*GapsPage, error) {
	result := &GapsPage{Gaps: []repositories.TradeGap{}}
	var err error
	if result.Totals, err = s.statusRepo.GapTotals(ctx, filter); err != nil {
		return nil, err
	}
	result.Total = result.Totals.Markets
	if result.Total > int64(page.Offset()) {
		if result.Gaps, err = s.statusRepo.PageGaps(ctx, filter, page); err != nil {
			return nil, err
		}
	}
	if result.WithoutDays, err = s.statusRepo.CountWithoutDays(ctx, filter); err != nil {
		return nil, err
	}
	return result, nil
}
