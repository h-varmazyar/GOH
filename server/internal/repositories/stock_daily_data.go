package repositories

import (
	"context"

	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StockDailyDataRepository struct {
	db *gorm.DB
}

func NewStockDailyDataRepository(db *gorm.DB) *StockDailyDataRepository {
	return &StockDailyDataRepository{db: db}
}

// Upsert inserts or updates a stock daily data record based on market_id + date
func (r *StockDailyDataRepository) Upsert(ctx context.Context, data *models.StockDailyData) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "market_id"}, {Name: "date"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "trade_count", "volume", "value", "yesterday", "first", "last", "last_change", "last_percent", "close", "close_change", "close_percent", "low", "high", "fetched_at", "updated_at"}),
		}).
		Create(data).Error
}

// BulkUpsert inserts or updates multiple stock daily data records
func (r *StockDailyDataRepository) BulkUpsert(ctx context.Context, records []*models.StockDailyData) error {
	if len(records) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "market_id"}, {Name: "date"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "trade_count", "volume", "value", "yesterday", "first", "last", "last_change", "last_percent", "close", "close_change", "close_percent", "low", "high", "fetched_at", "updated_at"}),
		}).
		CreateInBatches(records, 100).Error
}

// GetByMarketAndDate retrieves stock data for a specific market and date
func (r *StockDailyDataRepository) GetByMarketAndDate(ctx context.Context, marketID uint, date string) (*models.StockDailyData, error) {
	data := new(models.StockDailyData)
	err := r.db.WithContext(ctx).
		Where("market_id = ? AND date = ?", marketID, date).
		First(data).Error
	return data, err
}

// GetByDate retrieves all stock data for a specific date
func (r *StockDailyDataRepository) GetByDate(ctx context.Context, date string) ([]*models.StockDailyData, error) {
	var records []*models.StockDailyData
	err := r.db.WithContext(ctx).
		Where("date = ?", date).
		Preload("Market").
		Find(&records).Error
	return records, err
}
