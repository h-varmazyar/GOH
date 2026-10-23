package repositories

import (
	"context"

	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IndicatorDefinitionRepository struct {
	db *gorm.DB
}

func NewIndicatorDefinitionRepository(db *gorm.DB) *IndicatorDefinitionRepository {
	return &IndicatorDefinitionRepository{db: db}
}

func (r *IndicatorDefinitionRepository) Upsert(ctx context.Context, in *models.IndicatorDefinition) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "fingerprint"}},
		DoUpdates: clause.Assignments(map[string]any{"updated_at": gorm.Expr("NOW()")}),
	}).Create(in).Error
}

func (r *IndicatorDefinitionRepository) List(ctx context.Context) ([]models.IndicatorDefinition, error) {
	out := make([]models.IndicatorDefinition, 0)
	err := r.db.WithContext(ctx).Order("id ASC").Find(&out).Error
	return out, err
}

func (r *IndicatorDefinitionRepository) ListForProviderAndMarket(ctx context.Context, providerID uint, marketID uint64) ([]models.IndicatorDefinition, error) {
	out := make([]models.IndicatorDefinition, 0)
	err := r.db.WithContext(ctx).
		Where("provider_id = ? AND (market_id = ? OR market_id = 0)", providerID, marketID).
		Order("id ASC").
		Find(&out).Error
	return out, err
}

func (r *IndicatorDefinitionRepository) Return(ctx context.Context, id uint) (*models.IndicatorDefinition, error) {
	ind := new(models.IndicatorDefinition)
	return ind, r.db.WithContext(ctx).Where("id = ?", id).First(ind).Error
}
