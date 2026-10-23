package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TSEDailyClosingPriceRepository struct{ db *gorm.DB }

func NewTSEDailyClosingPriceRepository(db *gorm.DB) *TSEDailyClosingPriceRepository {
	return &TSEDailyClosingPriceRepository{db: db}
}

func (r *TSEDailyClosingPriceRepository) UpsertPrices(ctx context.Context, prices []*models.TSEDailyClosingPrice) error {
	for _, price := range prices {
		if price.MarketID == 0 || price.TSEMarketID == 0 {
			return fmt.Errorf("market_id and tse_market_id are required")
		}
		if price.Candle.MarketID != price.MarketID {
			return fmt.Errorf("candle market_id %d does not match daily closing price market_id %d", price.Candle.MarketID, price.MarketID)
		}
		if err := r.db.WithContext(ctx).Model(&models.Candle{}).
			Where("time = ?", price.Candle.Time).
			Where("market_id = ?", price.MarketID).
			Where("timeframe_id = ?", price.Candle.TimeframeID).
			FirstOrCreate(&price.Candle).Error; err != nil {
			return fmt.Errorf("find or create candle for market %d at %d: %w", price.MarketID, price.Candle.Time, err)
		}

		price.CandleID = price.Candle.ID
		price.Candle = models.Candle{}
		if err := r.db.WithContext(ctx).Omit("Candle", "Market", "TSEMarket").
			Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "tse_market_id"}, {Name: "candle_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"market_id", "trade_count", "volume", "value", "first", "yesterday", "last",
					"last_change", "last_change_percent", "closing", "close_change", "close_change_percent",
					"low", "high", "instrument_state_code", "buy_individual_volume", "buy_individual_value",
					"buy_individual_count", "buy_institutional_volume", "buy_institutional_value",
					"buy_institutional_count", "sell_individual_volume", "sell_individual_value",
					"sell_individual_count", "sell_institutional_volume", "sell_institutional_value",
					"sell_institutional_count",
				}),
			}).Create(price).Error; err != nil {
			return fmt.Errorf("upsert daily closing price for TSE market %d and candle %d: %w", price.TSEMarketID, price.CandleID, err)
		}
	}
	return nil
}

func (r *TSEDailyClosingPriceRepository) DistinctMarkets(ctx context.Context) ([]int64, error) {
	ids := make([]int64, 0)
	return ids, r.db.WithContext(ctx).Model(&models.TSEDailyClosingPrice{}).
		Where("tse_market_id <> 0").Distinct("tse_market_id").Pluck("tse_market_id", &ids).Error
}

func (r *TSEDailyClosingPriceRepository) MarketsWithPrices(ctx context.Context, from, to time.Time) ([]uint, error) {
	var ids []uint
	tx := r.db.WithContext(ctx).Model(&models.TSEDailyClosingPrice{}).
		Joins("JOIN candles ON candles.id = tse_daily_closing_prices.candle_id").
		Where("tse_daily_closing_prices.tse_market_id <> 0")
	if !from.IsZero() {
		tx = tx.Where("candles.time >= ?", from.Unix())
	}
	if !to.IsZero() {
		tx = tx.Where("candles.time < ?", to.Unix())
	}
	err := tx.Distinct("tse_daily_closing_prices.tse_market_id").Pluck("tse_daily_closing_prices.tse_market_id", &ids).Error
	return ids, err
}

// ListRange returns daily closing-price rows and their OHLCV candle for the
// inclusive calendar range [from, to]. The caller supplies dates in the market
// timezone; stored timestamps are Unix seconds.
func (r *TSEDailyClosingPriceRepository) ListRange(ctx context.Context, from, to time.Time) ([]*models.TSEDailyClosingPrice, error) {
	var rows []*models.TSEDailyClosingPrice
	query := r.db.WithContext(ctx).
		Model(&models.TSEDailyClosingPrice{}).
		Joins("JOIN candles ON candles.id = tse_daily_closing_prices.candle_id").
		Preload("Candle").
		Preload("TSEMarket").
		Order("candles.time ASC").
		Order("tse_daily_closing_prices.tse_market_id ASC")
	if !from.IsZero() {
		query = query.Where("candles.time >= ?", from.Unix())
	}
	if !to.IsZero() {
		query = query.Where("candles.time < ?", to.AddDate(0, 0, 1).Unix())
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list TSE daily market data: %w", err)
	}
	return rows, nil
}
