package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/pkg/messaging"
	"github.com/sirupsen/logrus"
)

// Service provides the business logic layer for bidirectional messaging.
// It integrates MessageManager, AccessControl, and repositories.
type Service struct {
	manager        *messaging.MessageManagerImpl
	accessControl  *messaging.AccessControlImpl
	messageRepo    *repositories.MessageRepository
	botUserRepo    *repositories.BotUserRepository
	accessRuleRepo *repositories.AccessRuleRepository
}

// NewService creates a new messaging service.
func NewService(
	messageRepo *repositories.MessageRepository,
	botUserRepo *repositories.BotUserRepository,
	accessRuleRepo *repositories.AccessRuleRepository,
) *Service {
	// Create access control with database persistence
	ac := messaging.NewAccessControl(accessRuleRepo)

	// Create message manager with access control
	manager := messaging.NewMessageManager(ac)

	return &Service{
		manager:        manager,
		accessControl:  ac,
		messageRepo:    messageRepo,
		botUserRepo:    botUserRepo,
		accessRuleRepo: accessRuleRepo,
	}
}

// LoadRules loads access control rules from the database at startup.
func (s *Service) LoadRules(ctx context.Context) error {
	return s.accessControl.LoadRules(ctx)
}

// --- Sender Registration ---

// RegisterSender registers a new message sender with the manager.
func (s *Service) RegisterSender(senderID string, sender messaging.MessageSender) error {
	if !s.manager.RegisterSender(senderID, sender) {
		return fmt.Errorf("failed to register sender: %s (already exists or invalid)", senderID)
	}
	return nil
}

// UnregisterSender removes a sender from the manager.
func (s *Service) UnregisterSender(senderID string) error {
	if !s.manager.UnregisterSender(senderID) {
		return fmt.Errorf("sender not found: %s", senderID)
	}
	return nil
}

// ListSenders returns all registered sender IDs.
func (s *Service) ListSenders() []string {
	return s.manager.ListSenders()
}

// --- Message Sending ---

// SendMessage sends a message immediately.
func (s *Service) SendMessage(ctx context.Context, senderID string, message *models.Message) (*models.SendResult, error) {
	if message.ID == uuid.Nil {
		message.ID = uuid.New()
	}
	message.Status = models.MessageStatusPending

	// Persist the message
	if err := s.messageRepo.Create(ctx, message); err != nil {
		logrus.WithError(err).Error("failed to persist message")
		// Continue with sending even if persistence fails
	}

	// Send through the manager
	result, err := s.manager.SendMessage(senderID, message)

	// Update status based on result
	if result != nil && result.Success {
		_ = s.messageRepo.UpdateStatus(ctx, message.ID, models.MessageStatusSent)
	} else {
		_ = s.messageRepo.UpdateStatus(ctx, message.ID, models.MessageStatusFailed)
	}

	return result, err
}

// ScheduleMessage schedules a message for future delivery.
func (s *Service) ScheduleMessage(ctx context.Context, senderID string, message *models.Message) error {
	if message.ScheduledAt == nil {
		return fmt.Errorf("scheduledAt is required for scheduled messages")
	}
	if message.ScheduledAt.Before(time.Now()) {
		return fmt.Errorf("scheduledAt must be in the future")
	}

	// Check access control before scheduling
	if !s.accessControl.CheckAccess(senderID, message.ClientID, models.PermissionSchedule) {
		return fmt.Errorf("access denied: sender %s cannot schedule messages via client %s", senderID, message.ClientID)
	}

	if message.ID == uuid.Nil {
		message.ID = uuid.New()
	}
	message.SenderID = senderID
	message.Status = models.MessageStatusScheduled

	return s.messageRepo.Create(ctx, message)
}

// CancelScheduledMessage cancels a scheduled message.
func (s *Service) CancelScheduledMessage(ctx context.Context, messageID uuid.UUID) error {
	return s.messageRepo.CancelScheduled(ctx, messageID)
}

// ProcessDueScheduledMessages processes messages that are due for delivery.
// This is called by the scheduler worker.
func (s *Service) ProcessDueScheduledMessages(ctx context.Context) (int, error) {
	messages, err := s.messageRepo.GetDueScheduled(ctx, time.Now())
	if err != nil {
		return 0, fmt.Errorf("get due scheduled messages: %w", err)
	}

	processed := 0
	for i := range messages {
		msg := &messages[i]

		// Update status to sending
		_ = s.messageRepo.UpdateStatus(ctx, msg.ID, models.MessageStatusSending)

		// Send through the manager
		result, sendErr := s.manager.SendMessage(msg.SenderID, msg)

		if sendErr != nil || (result != nil && !result.Success) {
			_ = s.messageRepo.UpdateStatus(ctx, msg.ID, models.MessageStatusFailed)
			logrus.WithError(sendErr).WithField("message_id", msg.ID).
				Warn("scheduled message send failed")
		} else {
			_ = s.messageRepo.UpdateStatus(ctx, msg.ID, models.MessageStatusSent)
			processed++
		}
	}

	return processed, nil
}

// --- Access Control ---

// GrantAccess grants a sender access to a specific client.
func (s *Service) GrantAccess(ctx context.Context, senderID, clientID, grantedBy, reason string, permission models.Permission, expiresAt *time.Time) (*models.AccessRule, error) {
	return s.accessControl.GrantAccessWithDetails(senderID, clientID, permission, grantedBy, reason, expiresAt)
}

// RevokeAccess revokes a sender's access to a client.
func (s *Service) RevokeAccess(ctx context.Context, ruleID uuid.UUID) (bool, error) {
	return s.accessControl.RevokeAccessByID(ruleID)
}

// ListAccessRules returns all active access rules.
func (s *Service) ListAccessRules() []*models.AccessRule {
	return s.accessControl.ListRules()
}

// CheckAccess checks if a sender has a specific permission on a client.
func (s *Service) CheckAccess(senderID, clientID string, permission models.Permission) bool {
	return s.accessControl.CheckAccess(senderID, clientID, permission)
}

// --- Bot User Management ---

// RegisterBotUser registers a user who started a bot.
func (s *Service) RegisterBotUser(ctx context.Context, botID string, chatID int64) error {
	botUser := &models.BotUser{
		BotID:             botID,
		ChatID:            chatID,
		StartedAt:         time.Now(),
		LastInteractionAt: time.Now(),
	}
	return s.botUserRepo.Upsert(ctx, botUser)
}

// RemoveBotUser removes a user from a bot (user blocked the bot).
func (s *Service) RemoveBotUser(ctx context.Context, botID string, chatID int64) error {
	return s.botUserRepo.Delete(ctx, botID, chatID)
}

// IsBotUserRegistered checks if a user has started a bot.
func (s *Service) IsBotUserRegistered(ctx context.Context, botID string, chatID int64) (bool, error) {
	return s.botUserRepo.Exists(ctx, botID, chatID)
}

// --- Event Subscription ---

// SubscribeToSendEvents subscribes to send events for a sender.
func (s *Service) SubscribeToSendEvents(senderID string, events chan<- *messaging.SendEvent) bool {
	return s.manager.SubscribeToSendEvents(senderID, events)
}

// UnsubscribeFromSendEvents unsubscribes from send events.
func (s *Service) UnsubscribeFromSendEvents(senderID string, events chan<- *messaging.SendEvent) bool {
	return s.manager.UnsubscribeFromSendEvents(senderID, events)
}

// --- Lifecycle ---

// GetManager returns the underlying message manager for direct access if needed.
func (s *Service) GetManager() *messaging.MessageManagerImpl {
	return s.manager
}

// Close cleans up resources.
func (s *Service) Close() error {
	_ = s.manager.Close()
	_ = s.accessControl.Close()
	return nil
}
