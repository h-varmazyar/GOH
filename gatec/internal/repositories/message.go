package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/h-varmazyar/gatec/internal/models"
	"gorm.io/gorm"
)

// MessageRepository handles Message database operations.
type MessageRepository struct {
	db *gorm.DB
}

// NewMessageRepository creates a new MessageRepository.
func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// Create stores a new message record.
func (r *MessageRepository) Create(ctx context.Context, message *models.Message) error {
	return r.db.WithContext(ctx).Create(message).Error
}

// Get retrieves a message by ID.
func (r *MessageRepository) Get(ctx context.Context, id uuid.UUID) (*models.Message, error) {
	var msg models.Message
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// UpdateStatus updates the status of a message.
func (r *MessageRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.MessageStatus) error {
	return r.db.WithContext(ctx).
		Model(&models.Message{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// GetDueScheduled returns messages that are scheduled and their time has come.
func (r *MessageRepository) GetDueScheduled(ctx context.Context, now time.Time) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.WithContext(ctx).
		Where("status = ? AND scheduled_at IS NOT NULL AND scheduled_at <= ?", models.MessageStatusScheduled, now).
		Order("scheduled_at ASC").
		Find(&messages).Error
	return messages, err
}

// ListBySender returns messages sent by a specific sender.
func (r *MessageRepository) ListBySender(ctx context.Context, senderID string, limit int) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.WithContext(ctx).
		Where("sender_id = ?", senderID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error
	return messages, err
}

// CancelScheduled cancels a scheduled message (sets status to failed).
func (r *MessageRepository) CancelScheduled(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&models.Message{}).
		Where("id = ? AND status = ?", id, models.MessageStatusScheduled).
		Update("status", models.MessageStatusFailed).Error
}
