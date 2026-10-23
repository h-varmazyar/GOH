package repositories

import (
	"context"
	"time"

	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DynamicConfigRepository struct {
	db *gorm.DB
}

func NewDynamicConfigRepository(db *gorm.DB) *DynamicConfigRepository {
	return &DynamicConfigRepository{db: db}
}

func (r *DynamicConfigRepository) Get(ctx context.Context, section, key string) (*models.DynamicConfig, error) {
	var config models.DynamicConfig
	if err := r.db.WithContext(ctx).Where("section = ? AND key = ?", section, key).First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *DynamicConfigRepository) ListBySections(ctx context.Context, sections []string) ([]models.DynamicConfig, error) {
	var configs []models.DynamicConfig
	err := r.db.WithContext(ctx).Where("section IN ?", sections).Order("section, key").Find(&configs).Error
	return configs, err
}

func (r *DynamicConfigRepository) EnsureMany(ctx context.Context, values map[string]map[string]string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for section, entries := range values {
			for key, value := range entries {
				item := &models.DynamicConfig{Section: section, Key: key, Value: value}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(item).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *DynamicConfigRepository) UpsertMany(ctx context.Context, section string, values map[string]string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for key, value := range values {
			item := &models.DynamicConfig{Section: section, Key: key, Value: value}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "section"}, {Name: "key"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"value":      value,
					"updated_at": time.Now(),
				}),
			}).Create(item).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
