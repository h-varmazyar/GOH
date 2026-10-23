package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// BotUser tracks users who have started a bot (required for bot messaging)
// Composite primary key: (BotID, ChatID)
// Validates: Requirements 4.1, 4.2, 4.3, 4.4, 4.5
type BotUser struct {
	// BotID identifies the bot (matches sender ID)
	BotID string `gorm:"primaryKey;not null;index:idx_bot_id" json:"bot_id"`

	// ChatID is the Telegram user/chat identifier
	ChatID int64 `gorm:"primaryKey;not null;index:idx_chat_id" json:"chat_id"`

	// StartedAt is when the user started the bot
	StartedAt time.Time `gorm:"not null;index:idx_started_at" json:"started_at"`

	// LastInteractionAt is the last time the bot interacted with this user
	LastInteractionAt time.Time `gorm:"not null;index:idx_last_interaction" json:"last_interaction_at"`

	// Metadata contains additional information about the bot-user relationship
	Metadata map[string]string `gorm:"type:jsonb" json:"metadata,omitempty"`

	// CreatedAt is when the record was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the record was last updated
	UpdatedAt time.Time `json:"updated_at"`

	// DeletedAt is for soft deletion
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName specifies the custom table name for BotUser
func (BotUser) TableName() string {
	return "bot_users"
}

// Validate validates the BotUser struct according to the design specification
func (b *BotUser) Validate() error {
	// Check BotID is not empty
	if b.BotID == "" {
		return errors.New("botID must be non-empty")
	}

	// Check ChatID is valid (> 0)
	if b.ChatID <= 0 {
		return errors.New("chatID must be a valid Telegram chat identifier (> 0)")
	}

	// Check StartedAt is not zero
	if b.StartedAt.IsZero() {
		return errors.New("startedAt must be set")
	}

	// Check LastInteractionAt is not zero
	if b.LastInteractionAt.IsZero() {
		return errors.New("lastInteractionAt must be set")
	}

	// Ensure LastInteractionAt is not before StartedAt
	if b.LastInteractionAt.Before(b.StartedAt) {
		return errors.New("lastInteractionAt must not be before startedAt")
	}

	return nil
}

// BeforeCreate is a GORM hook that runs before creating a new BotUser record
func (b *BotUser) BeforeCreate(tx *gorm.DB) error {
	// Set timestamps if not set
	if b.StartedAt.IsZero() {
		b.StartedAt = time.Now()
	}
	if b.LastInteractionAt.IsZero() {
		b.LastInteractionAt = time.Now()
	}
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now()
	}
	if b.UpdatedAt.IsZero() {
		b.UpdatedAt = time.Now()
	}

	// Validate the struct
	return b.Validate()
}

// BeforeUpdate is a GORM hook that runs before updating a BotUser record
func (b *BotUser) BeforeUpdate(tx *gorm.DB) error {
	// Update the UpdatedAt timestamp
	b.UpdatedAt = time.Now()

	// Validate the struct
	return b.Validate()
}

// IsActive returns true if the user has interacted with the bot recently
// A user is considered active if they've interacted within the last 30 days
func (b *BotUser) IsActive() bool {
	return time.Since(b.LastInteractionAt) <= 30*24*time.Hour
}

// UpdateInteraction updates the LastInteractionAt timestamp to now
func (b *BotUser) UpdateInteraction() {
	b.LastInteractionAt = time.Now()
}

// HasStartedWithin returns true if the user started the bot within the specified duration
func (b *BotUser) HasStartedWithin(duration time.Duration) bool {
	return time.Since(b.StartedAt) <= duration
}

// GetMetadataValue returns a value from metadata map or empty string if not found
func (b *BotUser) GetMetadataValue(key string) string {
	if b.Metadata == nil {
		return ""
	}
	return b.Metadata[key]
}

// SetMetadataValue sets a value in the metadata map
func (b *BotUser) SetMetadataValue(key, value string) {
	if b.Metadata == nil {
		b.Metadata = make(map[string]string)
	}
	b.Metadata[key] = value
}

// BotUserStats represents statistics for a bot user
type BotUserStats struct {
	BotID               string    `json:"bot_id"`
	TotalUsers          int64     `json:"total_users"`
	ActiveUsers         int64     `json:"active_users"`
	NewUsersLast24Hours int64     `json:"new_users_last_24_hours"`
	AvgInteractionAge   float64   `json:"avg_interaction_age_hours"`
	LastUpdated         time.Time `json:"last_updated"`
}

// BotUserQuery represents query parameters for searching BotUsers
type BotUserQuery struct {
	BotID         string     `form:"bot_id"`
	ChatID        int64      `form:"chat_id"`
	ActiveOnly    bool       `form:"active_only"`
	StartedAfter  *time.Time `form:"started_after"`
	StartedBefore *time.Time `form:"started_before"`
	Limit         int        `form:"limit" binding:"min=1,max=100"`
	Offset        int        `form:"offset" binding:"min=0"`
}
