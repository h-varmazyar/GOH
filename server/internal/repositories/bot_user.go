package repositories

import (
	"context"
	"time"

	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BotUserRepository handles BotUser database operations.
type BotUserRepository struct {
	db *gorm.DB
}

// NewBotUserRepository creates a new BotUserRepository.
func NewBotUserRepository(db *gorm.DB) *BotUserRepository {
	return &BotUserRepository{db: db}
}

// Upsert creates or updates a bot user record.
// If the user already exists, it updates the last interaction time.
func (r *BotUserRepository) Upsert(ctx context.Context, botUser *models.BotUser) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "bot_id"}, {Name: "chat_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"last_interaction_at", "updated_at", "metadata"}),
		}).
		Create(botUser).Error
}

// Exists checks if a bot user record exists (user has started the bot).
func (r *BotUserRepository) Exists(ctx context.Context, botID string, chatID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.BotUser{}).
		Where("bot_id = ? AND chat_id = ? AND deleted_at IS NULL", botID, chatID).
		Count(&count).Error
	return count > 0, err
}

// Get retrieves a bot user record.
func (r *BotUserRepository) Get(ctx context.Context, botID string, chatID int64) (*models.BotUser, error) {
	var user models.BotUser
	err := r.db.WithContext(ctx).
		Where("bot_id = ? AND chat_id = ?", botID, chatID).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ListByBot returns all users of a specific bot.
func (r *BotUserRepository) ListByBot(ctx context.Context, botID string) ([]models.BotUser, error) {
	var users []models.BotUser
	err := r.db.WithContext(ctx).
		Where("bot_id = ?", botID).
		Order("started_at DESC").
		Find(&users).Error
	return users, err
}

// Delete removes a bot user record (user blocked the bot).
func (r *BotUserRepository) Delete(ctx context.Context, botID string, chatID int64) error {
	return r.db.WithContext(ctx).
		Where("bot_id = ? AND chat_id = ?", botID, chatID).
		Delete(&models.BotUser{}).Error
}

// CountByBot returns the total number of users for a bot.
func (r *BotUserRepository) CountByBot(ctx context.Context, botID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.BotUser{}).
		Where("bot_id = ? AND deleted_at IS NULL", botID).
		Count(&count).Error
	return count, err
}

// CleanupStale removes bot user records that haven't interacted in the given duration.
func (r *BotUserRepository) CleanupStale(ctx context.Context, staleDuration time.Duration) (int64, error) {
	cutoff := time.Now().Add(-staleDuration)
	result := r.db.WithContext(ctx).
		Where("last_interaction_at < ?", cutoff).
		Delete(&models.BotUser{})
	return result.RowsAffected, result.Error
}
