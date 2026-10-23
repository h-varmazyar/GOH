package repositories

import (
	"context"

	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
)

type MarketRepository struct {
	db *gorm.DB
}

func NewMarketRepository(db *gorm.DB) *MarketRepository {
	return &MarketRepository{db}
}

func (r *MarketRepository) Create(ctx context.Context, market *models.Market) error {
	return r.db.WithContext(ctx).Create(market).Error
}

func (r *MarketRepository) Update(ctx context.Context, id uint, columns map[string]any) error {
	return r.db.WithContext(ctx).Model(&models.Market{}).Where("id = ?", id).Updates(columns).Error
}

// func (r *MarketRepository) List(ctx context.Context, providerID uint) ([]models.Market, error) {
// 	markets := make([]models.Market, 0)
// 	err := r.db.WithContext(ctx).
// 		Model(&models.Market{}).
// 		Where("provider_id = ?", providerID).
// 		Preload("TimeFrames").
// 		Find(&markets).
// 		Error

// 	if err != nil {
// 		return nil, err
// 	}

// 	return markets, nil
// }

func (r *MarketRepository) List(ctx context.Context, conditions map[string]any) ([]*models.Market, error) {
	markets := make([]*models.Market, 0)
	tx := r.db.WithContext(ctx).Model(&models.Market{}).Preload("TimeFrames").Preload("Provider")
	for condition, value := range conditions {
		tx = tx.Where(condition, value)
	}
	return markets, tx.Find(&markets).Error
}

func (r *MarketRepository) UpdateStatus(ctx context.Context, id uint, status consts.MarketStatus) error {
	return r.db.WithContext(ctx).Model(&models.Market{}).Where("id = ?", id).UpdateColumn("status", status).Error
}

func (r *MarketRepository) ReturnByID(ctx context.Context, id uint) (*models.Market, error) {
	market := new(models.Market)
	return market, r.db.WithContext(ctx).Model(&models.Market{}).Where("id = ?", id).Preload("Provider").First(market).Error
}
