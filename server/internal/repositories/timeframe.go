package repositories

import (
	"context"

	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
)

type TimeframeRepository struct {
	db *gorm.DB
}

func NewTimeframeRepository(db *gorm.DB) *TimeframeRepository {
	return &TimeframeRepository{db: db}
}

func (r *TimeframeRepository) List(ctx context.Context, conditions map[string]any) ([]models.Timeframe, error) {
	timeframes := make([]models.Timeframe, 0)
	tx := r.db.WithContext(ctx).Model(&models.Timeframe{})
	for condition, value := range conditions {
		tx.Where(condition, value)
	}
	return timeframes, tx.Find(&timeframes).Error
}

func (r *TimeframeRepository) ReturnByID(ctx context.Context, id uint) (*models.Timeframe, error) {
	tf := new(models.Timeframe)

	return tf, r.db.Model(&models.Timeframe{}).Where("id = ?", id).First(&tf).Error
}
