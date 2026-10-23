package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
)

type CandleRepository struct {
	db *gorm.DB
}

func NewCandleRepository(db *gorm.DB) *CandleRepository {
	return &CandleRepository{db}
}

func (r *CandleRepository) Create(ctx context.Context, candle *models.Candle) error {
	return r.db.WithContext(ctx).Create(candle).Error
}

func (r *CandleRepository) BatchInsert(ctx context.Context, candles []*models.Candle) error {
	err := r.db.WithContext(ctx).CreateInBatches(&candles, 100).Error
	fmt.Println("batch inserting:", err)

	return err
}

func (r *CandleRepository) List(ctx context.Context, marketID uint, tfID uint, limit int) ([]models.Candle, error) {
	query := r.db.WithContext(ctx).Model(&models.Candle{}).
		Where("market_id = ?", marketID).
		Where("timeframe_id = ?", tfID).
		Where("open <> '0'").
		Where("high <> '0'").
		Where("low <> '0'")
	if limit > 0 {
		query = query.Order("time DESC").Limit(limit)
	} else {
		query = query.Order("time ASC")
	}
	candles := make([]models.Candle, 0)
	if err := query.Find(&candles).Error; err != nil {
		return nil, err
	}
	if limit > 0 {
		for left, right := 0, len(candles)-1; left < right; left, right = left+1, right-1 {
			candles[left], candles[right] = candles[right], candles[left]
		}
	}
	return candles, nil
}

func (r *CandleRepository) Range(ctx context.Context, marketID, tfID uint, from, to time.Time) ([]models.Candle, error) {
	if from.After(to) {
		return []models.Candle{}, nil
	}
	out := make([]models.Candle, 0)
	err := r.db.WithContext(ctx).Where("market_id = ? AND timeframe_id = ? AND time >= ? AND time <= ?", marketID, tfID, from.Unix(), to.Unix()).
		Where("open <> '0' AND high <> '0' AND low <> '0'").Order("time ASC").Find(&out).Error
	return out, err
}
