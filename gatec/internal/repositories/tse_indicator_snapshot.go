package repositories

import (
	"context"

	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TSEIndicatorSnapshotRepository struct{ db *gorm.DB }

func NewTSEIndicatorSnapshotRepository(db *gorm.DB) *TSEIndicatorSnapshotRepository {
	return &TSEIndicatorSnapshotRepository{db: db}
}

func (r *TSEIndicatorSnapshotRepository) Upsert(ctx context.Context, snapshot *models.TSEIndicatorSnapshot) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "market_id"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"provider_id", "results", "updated_at"}),
	}).Create(snapshot).Error
}
