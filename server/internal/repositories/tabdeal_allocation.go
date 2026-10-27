package repositories

import (
	"context"

	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
)

type TabdealAllocationRepository struct {
	db *gorm.DB
}

func NewTabdealAllocationRepository(db *gorm.DB) *TabdealAllocationRepository {
	return &TabdealAllocationRepository{db: db}
}

func (r *TabdealAllocationRepository) Create(ctx context.Context, allocation *models.TabdealGitAllocation) error {
	return r.db.WithContext(ctx).Model(&models.TabdealGitAllocation{}).Create(allocation).Error
}

func (r *TabdealAllocationRepository) Update(ctx context.Context, allocation *models.TabdealGitAllocation) error {
	return r.db.WithContext(ctx).Model(&models.TabdealGitAllocation{}).Where("id = ?", allocation.ID).Updates(allocation).Error
}

func (r *TabdealAllocationRepository) ReturnByID(ctx context.Context, id uint) (*models.TabdealGitAllocation, error) {
	allocation := new(models.TabdealGitAllocation)
	return allocation, r.db.
		WithContext(ctx).
		Model(&models.TabdealGitAllocation{}).
		Where("id = ?", allocation.ID).
		Preload("Gift").
		First(allocation).
		Error
}
