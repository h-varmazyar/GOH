package messaging

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/h-varmazyar/gatec/internal/models"
)

// MessageSender defines the interface for sending messages through Telegram.
// Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.5
type MessageSender interface {
	// SendMessage sends a message to the specified chat.
	// Validates: Requirements 1.1
	SendMessage(message *models.Message) (*models.SendResult, error)

	// CanSendTo checks if the sender can send to the specified chat ID.
	// Validates: Requirements 1.2
	CanSendTo(chatID int64) bool

	// GetSenderInfo returns information about the sender.
	// Validates: Requirements 1.3
	GetSenderInfo() *models.SenderInfo
}

// MessageManager defines the interface for managing message senders and sending operations.
// Validates: Requirements 2.1, 2.2, 2.3, 2.4, 2.5
type MessageManager interface {
	// RegisterSender registers a new message sender.
	// Validates: Requirements 2.1, 2.5
	RegisterSender(senderID string, sender MessageSender) bool

	// UnregisterSender removes a registered message sender.
	// Validates: Requirements 2.2, 2.5
	UnregisterSender(senderID string) bool

	// SendMessage sends a message using the specified sender.
	// Validates: Requirements 2.3, 2.5, 10.1, 10.2, 10.3
	SendMessage(senderID string, message *models.Message) (*models.SendResult, error)

	// SubscribeToSendEvents subscribes to send events for a sender.
	// Validates: Requirements 2.4, 2.5, 6.1, 6.2, 6.3, 6.4, 6.5
	SubscribeToSendEvents(senderID string, events chan<- *SendEvent) bool

	// UnsubscribeFromSendEvents unsubscribes from send events for a sender.
	// Validates: Requirements 2.4, 2.5, 6.4
	UnsubscribeFromSendEvents(senderID string, events chan<- *SendEvent) bool
}

// ExtendedMessageManager provides additional methods for message management.
type ExtendedMessageManager interface {
	MessageManager

	// GetSender retrieves a registered message sender.
	GetSender(senderID string) (MessageSender, bool)

	// ListSenders lists all registered message senders.
	ListSenders() []string

	// Close stops background processes and cleans up resources.
	Close() error
}

// AccessControl defines the interface for access control management.
// Validates: Requirements 3.1, 3.2, 3.3, 3.6
type AccessControl interface {
	// CheckAccess checks if a sender has permission to use a specific client.
	// Validates: Requirements 3.1, 3.6
	CheckAccess(senderID, clientID string, permission models.Permission) bool

	// GrantAccess grants permission for a sender to use a specific client.
	// Validates: Requirements 3.2, 3.4, 3.5
	GrantAccess(senderID, clientID string, permission models.Permission) bool

	// RevokeAccess revokes permission for a sender to use a specific client.
	// Validates: Requirements 3.3, 3.4
	RevokeAccess(senderID, clientID string, permission models.Permission) bool

	// UpdateRule updates an existing access rule.
	// Validates: Requirements 3.4, 3.5
	UpdateRule(rule *models.AccessRule) bool

	// RemoveRule removes an access rule by senderID and clientID.
	// Validates: Requirements 3.3, 3.4
	RemoveRule(senderID, clientID string) bool
}

// ExtendedAccessControl provides additional methods for more detailed access control operations.
type ExtendedAccessControl interface {
	AccessControl

	// GrantAccessWithDetails grants access with additional metadata (grantedBy, reason, expiration).
	// Validates: Requirements 3.4, 3.5
	GrantAccessWithDetails(senderID, clientID string, permission models.Permission, grantedBy, reason string, expiresAt *time.Time) (*models.AccessRule, error)

	// RevokeAccessByID revokes access by rule ID.
	RevokeAccessByID(ruleID uuid.UUID) (bool, error)

	// GetRule retrieves an access rule by senderID and clientID.
	GetRule(senderID, clientID string) (*models.AccessRule, bool)

	// ListRules lists all active (non-expired) access rules.
	ListRules() []*models.AccessRule

	// LoadRules loads rules from persistence.
	LoadRules(ctx context.Context) error

	// Close stops background processes and cleans up resources.
	Close() error
}

// SendEvent represents an event generated during message sending.
// Validates: Requirements 6.1
type SendEvent struct {
	// MessageID is the unique identifier of the message
	MessageID uuid.UUID `json:"message_id"`

	// SenderID identifies who sent the message
	SenderID string `json:"sender_id"`

	// Result contains the result of the send operation
	Result *models.SendResult `json:"result"`

	// Timestamp is when the event occurred
	Timestamp time.Time `json:"timestamp"`
}
