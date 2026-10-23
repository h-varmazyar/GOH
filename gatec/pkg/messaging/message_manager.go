package messaging

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/h-varmazyar/gatec/internal/models"
)

// MessageManagerImpl implements the MessageManager and ExtendedMessageManager interfaces.
// It coordinates sender registration, message sending with access control,
// and send event subscriptions.
type MessageManagerImpl struct {
	mu sync.RWMutex

	// senders maps senderID -> MessageSender
	senders map[string]MessageSender

	// accessControl provides access permission checking
	accessControl ExtendedAccessControl

	// eventSubscribers maps senderID -> list of event channels
	eventMu          sync.RWMutex
	eventSubscribers map[string][]chan<- *SendEvent
}

// NewMessageManager creates a new MessageManagerImpl.
func NewMessageManager(accessControl ExtendedAccessControl) *MessageManagerImpl {
	return &MessageManagerImpl{
		senders:          make(map[string]MessageSender),
		accessControl:    accessControl,
		eventSubscribers: make(map[string][]chan<- *SendEvent),
	}
}

// RegisterSender registers a new message sender.
func (m *MessageManagerImpl) RegisterSender(senderID string, sender MessageSender) bool {
	if senderID == "" || sender == nil {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.senders[senderID]; exists {
		return false
	}

	m.senders[senderID] = sender
	log.Printf("[MessageManager] Sender registered: %s", senderID)
	return true
}

// UnregisterSender removes a registered message sender.
func (m *MessageManagerImpl) UnregisterSender(senderID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.senders[senderID]; !exists {
		return false
	}

	delete(m.senders, senderID)
	log.Printf("[MessageManager] Sender unregistered: %s", senderID)
	return true
}

// SendMessage sends a message using the specified sender.
// Flow: validate → check access → check canSendTo → send → notify subscribers.
func (m *MessageManagerImpl) SendMessage(senderID string, message *models.Message) (*models.SendResult, error) {
	if senderID == "" {
		return failResult("senderID is required"), fmt.Errorf("senderID is required")
	}
	if message == nil {
		return failResult("message is required"), fmt.Errorf("message is required")
	}

	// Assign ID if not set
	if message.ID == uuid.Nil {
		message.ID = uuid.New()
	}

	// Step 1: Get the sender
	m.mu.RLock()
	sender, exists := m.senders[senderID]
	m.mu.RUnlock()

	if !exists {
		return failResult("sender not registered"), fmt.Errorf("sender not registered: %s", senderID)
	}

	// Step 2: Check access control
	if m.accessControl != nil {
		hasAccess := m.accessControl.CheckAccess(senderID, message.ClientID, models.PermissionSend)
		if !hasAccess {
			return failResult("access denied"), fmt.Errorf("access denied: sender %s cannot use client %s", senderID, message.ClientID)
		}
	}

	// Step 3: Check if sender can send to this chat
	if !sender.CanSendTo(message.ChatID) {
		return failResult(fmt.Sprintf("cannot send to chat %d", message.ChatID)),
			fmt.Errorf("sender %s cannot send to chat %d", senderID, message.ChatID)
	}

	// Step 4: Send the message
	result, err := sender.SendMessage(message)
	if err != nil {
		log.Printf("[MessageManager] Send failed: sender=%s, chat=%d, err=%v", senderID, message.ChatID, err)
	}

	// Step 5: Notify subscribers
	event := &SendEvent{
		MessageID: message.ID,
		SenderID:  senderID,
		Result:    result,
		Timestamp: time.Now(),
	}
	m.notifySubscribers(senderID, event)

	return result, err
}

// SubscribeToSendEvents subscribes to send events for a sender.
func (m *MessageManagerImpl) SubscribeToSendEvents(senderID string, events chan<- *SendEvent) bool {
	if senderID == "" || events == nil {
		return false
	}

	m.eventMu.Lock()
	defer m.eventMu.Unlock()

	m.eventSubscribers[senderID] = append(m.eventSubscribers[senderID], events)
	log.Printf("[MessageManager] Event subscriber added for sender: %s", senderID)
	return true
}

// UnsubscribeFromSendEvents unsubscribes from send events for a sender.
func (m *MessageManagerImpl) UnsubscribeFromSendEvents(senderID string, events chan<- *SendEvent) bool {
	if senderID == "" || events == nil {
		return false
	}

	m.eventMu.Lock()
	defer m.eventMu.Unlock()

	subs, ok := m.eventSubscribers[senderID]
	if !ok {
		return false
	}

	// Find and remove the channel
	for i, ch := range subs {
		if ch == events {
			m.eventSubscribers[senderID] = append(subs[:i], subs[i+1:]...)
			log.Printf("[MessageManager] Event subscriber removed for sender: %s", senderID)
			return true
		}
	}

	return false
}

// GetSender retrieves a registered message sender.
func (m *MessageManagerImpl) GetSender(senderID string) (MessageSender, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sender, exists := m.senders[senderID]
	return sender, exists
}

// ListSenders lists all registered message senders.
func (m *MessageManagerImpl) ListSenders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.senders))
	for id := range m.senders {
		ids = append(ids, id)
	}
	return ids
}

// Close stops background processes and cleans up resources.
func (m *MessageManagerImpl) Close() error {
	m.mu.Lock()
	m.senders = make(map[string]MessageSender)
	m.mu.Unlock()

	m.eventMu.Lock()
	m.eventSubscribers = make(map[string][]chan<- *SendEvent)
	m.eventMu.Unlock()

	return nil
}

// notifySubscribers sends an event to all subscribers of a sender.
func (m *MessageManagerImpl) notifySubscribers(senderID string, event *SendEvent) {
	m.eventMu.RLock()
	subs := m.eventSubscribers[senderID]
	m.eventMu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
			log.Printf("[MessageManager] Event subscriber channel full for sender: %s, dropping event", senderID)
		}
	}
}

func failResult(msg string) *models.SendResult {
	return &models.SendResult{
		Success: false,
		Error:   msg,
		SentAt:  time.Now(),
	}
}
