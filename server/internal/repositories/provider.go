package repositories

import (
	"context"

	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
)

type ProviderRepository struct {
	db *gorm.DB
}

func NewProviderRepository(db *gorm.DB) *ProviderRepository {
	return &ProviderRepository{db: db}
}

func (r *ProviderRepository) Active(ctx context.Context) ([]models.Provider, error) {
	providers := make([]models.Provider, 0)

	err := r.db.WithContext(ctx).
		Model(&models.Provider{}).
		Where("status = ?", consts.ProviderStatusEnable).
		Find(&providers).
		Error

	if err != nil {
		return nil, err
	}

	return providers, nil
}

func (r *ProviderRepository) ActiveStock(ctx context.Context) ([]models.Provider, error) {
	providers := make([]models.Provider, 0)
	err := r.db.WithContext(ctx).
		Model(&models.Provider{}).
		Where("status = ?", consts.ProviderStatusEnable).
		Where("name = ?", consts.ProviderTSE).
		Find(&providers).Error
	if err != nil {
		return nil, err
	}
	return providers, nil
}

func (r *ProviderRepository) ReturnByName(ctx context.Context, name consts.Provider) (*models.Provider, error) {
	provider := new(models.Provider)
	return provider, r.db.WithContext(ctx).Model(&models.Provider{}).Where("name = ?", name).First(provider).Error
}
