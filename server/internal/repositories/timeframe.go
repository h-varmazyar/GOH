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

func (r *TimeframeRepository) Create(ctx context.Context, timeframe *models.Timeframe) error {
	return r.db.WithContext(ctx).Create(timeframe).Error
}

// ListByProvider returns the timeframes of a provider.
func (r *TimeframeRepository) ListByProvider(ctx context.Context, providerID uint) ([]models.Timeframe, error) {
	timeframes := make([]models.Timeframe, 0)
	err := r.db.WithContext(ctx).Where("provider_id = ?", providerID).Order("duration, id").Find(&timeframes).Error
	return timeframes, err
}
