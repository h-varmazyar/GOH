package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MessageType represents the type of message content
type MessageType string

const (
	// MessageTypeText represents a text message
	MessageTypeText MessageType = "text"
	// MessageTypePhoto represents a photo message
	MessageTypePhoto MessageType = "photo"
	// MessageTypeDocument represents a document message
	MessageTypeDocument MessageType = "document"
	// MessageTypeSticker represents a sticker message
	MessageTypeSticker MessageType = "sticker"
)

// MessageStatus represents the status of a message in the sending lifecycle
type MessageStatus string

const (
	// MessageStatusPending represents a message that is queued for sending
	MessageStatusPending MessageStatus = "pending"
	// MessageStatusScheduled represents a message scheduled for future delivery
	MessageStatusScheduled MessageStatus = "scheduled"
	// MessageStatusSending represents a message currently being sent
	MessageStatusSending MessageStatus = "sending"
	// MessageStatusSent represents a successfully sent message
	MessageStatusSent MessageStatus = "sent"
	// MessageStatusFailed represents a failed message send attempt
	MessageStatusFailed MessageStatus = "failed"
)

// SenderType represents the type of message sender
type SenderType string

const (
	// SenderTypeBot represents a Telegram bot sender
	SenderTypeBot SenderType = "bot"
	// SenderTypePersonalAccount represents a personal Telegram account sender
	SenderTypePersonalAccount SenderType = "personal_account"
	// SenderTypeService represents a service/worker sender
	SenderTypeService SenderType = "service"
	// SenderTypeWorker represents a worker process sender
	SenderTypeWorker SenderType = "worker"
)

// Permission represents individual access control permissions
type Permission string

const (
	// PermissionSend represents permission to send messages
	PermissionSend Permission = "send"
	// PermissionSchedule represents permission to schedule messages
	PermissionSchedule Permission = "schedule"
	// PermissionSendToAllChats represents permission to send to any chat
	PermissionSendToAllChats Permission = "send_to_all_chats"
)

// PermissionSet contains access control permissions for a sender
type PermissionSet struct {
	// CanSendMessages indicates if sender can send messages
	CanSendMessages bool `gorm:"not null;default:false" json:"can_send_messages"`
	// CanSendToAllChats indicates if sender can send to any chat without restriction
	CanSendToAllChats bool `gorm:"not null;default:false" json:"can_send_to_all_chats"`
	// AllowedChatIDs contains specific chat IDs the sender can message
	AllowedChatIDs []int64 `gorm:"type:bigint[]" json:"allowed_chat_ids"`
	// MaxMessagesPerMinute defines rate limit for the sender
	MaxMessagesPerMinute int `gorm:"not null;default:30" json:"max_messages_per_minute"`
	// CanScheduleMessages indicates if sender can schedule future messages
	CanScheduleMessages bool `gorm:"not null;default:false" json:"can_schedule_messages"`
}

// Message represents a Telegram message to be sent
type Message struct {
	gorm.Model

	// ID is the unique identifier for the message (UUID)
	ID uuid.UUID `gorm:"primarykey" json:"id"`

	// SenderID identifies the service/worker sending the message
	SenderID string `gorm:"not null;index" json:"sender_id"`

	// ClientID identifies the Telegram client/account to use for sending
	ClientID string `gorm:"not null;index" json:"client_id"`

	// ChatID is the Telegram chat identifier to send the message to
	ChatID int64 `gorm:"not null;index" json:"chat_id"`

	// Content contains the message text or content
	Content string `gorm:"type:text;not null" json:"content"`

	// ContentType specifies the type of message content
	ContentType MessageType `gorm:"not null;default:'text'" json:"content_type"`

	// ScheduledAt contains the timestamp when the message should be sent (null for immediate)
	ScheduledAt *time.Time `gorm:"index" json:"scheduled_at,omitempty"`

	// Status indicates the current state of the message in the sending lifecycle
	Status MessageStatus `gorm:"not null;default:'pending'" json:"status"`

	// Metadata contains additional key-value pairs for the message
	Metadata map[string]string `gorm:"type:jsonb" json:"metadata,omitempty"`
}

// Validate validates the Message struct according to the design specification
func (m *Message) Validate() error {
	// Check that ID is set (non-zero UUID)
	if m.ID == uuid.Nil {
		return errors.New("message ID must be set")
	}

	// Check senderID is not empty
	if m.SenderID == "" {
		return errors.New("senderID must be non-empty")
	}

	// Check clientID is not empty
	if m.ClientID == "" {
		return errors.New("clientID must be non-empty")
	}

	// Check chatID is valid (> 0)
	if m.ChatID <= 0 {
		return errors.New("chatID must be a valid Telegram chat identifier (> 0)")
	}

	// Check content is not empty
	if m.Content == "" {
		return errors.New("content must be non-empty string")
	}

	// Check content length doesn't exceed Telegram limits (4096 chars for text)
	if m.ContentType == MessageTypeText && len(m.Content) > 4096 {
		return errors.New("text content length must not exceed Telegram limit of 4096 characters")
	}

	// Check scheduledAt is null or future timestamp
	if m.ScheduledAt != nil {
		if m.ScheduledAt.Before(time.Now()) {
			return errors.New("scheduledAt must be null or future timestamp")
		}
		// Ensure status is Scheduled if scheduledAt is set
		if m.Status != MessageStatusScheduled {
			return errors.New("message with scheduledAt must have status 'scheduled'")
		}
	}

	return nil
}

// IsImmediate returns true if the message should be sent immediately (not scheduled)
func (m *Message) IsImmediate() bool {
	return m.ScheduledAt == nil
}

// SendResult contains the result of a message send operation
type SendResult struct {
	// Success indicates whether the message was sent successfully
	Success bool `json:"success"`

	// MessageID is the Telegram message ID if successfully sent (0 if failed)
	MessageID int64 `json:"message_id"`

	// Error contains error message if the send operation failed
	Error string `json:"error,omitempty"`

	// SentAt is the timestamp when the message was sent (or attempted)
	SentAt time.Time `json:"sent_at"`
}

// SenderInfo contains information about a message sender
type SenderInfo struct {
	// ID uniquely identifies the sender
	ID string `json:"id"`

	// Type indicates the type of sender (bot, personal account, etc.)
	Type SenderType `json:"type"`

	// Permissions contains the access control permissions for this sender
	Permissions PermissionSet `json:"permissions"`

	// ClientIDs contains the Telegram client IDs this sender can use
	ClientIDs []string `json:"client_ids"`
}

// AccessRule defines an access control rule for message sending
type AccessRule struct {
	gorm.Model

	// ID is the unique identifier for the access rule (UUID)
	ID uuid.UUID `gorm:"primarykey" json:"id"`

	// SenderID identifies the service/worker granted access
	SenderID string `gorm:"not null;index" json:"sender_id"`

	// ClientID identifies the Telegram client/account that can be used
	ClientID string `gorm:"not null;index" json:"client_id"`

	// Permissions contains the granted permissions for this rule
	Permissions PermissionSet `gorm:"type:jsonb;not null" json:"permissions"`

	// GrantedAt is when the access was granted
	GrantedAt time.Time `gorm:"not null" json:"granted_at"`

	// GrantedBy identifies who granted the access
	GrantedBy string `gorm:"not null" json:"granted_by"`

	// ExpiresAt is when the access rule expires (null for no expiration)
	ExpiresAt *time.Time `gorm:"index" json:"expires_at,omitempty"`

	// Reason explains why access was granted
	Reason string `gorm:"type:text" json:"reason,omitempty"`
}

// Validate validates the AccessRule struct according to the design specification
func (r *AccessRule) Validate() error {
	// Check that ID is set (non-zero UUID)
	if r.ID == uuid.Nil {
		return errors.New("access rule ID must be set")
	}

	// Check senderID is not empty
	if r.SenderID == "" {
		return errors.New("senderID must be non-empty")
	}

	// Check clientID is not empty
	if r.ClientID == "" {
		return errors.New("clientID must be non-empty")
	}

	// Check grantedAt is not zero
	if r.GrantedAt.IsZero() {
		return errors.New("grantedAt must be set")
	}

	// Check grantedBy is not empty
	if r.GrantedBy == "" {
		return errors.New("grantedBy must be non-empty")
	}

	// Check expiresAt is null or after grantedAt if set
	if r.ExpiresAt != nil {
		if r.ExpiresAt.Before(r.GrantedAt) {
			return errors.New("expiresAt must be null or after grantedAt")
		}
	}

	return nil
}

// IsExpired returns true if the access rule has expired
func (r *AccessRule) IsExpired() bool {
	if r.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*r.ExpiresAt)
}

// Note: BotUser struct has been moved to bot_users.go with proper composite primary key implementation
// and additional helper methods. Please use models.BotUser from the bot_users.go file.

// SendRecord tracks the result of a message send operation
type SendRecord struct {
	gorm.Model

	// MessageID references the original message
	MessageID uuid.UUID `gorm:"type:uuid;not null;index" json:"message_id"`

	// SenderID identifies who sent the message
	SenderID string `gorm:"not null;index" json:"sender_id"`

	// ClientID identifies which Telegram client was used
	ClientID string `gorm:"not null;index" json:"client_id"`

	// ChatID is the Telegram chat the message was sent to
	ChatID int64 `gorm:"not null;index" json:"chat_id"`

	// SentAt is when the message was actually sent
	SentAt time.Time `gorm:"not null" json:"sent_at"`

	// MessageIDInTelegram is the Telegram message ID if successful
	MessageIDInTelegram int64 `json:"message_id_in_telegram"`

	// Success indicates whether the send was successful
	Success bool `gorm:"not null" json:"success"`

	// ErrorMessage contains the error if the send failed
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"`

	// RetryCount indicates how many times the message was retried
	RetryCount int `gorm:"not null;default:0" json:"retry_count"`
}
