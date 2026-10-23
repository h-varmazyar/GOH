package repositories

import (
	"context"
	"time"

	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StrategyDefinitionRepository struct{ db *gorm.DB }

func NewStrategyDefinitionRepository(db *gorm.DB) *StrategyDefinitionRepository {
	return &StrategyDefinitionRepository{db: db}
}
func (r *StrategyDefinitionRepository) List(ctx context.Context) ([]models.StrategyDefinition, error) {
	out := make([]models.StrategyDefinition, 0)
	err := r.db.WithContext(ctx).Order("id ASC").Find(&out).Error
	return out, err
}
func (r *StrategyDefinitionRepository) Upsert(ctx context.Context, row *models.StrategyDefinition) error {
	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = time.Now().UTC()
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoUpdates: clause.AssignmentColumns([]string{"version", "enabled", "market", "definition", "updated_at"})}).Create(row).Error
}
func (r *StrategyDefinitionRepository) ChangedSince(ctx context.Context, since time.Time) ([]models.StrategyDefinition, error) {
	out := make([]models.StrategyDefinition, 0)
	err := r.db.WithContext(ctx).Where("updated_at >= ?", since).Order("updated_at ASC, id ASC").Find(&out).Error
	return out, err
}
