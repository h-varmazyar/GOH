package repositories

import (
	"context"

	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
)

type StrategyRepository struct {
	db *gorm.DB
}

func NewStrategyRepository(db *gorm.DB) *StrategyRepository {
	return &StrategyRepository{db}
}

func (r *StrategyRepository) List(ctx context.Context, userID uint) ([]*models.Strategy, error) {
	strategies := make([]*models.Strategy, 0)

	err := r.db.WithContext(ctx).Model(&models.Strategy{}).Where("user_id = ?", userID).Find(&strategies).Error
	if err != nil {
		return nil, err
	}
	return strategies, nil
}
