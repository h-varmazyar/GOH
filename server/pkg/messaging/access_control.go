package messaging

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/h-varmazyar/gatec/internal/models"
)

// AccessControlImpl implements AccessControl and ExtendedAccessControl interfaces.
// It provides thread-safe access control management with rule expiration handling,
// audit logging, and persistence support.
// Validates: Requirements 3.1-3.6, 13.1, 8.6
type AccessControlImpl struct {
	mu sync.RWMutex

	// rules maps (senderID, clientID) -> AccessRule for quick lookups
	rules map[string]map[string]*models.AccessRule

	// ruleIndex maps rule ID -> (senderID, clientID) for reverse lookups
	ruleIndex map[uuid.UUID][2]string

	// expiredRules tracks expired rules for cleanup
	expiredRules map[uuid.UUID]time.Time

	// auditLog channel for access control changes
	auditLog chan AccessAuditEvent

	// persistence provides optional persistent storage
	persistence AccessPersistence

	// cleanupTicker triggers expired rule cleanup
	cleanupTicker *time.Ticker
	cleanupDone   chan bool
}

// AccessPersistence defines optional persistence layer for access rules
type AccessPersistence interface {
	SaveRule(ctx context.Context, rule *models.AccessRule) error
	DeleteRule(ctx context.Context, ruleID uuid.UUID) error
	LoadRules(ctx context.Context) ([]*models.AccessRule, error)
}

// AccessAuditEvent represents an audit log entry for access control changes
type AccessAuditEvent struct {
	Timestamp  time.Time
	Action     string
	RuleID     uuid.UUID
	SenderID   string
	ClientID   string
	Permission string
	GrantedBy  string
	Reason     string
	ExpiresAt  *time.Time
	Success    bool
	Error      string
}

// NewAccessControl creates a new AccessControlImpl instance with optional persistence.
func NewAccessControl(persistence AccessPersistence) *AccessControlImpl {
	ac := &AccessControlImpl{
		rules:         make(map[string]map[string]*models.AccessRule),
		ruleIndex:     make(map[uuid.UUID][2]string),
		expiredRules:  make(map[uuid.UUID]time.Time),
		auditLog:      make(chan AccessAuditEvent, 100),
		persistence:   persistence,
		cleanupTicker: time.NewTicker(1 * time.Hour),
		cleanupDone:   make(chan bool),
	}

	// Start background cleanup goroutine
	go ac.cleanupExpiredRules()

	// Start audit log processor
	go ac.processAuditLog()

	return ac
}

// ===== AccessControl Interface Implementation =====

// CheckAccess implements AccessControl.CheckAccess
// Validates: Requirements 3.1, 3.6, 13.1, 8.6
func (ac *AccessControlImpl) CheckAccess(senderID, clientID string, permission models.Permission) bool {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	// Get rules for sender
	clientRules, ok := ac.rules[senderID]
	if !ok {
		ac.logAuditEvent(AccessAuditEvent{
			Timestamp:  time.Now(),
			Action:     "check_access",
			SenderID:   senderID,
			ClientID:   clientID,
			Permission: string(permission),
			Success:    false,
			Error:      "no rules found for sender",
		})
		return false
	}

	// Get specific rule for client
	rule, ok := clientRules[clientID]
	if !ok {
		ac.logAuditEvent(AccessAuditEvent{
			Timestamp:  time.Now(),
			Action:     "check_access",
			SenderID:   senderID,
			ClientID:   clientID,
			Permission: string(permission),
			Success:    false,
			Error:      "no rule found for client",
		})
		return false
	}

	// Check if rule is expired
	if rule.IsExpired() {
		ac.logAuditEvent(AccessAuditEvent{
			Timestamp:  time.Now(),
			Action:     "check_access",
			SenderID:   senderID,
			ClientID:   clientID,
			Permission: string(permission),
			Success:    false,
			Error:      "rule expired",
		})
		return false
	}

	// Check specific permission
	hasPermission := false
	switch permission {
	case models.PermissionSend:
		hasPermission = rule.Permissions.CanSendMessages
	case models.PermissionSchedule:
		hasPermission = rule.Permissions.CanScheduleMessages
	case models.PermissionSendToAllChats:
		hasPermission = rule.Permissions.CanSendToAllChats
	default:
		hasPermission = false
	}

	ac.logAuditEvent(AccessAuditEvent{
		Timestamp:  time.Now(),
		Action:     "check_access",
		RuleID:     rule.ID,
		SenderID:   senderID,
		ClientID:   clientID,
		Permission: string(permission),
		Success:    hasPermission,
		Error:      "",
	})

	return hasPermission
}

// GrantAccess implements AccessControl.GrantAccess
// Validates: Requirements 3.2, 3.4, 3.5
func (ac *AccessControlImpl) GrantAccess(senderID, clientID string, permission models.Permission) bool {
	// Use default values for grantedBy, reason, and no expiration
	rule, err := ac.GrantAccessWithDetails(senderID, clientID, permission, "system", "automatic grant", nil)
	return err == nil && rule != nil
}

// RevokeAccess implements AccessControl.RevokeAccess
// Validates: Requirements 3.3, 3.4
func (ac *AccessControlImpl) RevokeAccess(senderID, clientID string, permission models.Permission) bool {
	success, err := ac.revokeAccessInternal(senderID, clientID)
	return err == nil && success
}

// UpdateRule implements AccessControl.UpdateRule
// Validates: Requirements 3.4, 3.5
func (ac *AccessControlImpl) UpdateRule(rule *models.AccessRule) bool {
	success, err := ac.updateRuleInternal(rule)
	return err == nil && success
}

// RemoveRule implements AccessControl.RemoveRule
// Validates: Requirements 3.3, 3.4
func (ac *AccessControlImpl) RemoveRule(senderID, clientID string) bool {
	success, err := ac.removeRuleInternal(senderID, clientID)
	return err == nil && success
}

// ===== ExtendedAccessControl Interface Implementation =====

// GrantAccessWithDetails implements ExtendedAccessControl.GrantAccessWithDetails
// Validates: Requirements 3.2, 3.4, 3.5
func (ac *AccessControlImpl) GrantAccessWithDetails(senderID, clientID string, permission models.Permission, grantedBy, reason string, expiresAt *time.Time) (*models.AccessRule, error) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// Check if rule already exists
	clientRules, ok := ac.rules[senderID]
	if ok {
		if existingRule, exists := clientRules[clientID]; exists {
			// Update existing rule with new permission
			return ac.updateRulePermissions(existingRule, permission, grantedBy, reason, expiresAt)
		}
	}

	// Create new rule
	rule := &models.AccessRule{
		ID:        uuid.New(),
		SenderID:  senderID,
		ClientID:  clientID,
		GrantedAt: time.Now(),
		GrantedBy: grantedBy,
		ExpiresAt: expiresAt,
		Reason:    reason,
		Permissions: models.PermissionSet{
			CanSendMessages:      permission == models.PermissionSend,
			CanScheduleMessages:  permission == models.PermissionSchedule,
			CanSendToAllChats:    permission == models.PermissionSendToAllChats,
			MaxMessagesPerMinute: 30, // Default value
			AllowedChatIDs:       []int64{},
		},
	}

	// Validate the rule
	if err := rule.Validate(); err != nil {
		ac.logAuditEvent(AccessAuditEvent{
			Timestamp: time.Now(),
			Action:    "grant_access",
			SenderID:  senderID,
			ClientID:  clientID,
			GrantedBy: grantedBy,
			Reason:    reason,
			ExpiresAt: expiresAt,
			Success:   false,
			Error:     fmt.Sprintf("validation failed: %v", err),
		})
		return nil, fmt.Errorf("invalid access rule: %w", err)
	}

	// Save to persistence if available
	if ac.persistence != nil {
		if err := ac.persistence.SaveRule(context.Background(), rule); err != nil {
			ac.logAuditEvent(AccessAuditEvent{
				Timestamp: time.Now(),
				Action:    "grant_access",
				SenderID:  senderID,
				ClientID:  clientID,
				GrantedBy: grantedBy,
				Reason:    reason,
				ExpiresAt: expiresAt,
				Success:   false,
				Error:     fmt.Sprintf("persistence failed: %v", err),
			})
			return nil, fmt.Errorf("failed to save rule: %w", err)
		}
	}

	// Add to in-memory storage
	ac.addRuleInternal(rule)

	ac.logAuditEvent(AccessAuditEvent{
		Timestamp: time.Now(),
		Action:    "grant_access",
		RuleID:    rule.ID,
		SenderID:  senderID,
		ClientID:  clientID,
		GrantedBy: grantedBy,
		Reason:    reason,
		ExpiresAt: expiresAt,
		Success:   true,
		Error:     "",
	})

	return rule, nil
}

// RevokeAccessByID implements ExtendedAccessControl.RevokeAccessByID
func (ac *AccessControlImpl) RevokeAccessByID(ruleID uuid.UUID) (bool, error) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// Find rule by ID
	key, exists := ac.ruleIndex[ruleID]
	if !exists {
		ac.logAuditEvent(AccessAuditEvent{
			Timestamp: time.Now(),
			Action:    "revoke_access_by_id",
			RuleID:    ruleID,
			Success:   false,
			Error:     "rule not found",
		})
		return false, nil
	}

	// Get the rule
	clientRules, ok := ac.rules[key[0]]
	if !ok {
		ac.logAuditEvent(AccessAuditEvent{
			Timestamp: time.Now(),
			Action:    "revoke_access_by_id",
			RuleID:    ruleID,
			Success:   false,
			Error:     "sender not found",
		})
		return false, nil
	}

	rule, ok := clientRules[key[1]]
	if !ok {
		ac.logAuditEvent(AccessAuditEvent{
			Timestamp: time.Now(),
			Action:    "revoke_access_by_id",
			RuleID:    ruleID,
			Success:   false,
			Error:     "client not found",
		})
		return false, nil
	}

	// Remove from persistence if available
	if ac.persistence != nil {
		if err := ac.persistence.DeleteRule(context.Background(), rule.ID); err != nil {
			ac.logAuditEvent(AccessAuditEvent{
				Timestamp: time.Now(),
				Action:    "revoke_access_by_id",
				RuleID:    ruleID,
				Success:   false,
				Error:     fmt.Sprintf("persistence delete failed: %v", err),
			})
			return false, fmt.Errorf("failed to delete rule from persistence: %w", err)
		}
	}

	// Remove from in-memory storage
	ac.removeRuleInternalHelper(key[0], key[1], ruleID)

	ac.logAuditEvent(AccessAuditEvent{
		Timestamp: time.Now(),
		Action:    "revoke_access_by_id",
		RuleID:    ruleID,
		Success:   true,
		Error:     "",
	})

	return true, nil
}

// GetRule implements ExtendedAccessControl.GetRule
func (ac *AccessControlImpl) GetRule(senderID, clientID string) (*models.AccessRule, bool) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	clientRules, ok := ac.rules[senderID]
	if !ok {
		return nil, false
	}

	rule, ok := clientRules[clientID]
	if !ok {
		return nil, false
	}

	// Check if expired
	if rule.IsExpired() {
		return nil, false
	}

	return rule, true
}

// ListRules implements ExtendedAccessControl.ListRules
func (ac *AccessControlImpl) ListRules() []*models.AccessRule {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	rules := make([]*models.AccessRule, 0)
	now := time.Now()

	for _, clientRules := range ac.rules {
		for _, rule := range clientRules {
			// Skip expired rules
			if rule.ExpiresAt != nil && rule.ExpiresAt.Before(now) {
				continue
			}
			rules = append(rules, rule)
		}
	}

	return rules
}

// LoadRules implements ExtendedAccessControl.LoadRules
func (ac *AccessControlImpl) LoadRules(ctx context.Context) error {
	if ac.persistence == nil {
		return nil
	}

	rules, err := ac.persistence.LoadRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	ac.mu.Lock()
	defer ac.mu.Unlock()

	for _, rule := range rules {
		// Skip expired rules
		if rule.IsExpired() {
			ac.expiredRules[rule.ID] = time.Now()
			continue
		}

		ac.addRuleInternal(rule)
	}

	return nil
}

// Close implements ExtendedAccessControl.Close
func (ac *AccessControlImpl) Close() error {
	ac.cleanupTicker.Stop()
	ac.cleanupDone <- true
	close(ac.auditLog)
	return nil
}

// ===== Internal Helper Methods =====

func (ac *AccessControlImpl) revokeAccessInternal(senderID, clientID string) (bool, error) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// Find the rule
	clientRules, ok := ac.rules[senderID]
	if !ok {
		ac.logAuditEvent(AccessAuditEvent{
			Timestamp: time.Now(),
			Action:    "revoke_access",
			SenderID:  senderID,
			ClientID:  clientID,
			Success:   false,
			Error:     "no rules found for sender",
		})
		return false, nil
	}

	rule, ok := clientRules[clientID]
	if !ok {
		ac.logAuditEvent(AccessAuditEvent{
			Timestamp: time.Now(),
			Action:    "revoke_access",
			SenderID:  senderID,
			ClientID:  clientID,
			Success:   false,
			Error:     "no rule found for client",
		})
		return false, nil
	}

	// Remove from persistence if available
	if ac.persistence != nil {
		if err := ac.persistence.DeleteRule(context.Background(), rule.ID); err != nil {
			ac.logAuditEvent(AccessAuditEvent{
				Timestamp: time.Now(),
				Action:    "revoke_access",
				RuleID:    rule.ID,
				SenderID:  senderID,
				ClientID:  clientID,
				Success:   false,
				Error:     fmt.Sprintf("persistence delete failed: %v", err),
			})
			return false, fmt.Errorf("failed to delete rule from persistence: %w", err)
		}
	}

	// Remove from in-memory storage
	ac.removeRuleInternalHelper(senderID, clientID, rule.ID)

	ac.logAuditEvent(AccessAuditEvent{
		Timestamp: time.Now(),
		Action:    "revoke_access",
		RuleID:    rule.ID,
		SenderID:  senderID,
		ClientID:  clientID,
		Success:   true,
		Error:     "",
	})

	return true, nil
}

func (ac *AccessControlImpl) updateRuleInternal(rule *models.AccessRule) (bool, error) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// Validate the rule
	if err := rule.Validate(); err != nil {
		ac.logAuditEvent(AccessAuditEvent{
			Timestamp: time.Now(),
			Action:    "update_rule",
			RuleID:    rule.ID,
			SenderID:  rule.SenderID,
			ClientID:  rule.ClientID,
			Success:   false,
			Error:     fmt.Sprintf("validation failed: %v", err),
		})
		return false, fmt.Errorf("invalid access rule: %w", err)
	}

	// Check if rule exists
	key := [2]string{rule.SenderID, rule.ClientID}
	existingKey, exists := ac.ruleIndex[rule.ID]
	if !exists {
		ac.logAuditEvent(AccessAuditEvent{
			Timestamp: time.Now(),
			Action:    "update_rule",
			RuleID:    rule.ID,
			SenderID:  rule.SenderID,
			ClientID:  rule.ClientID,
			Success:   false,
			Error:     "rule not found",
		})
		return false, nil
	}

	// If senderID or clientID changed, we need to move the rule
	if existingKey != key {
		// Remove from old location
		ac.removeRuleInternalHelper(existingKey[0], existingKey[1], rule.ID)
	}

	// Check if already expired
	if rule.IsExpired() {
		ac.expiredRules[rule.ID] = time.Now()
		ac.logAuditEvent(AccessAuditEvent{
			Timestamp: time.Now(),
			Action:    "update_rule",
			RuleID:    rule.ID,
			SenderID:  rule.SenderID,
			ClientID:  rule.ClientID,
			Success:   false,
			Error:     "rule expired",
		})
		return false, nil
	}

	// Save to persistence if available
	if ac.persistence != nil {
		if err := ac.persistence.SaveRule(context.Background(), rule); err != nil {
			ac.logAuditEvent(AccessAuditEvent{
				Timestamp: time.Now(),
				Action:    "update_rule",
				RuleID:    rule.ID,
				SenderID:  rule.SenderID,
				ClientID:  rule.ClientID,
				Success:   false,
				Error:     fmt.Sprintf("persistence failed: %v", err),
			})
			return false, fmt.Errorf("failed to save rule: %w", err)
		}
	}

	// Add to in-memory storage at new location
	ac.addRuleInternal(rule)

	ac.logAuditEvent(AccessAuditEvent{
		Timestamp: time.Now(),
		Action:    "update_rule",
		RuleID:    rule.ID,
		SenderID:  rule.SenderID,
		ClientID:  rule.ClientID,
		Success:   true,
		Error:     "",
	})

	return true, nil
}

func (ac *AccessControlImpl) removeRuleInternal(senderID, clientID string) (bool, error) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// Find the rule
	clientRules, ok := ac.rules[senderID]
	if !ok {
		ac.logAuditEvent(AccessAuditEvent{
			Timestamp: time.Now(),
			Action:    "remove_rule",
			SenderID:  senderID,
			ClientID:  clientID,
			Success:   false,
			Error:     "no rules found for sender",
		})
		return false, nil
	}

	rule, ok := clientRules[clientID]
	if !ok {
		ac.logAuditEvent(AccessAuditEvent{
			Timestamp: time.Now(),
			Action:    "remove_rule",
			SenderID:  senderID,
			ClientID:  clientID,
			Success:   false,
			Error:     "no rule found for client",
		})
		return false, nil
	}

	// Remove from persistence if available
	if ac.persistence != nil {
		if err := ac.persistence.DeleteRule(context.Background(), rule.ID); err != nil {
			ac.logAuditEvent(AccessAuditEvent{
				Timestamp: time.Now(),
				Action:    "remove_rule",
				RuleID:    rule.ID,
				SenderID:  senderID,
				ClientID:  clientID,
				Success:   false,
				Error:     fmt.Sprintf("persistence delete failed: %v", err),
			})
			return false, fmt.Errorf("failed to delete rule from persistence: %w", err)
		}
	}

	// Remove from in-memory storage
	ac.removeRuleInternalHelper(senderID, clientID, rule.ID)

	ac.logAuditEvent(AccessAuditEvent{
		Timestamp: time.Now(),
		Action:    "remove_rule",
		RuleID:    rule.ID,
		SenderID:  senderID,
		ClientID:  clientID,
		Success:   true,
		Error:     "",
	})

	return true, nil
}

func (ac *AccessControlImpl) addRuleInternal(rule *models.AccessRule) {
	// Initialize client map for sender if needed
	if _, ok := ac.rules[rule.SenderID]; !ok {
		ac.rules[rule.SenderID] = make(map[string]*models.AccessRule)
	}

	// Add rule to client map
	ac.rules[rule.SenderID][rule.ClientID] = rule

	// Add to index
	ac.ruleIndex[rule.ID] = [2]string{rule.SenderID, rule.ClientID}

	// Remove from expired rules if it was there
	delete(ac.expiredRules, rule.ID)
}

func (ac *AccessControlImpl) removeRuleInternalHelper(senderID, clientID string, ruleID uuid.UUID) {
	// Remove from client map
	if clientRules, ok := ac.rules[senderID]; ok {
		delete(clientRules, clientID)
		if len(clientRules) == 0 {
			delete(ac.rules, senderID)
		}
	}

	// Remove from index
	delete(ac.ruleIndex, ruleID)

	// Remove from expired rules
	delete(ac.expiredRules, ruleID)
}

func (ac *AccessControlImpl) updateRulePermissions(rule *models.AccessRule, permission models.Permission, grantedBy, reason string, expiresAt *time.Time) (*models.AccessRule, error) {
	// Update permissions based on the permission type
	switch permission {
	case models.PermissionSend:
		rule.Permissions.CanSendMessages = true
	case models.PermissionSchedule:
		rule.Permissions.CanScheduleMessages = true
	case models.PermissionSendToAllChats:
		rule.Permissions.CanSendToAllChats = true
	}

	// Update metadata
	rule.GrantedBy = grantedBy
	rule.Reason = reason
	if expiresAt != nil {
		rule.ExpiresAt = expiresAt
	}

	// Validate the updated rule
	if err := rule.Validate(); err != nil {
		return nil, fmt.Errorf("invalid updated rule: %w", err)
	}

	// Save to persistence if available
	if ac.persistence != nil {
		if err := ac.persistence.SaveRule(context.Background(), rule); err != nil {
			return nil, fmt.Errorf("failed to save updated rule: %w", err)
		}
	}

	ac.logAuditEvent(AccessAuditEvent{
		Timestamp: time.Now(),
		Action:    "update_permissions",
		RuleID:    rule.ID,
		SenderID:  rule.SenderID,
		ClientID:  rule.ClientID,
		GrantedBy: grantedBy,
		Reason:    reason,
		ExpiresAt: expiresAt,
		Success:   true,
		Error:     "",
	})

	return rule, nil
}

func (ac *AccessControlImpl) cleanupExpiredRules() {
	for {
		select {
		case <-ac.cleanupTicker.C:
			ac.mu.Lock()
			now := time.Now()
			toRemove := make([]uuid.UUID, 0)

			// Find expired rules
			for ruleID, expiredTime := range ac.expiredRules {
				if now.Sub(expiredTime) > 24*time.Hour { // Keep for 24 hours for audit
					toRemove = append(toRemove, ruleID)
				}
			}

			// Remove from memory
			for _, ruleID := range toRemove {
				if key, ok := ac.ruleIndex[ruleID]; ok {
					if clientRules, ok := ac.rules[key[0]]; ok {
						delete(clientRules, key[1])
						if len(clientRules) == 0 {
							delete(ac.rules, key[0])
						}
					}
					delete(ac.ruleIndex, ruleID)
					delete(ac.expiredRules, ruleID)
				}
			}
			ac.mu.Unlock()

		case <-ac.cleanupDone:
			return
		}
	}
}

func (ac *AccessControlImpl) processAuditLog() {
	for event := range ac.auditLog {
		// In a real implementation, this would write to a proper audit log
		// For now, just log to stdout with structured logging
		log.Printf("[AccessControl] %s - Sender:%s Client:%s Rule:%s Success:%v Error:%s",
			event.Action, event.SenderID, event.ClientID, event.RuleID.String(), event.Success, event.Error)
	}
}

func (ac *AccessControlImpl) logAuditEvent(event AccessAuditEvent) {
	defer func() {
		if r := recover(); r != nil {
			// Channel is closed, log directly without panic
			log.Printf("[AccessControl] AUDIT LOG (channel closed) - %s - Sender:%s Client:%s Success:%v",
				event.Action, event.SenderID, event.ClientID, event.Success)
		}
	}()

	select {
	case ac.auditLog <- event:
		// Event queued
	default:
		// Channel full, log directly
		log.Printf("[AccessControl] AUDIT LOG DROPPED - %s - Sender:%s Client:%s",
			event.Action, event.SenderID, event.ClientID)
	}
}
