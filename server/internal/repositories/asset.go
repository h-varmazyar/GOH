package repositories

import (
	"context"

	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
)

type AssetRepository struct {
	db *gorm.DB
}

func NewAssetRepository(db *gorm.DB) *AssetRepository {
	return &AssetRepository{db: db}
}

func (r *AssetRepository) ReturnByLabel(ctx context.Context, label string) (*models.Asset, error) {
	asset := new(models.Asset)
	return asset, r.db.WithContext(ctx).Model(&models.Asset{}).Where("label = ?", label).First(asset).Error
}

func (r *AssetRepository) Create(ctx context.Context, asset *models.Asset) error {
	return r.db.Create(asset).Error
}
