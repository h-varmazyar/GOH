package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/pkg/messaging"
	"gorm.io/gorm"
)

// AccessRuleRepository handles AccessRule database operations.
// It also implements the messaging.AccessPersistence interface.
type AccessRuleRepository struct {
	db *gorm.DB
}

// NewAccessRuleRepository creates a new AccessRuleRepository.
func NewAccessRuleRepository(db *gorm.DB) *AccessRuleRepository {
	return &AccessRuleRepository{db: db}
}

// SaveRule implements messaging.AccessPersistence.
func (r *AccessRuleRepository) SaveRule(ctx context.Context, rule *models.AccessRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

// DeleteRule implements messaging.AccessPersistence.
func (r *AccessRuleRepository) DeleteRule(ctx context.Context, ruleID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", ruleID).Delete(&models.AccessRule{}).Error
}

// LoadRules implements messaging.AccessPersistence.
func (r *AccessRuleRepository) LoadRules(ctx context.Context) ([]*models.AccessRule, error) {
	var rules []*models.AccessRule
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Find(&rules).Error
	return rules, err
}

// ListAll returns all access rules (including expired, for admin view).
func (r *AccessRuleRepository) ListAll(ctx context.Context) ([]models.AccessRule, error) {
	var rules []models.AccessRule
	err := r.db.WithContext(ctx).Find(&rules).Error
	return rules, err
}

// GetByID retrieves an access rule by its ID.
func (r *AccessRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.AccessRule, error) {
	var rule models.AccessRule
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// Ensure AccessRuleRepository satisfies the messaging.AccessPersistence interface.
var _ messaging.AccessPersistence = (*AccessRuleRepository)(nil)
