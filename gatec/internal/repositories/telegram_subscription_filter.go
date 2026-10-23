package repositories

import (
	"context"

	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/pkg/paginator"
	"gorm.io/gorm"
)

type TelegramSubscriptionFilterRepository struct {
	db *gorm.DB
}

func NewTelegramSubscriptionFilterRepository(db *gorm.DB) *TelegramSubscriptionFilterRepository {
	return &TelegramSubscriptionFilterRepository{db: db}
}

func (r *TelegramSubscriptionFilterRepository) Create(ctx context.Context, filter *models.TelegramSubscriptionFilter) error {
	return r.db.WithContext(ctx).Create(filter).Error
}

func (r *TelegramSubscriptionFilterRepository) List(ctx context.Context, pg paginator.Params) ([]models.TelegramSubscriptionFilter, int64, error) {
	var filters []models.TelegramSubscriptionFilter
	var total int64

	r.db.WithContext(ctx).Model(&models.TelegramSubscriptionFilter{}).Count(&total)

	err := pg.Apply(r.db.WithContext(ctx).Order("created_at desc")).Find(&filters).Error
	return filters, total, err
}

func (r *TelegramSubscriptionFilterRepository) ListActive(ctx context.Context) ([]models.TelegramSubscriptionFilter, error) {
	var filters []models.TelegramSubscriptionFilter
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&filters).Error
	return filters, err
}

func (r *TelegramSubscriptionFilterRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.TelegramSubscriptionFilter{}, id).Error
}
