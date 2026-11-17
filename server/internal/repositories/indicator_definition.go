package repositories

import (
	"context"
	"fmt"

	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/pkg/paginator"
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

// ListForProviderAndMarket returns the definitions that apply to a market:
// those of the market, of every market and of the groups that include it.
func (r *IndicatorDefinitionRepository) ListForProviderAndMarket(ctx context.Context, providerID uint, marketID uint64) ([]models.IndicatorDefinition, error) {
	out := make([]models.IndicatorDefinition, 0)
	err := r.db.WithContext(ctx).
		Where("provider_id = ? AND (market_id = ? OR (market_id = 0 AND (NOT "+indicatorGroupSQL+" OR market_ids @> ?::jsonb)))",
			providerID, marketID, fmt.Sprintf("[%d]", marketID)).
		Order("id ASC").
		Find(&out).Error
	return out, err
}

func (r *IndicatorDefinitionRepository) Return(ctx context.Context, id uint) (*models.IndicatorDefinition, error) {
	ind := new(models.IndicatorDefinition)
	return ind, r.db.WithContext(ctx).Where("id = ?", id).First(ind).Error
}

// indicatorGroupSQL is true for the definitions of a group of markets.
const indicatorGroupSQL = "COALESCE(jsonb_array_length(CASE WHEN jsonb_typeof(market_ids) = 'array' THEN market_ids END), 0) > 0"

// Scopes of indicator definitions.
const (
	IndicatorScopeAll    = "ALL"
	IndicatorScopeGroup  = "GROUP"
	IndicatorScopeMarket = "MARKET"
)

// IndicatorDefinitionFilter selects the definitions of a provider in the
// panel; empty fields match everything.
type IndicatorDefinitionFilter struct {
	ProviderID  uint
	Types       []string
	TimeframeID uint
	// MarketID selects the definitions of a market and of the groups that
	// include it.
	MarketID uint
	// Scope is ALL, GROUP or MARKET.
	Scope   string
	Enabled *bool
}

func (f IndicatorDefinitionFilter) apply(tx *gorm.DB) *gorm.DB {
	tx = tx.Where("provider_id = ?", f.ProviderID)
	if f.Types != nil {
		tx = tx.Where("type IN ?", f.Types)
	}
	if f.TimeframeID != 0 {
		tx = tx.Where("timeframe_id = ?", f.TimeframeID)
	}
	if f.MarketID != 0 {
		tx = tx.Where("market_id = ? OR market_ids @> ?::jsonb", f.MarketID, fmt.Sprintf("[%d]", f.MarketID))
	}
	switch f.Scope {
	case IndicatorScopeAll:
		tx = tx.Where("market_id = 0 AND NOT " + indicatorGroupSQL)
	case IndicatorScopeGroup:
		tx = tx.Where(indicatorGroupSQL)
	case IndicatorScopeMarket:
		tx = tx.Where("market_id <> 0")
	}
	if f.Enabled != nil {
		tx = tx.Where("enabled = ?", *f.Enabled)
	}
	return tx
}

// ListFiltered returns a page of the definitions, newest first.
func (r *IndicatorDefinitionRepository) ListFiltered(ctx context.Context, filter IndicatorDefinitionFilter, pg paginator.Params) ([]models.IndicatorDefinition, int64, error) {
	out := make([]models.IndicatorDefinition, 0)
	if filter.Types != nil && len(filter.Types) == 0 {
		return out, 0, nil
	}
	var total int64
	tx := filter.apply(r.db.WithContext(ctx).Model(&models.IndicatorDefinition{}))
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := pg.Apply(filter.apply(r.db.WithContext(ctx))).Order("id DESC").Find(&out).Error
	return out, total, err
}

// ExistingFingerprints returns which of the fingerprints are stored, except
// the definition exceptID.
func (r *IndicatorDefinitionRepository) ExistingFingerprints(ctx context.Context, fingerprints []string, exceptID uint) ([]string, error) {
	out := make([]string, 0)
	err := r.db.WithContext(ctx).Model(&models.IndicatorDefinition{}).
		Where("fingerprint IN ? AND id <> ?", fingerprints, exceptID).
		Pluck("fingerprint", &out).Error
	return out, err
}

// CreateMany stores the definitions in one transaction. Soft-deleted rows
// with the same fingerprints are removed first, since the fingerprint is
// unique among deleted rows too.
func (r *IndicatorDefinitionRepository) CreateMany(ctx context.Context, definitions []*models.IndicatorDefinition) error {
	fingerprints := make([]string, len(definitions))
	for i, definition := range definitions {
		fingerprints[i] = definition.Fingerprint
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("fingerprint IN ? AND deleted_at IS NOT NULL", fingerprints).
			Delete(&models.IndicatorDefinition{}).Error; err != nil {
			return err
		}
		return tx.Create(definitions).Error
	})
}

// UpdateDefinition replaces the params, markets and fingerprint of a definition.
func (r *IndicatorDefinitionRepository) UpdateDefinition(ctx context.Context, in *models.IndicatorDefinition) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("fingerprint = ? AND deleted_at IS NOT NULL", in.Fingerprint).
			Delete(&models.IndicatorDefinition{}).Error; err != nil {
			return err
		}
		// A struct update applies the JSON serializer of MarketIDs; Select
		// writes the zero values too.
		return tx.Model(&models.IndicatorDefinition{ID: in.ID}).
			Select("params", "fingerprint", "market_id", "market_ids", "updated_at").Updates(in).Error
	})
}

func (r *IndicatorDefinitionRepository) SetEnabled(ctx context.Context, id uint, enabled bool) error {
	return r.db.WithContext(ctx).Model(&models.IndicatorDefinition{}).Where("id = ?", id).
		Update("enabled", enabled).Error
}

// Delete removes a definition for good, so the same definition can be added again.
func (r *IndicatorDefinitionRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&models.IndicatorDefinition{}).Error
}
