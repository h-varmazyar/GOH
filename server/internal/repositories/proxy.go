package repositories

import (
	"context"
	"time"

	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
)

type ProxyRepository struct {
	db *gorm.DB
}

func NewProxyRepository(db *gorm.DB) *ProxyRepository {
	return &ProxyRepository{db: db}
}

// Create creates a new proxy record
func (r *ProxyRepository) Create(ctx context.Context, proxy *models.Proxy) error {
	return r.db.WithContext(ctx).Create(proxy).Error
}

// GetByID returns a proxy by ID
func (r *ProxyRepository) GetByID(ctx context.Context, id uint) (*models.Proxy, error) {
	proxy := &models.Proxy{}
	return proxy, r.db.WithContext(ctx).First(proxy, id).Error
}

// ListActive returns all enabled proxies with active status
func (r *ProxyRepository) ListActive(ctx context.Context) ([]models.Proxy, error) {
	proxies := make([]models.Proxy, 0)
	return proxies, r.db.WithContext(ctx).
		Where("is_enable = ? AND status = ?", true, models.ProxyStatusActive).
		Order("assigned_accounts ASC").
		Find(&proxies).Error
}

// ListEnabled returns all enabled proxies regardless of status
func (r *ProxyRepository) ListEnabled(ctx context.Context) ([]models.Proxy, error) {
	proxies := make([]models.Proxy, 0)
	return proxies, r.db.WithContext(ctx).
		Where("is_enable = ?", true).
		Find(&proxies).Error
}

// ListAll returns all proxies
func (r *ProxyRepository) ListAll(ctx context.Context) ([]models.Proxy, error) {
	proxies := make([]models.Proxy, 0)
	return proxies, r.db.WithContext(ctx).Find(&proxies).Error
}

// UpdateStatus updates the status and health tracking fields of a proxy
func (r *ProxyRepository) UpdateStatus(ctx context.Context, id uint, status models.ProxyStatus, success bool) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":          status,
		"last_checked_at": now,
	}

	if success {
		updates["last_success_at"] = now
		updates["consecutive_fails"] = 0
	} else {
		updates["consecutive_fails"] = gorm.Expr("consecutive_fails + 1")
		updates["fail_count"] = gorm.Expr("fail_count + 1")
	}

	return r.db.WithContext(ctx).
		Model(&models.Proxy{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// DeleteStaleProxies deletes proxies that haven't had a successful connection
// for longer than the specified duration
func (r *ProxyRepository) DeleteStaleProxies(ctx context.Context, staleDuration time.Duration) (int64, error) {
	threshold := time.Now().Add(-staleDuration)

	result := r.db.WithContext(ctx).
		Where("(last_success_at IS NULL AND created_at < ?) OR (last_success_at IS NOT NULL AND last_success_at < ?)", threshold, threshold).
		Where("is_enable = ?", true).
		Delete(&models.Proxy{})

	return result.RowsAffected, result.Error
}

// GetLeastUsedActiveProxy returns the active proxy with the fewest assigned accounts
func (r *ProxyRepository) GetLeastUsedActiveProxy(ctx context.Context, excludeIDs []uint) (*models.Proxy, error) {
	proxy := &models.Proxy{}
	tx := r.db.WithContext(ctx).
		Where("is_enable = ? AND status = ?", true, models.ProxyStatusActive).
		Order("assigned_accounts ASC")

	if len(excludeIDs) > 0 {
		tx = tx.Where("id NOT IN ?", excludeIDs)
	}

	return proxy, tx.First(proxy).Error
}

// IncrementAssignedAccounts increases the assigned_accounts counter
func (r *ProxyRepository) IncrementAssignedAccounts(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&models.Proxy{}).
		Where("id = ?", id).
		Update("assigned_accounts", gorm.Expr("assigned_accounts + 1")).Error
}

// DecrementAssignedAccounts decreases the assigned_accounts counter
func (r *ProxyRepository) DecrementAssignedAccounts(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&models.Proxy{}).
		Where("id = ?", id).
		Update("assigned_accounts", gorm.Expr("GREATEST(assigned_accounts - 1, 0)")).Error
}

// --- ProxyAssignment repository ---

type ProxyAssignmentRepository struct {
	db *gorm.DB
}

func NewProxyAssignmentRepository(db *gorm.DB) *ProxyAssignmentRepository {
	return &ProxyAssignmentRepository{db: db}
}

// Assign creates a new proxy assignment for an account
func (r *ProxyAssignmentRepository) Assign(ctx context.Context, accountID string, proxyID uint) error {
	// Deactivate any existing assignment for this account
	r.db.WithContext(ctx).
		Model(&models.ProxyAssignment{}).
		Where("account_id = ? AND is_active = ?", accountID, true).
		Update("is_active", false)

	assignment := &models.ProxyAssignment{
		AccountID: accountID,
		ProxyID:   proxyID,
		IsActive:  true,
	}
	return r.db.WithContext(ctx).Create(assignment).Error
}

// GetActiveAssignment returns the current active proxy assignment for an account
func (r *ProxyAssignmentRepository) GetActiveAssignment(ctx context.Context, accountID string) (*models.ProxyAssignment, error) {
	assignment := &models.ProxyAssignment{}
	return assignment, r.db.WithContext(ctx).
		Preload("Proxy").
		Where("account_id = ? AND is_active = ?", accountID, true).
		First(assignment).Error
}

// DeactivateForAccount deactivates all proxy assignments for an account
func (r *ProxyAssignmentRepository) DeactivateForAccount(ctx context.Context, accountID string) error {
	return r.db.WithContext(ctx).
		Model(&models.ProxyAssignment{}).
		Where("account_id = ? AND is_active = ?", accountID, true).
		Update("is_active", false).Error
}

// ListActiveAssignments returns all active proxy assignments
func (r *ProxyAssignmentRepository) ListActiveAssignments(ctx context.Context) ([]models.ProxyAssignment, error) {
	assignments := make([]models.ProxyAssignment, 0)
	return assignments, r.db.WithContext(ctx).
		Preload("Proxy").
		Where("is_active = ?", true).
		Find(&assignments).Error
}
