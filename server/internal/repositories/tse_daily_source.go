package repositories

import (
	"context"
	"time"

	"github.com/h-varmazyar/gatec/internal/models"
)

// TSEDailyMarketDataSource combines the listed TSE universe with its stored
// daily price and participant data for the bourse daily analysis service.
type TSEDailyMarketDataSource struct {
	markets *TSEMarketRepository
	prices  *TSEDailyClosingPriceRepository
}

func NewTSEDailyMarketDataSource(markets *TSEMarketRepository, prices *TSEDailyClosingPriceRepository) *TSEDailyMarketDataSource {
	return &TSEDailyMarketDataSource{markets: markets, prices: prices}
}

func (s *TSEDailyMarketDataSource) ListMarkets(ctx context.Context) ([]*models.TSEMarket, error) {
	return s.markets.List(ctx, nil)
}

func (s *TSEDailyMarketDataSource) ListDailyRows(ctx context.Context, from, to time.Time) ([]*models.TSEDailyClosingPrice, error) {
	return s.prices.ListRange(ctx, from, to)
}
