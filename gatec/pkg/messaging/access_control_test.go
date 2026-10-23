package messaging

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/h-varmazyar/gatec/internal/models"
)

// MockPersistence implements AccessPersistence for testing
type MockPersistence struct {
	rules map[uuid.UUID]*models.AccessRule
}

func NewMockPersistence() *MockPersistence {
	return &MockPersistence{
		rules: make(map[uuid.UUID]*models.AccessRule),
	}
}

func (m *MockPersistence) SaveRule(ctx context.Context, rule *models.AccessRule) error {
	m.rules[rule.ID] = rule
	return nil
}

func (m *MockPersistence) DeleteRule(ctx context.Context, ruleID uuid.UUID) error {
	delete(m.rules, ruleID)
	return nil
}

func (m *MockPersistence) LoadRules(ctx context.Context) ([]*models.AccessRule, error) {
	rules := make([]*models.AccessRule, 0, len(m.rules))
	for _, rule := range m.rules {
		rules = append(rules, rule)
	}
	return rules, nil
}

func TestAccessControlImpl_CheckAccess(t *testing.T) {
	ac := NewAccessControl(nil)

	// Test with no rules
	if ac.CheckAccess("sender1", "client1", models.PermissionSend) {
		t.Error("CheckAccess should return false when no rules exist")
	}

	// Grant access
	rule, err := ac.GrantAccessWithDetails("sender1", "client1", models.PermissionSend, "admin", "test", nil)
	if err != nil {
		t.Fatalf("GrantAccessWithDetails failed: %v", err)
	}
	if rule == nil {
		t.Fatal("GrantAccessWithDetails returned nil rule")
	}

	// Check access should now be true
	if !ac.CheckAccess("sender1", "client1", models.PermissionSend) {
		t.Error("CheckAccess should return true after granting access")
	}

	// Check wrong permission
	if ac.CheckAccess("sender1", "client1", models.PermissionSchedule) {
		t.Error("CheckAccess should return false for wrong permission")
	}

	// Check wrong client
	if ac.CheckAccess("sender1", "client2", models.PermissionSend) {
		t.Error("CheckAccess should return false for wrong client")
	}

	// Check wrong sender
	if ac.CheckAccess("sender2", "client1", models.PermissionSend) {
		t.Error("CheckAccess should return false for wrong sender")
	}
}

func TestAccessControlImpl_GrantAccess(t *testing.T) {
	ac := NewAccessControl(nil)

	// Grant access
	if !ac.GrantAccess("sender1", "client1", models.PermissionSend) {
		t.Error("GrantAccess should return true on success")
	}

	// Check access
	if !ac.CheckAccess("sender1", "client1", models.PermissionSend) {
		t.Error("CheckAccess should return true after granting access")
	}

	// Grant another permission
	if !ac.GrantAccess("sender1", "client1", models.PermissionSchedule) {
		t.Error("GrantAccess should return true on success")
	}

	// Both permissions should work now
	if !ac.CheckAccess("sender1", "client1", models.PermissionSend) {
		t.Error("CheckAccess should return true for send permission")
	}
	if !ac.CheckAccess("sender1", "client1", models.PermissionSchedule) {
		t.Error("CheckAccess should return true for schedule permission")
	}
}

func TestAccessControlImpl_RevokeAccess(t *testing.T) {
	ac := NewAccessControl(nil)

	// Grant access first
	if !ac.GrantAccess("sender1", "client1", models.PermissionSend) {
		t.Error("GrantAccess should return true on success")
	}

	// Check access is granted
	if !ac.CheckAccess("sender1", "client1", models.PermissionSend) {
		t.Error("CheckAccess should return true after granting access")
	}

	// Revoke access
	if !ac.RevokeAccess("sender1", "client1", models.PermissionSend) {
		t.Error("RevokeAccess should return true on success")
	}

	// Check access is revoked
	if ac.CheckAccess("sender1", "client1", models.PermissionSend) {
		t.Error("CheckAccess should return false after revoking access")
	}
}

func TestAccessControlImpl_UpdateRule(t *testing.T) {
	ac := NewAccessControl(nil)

	// Create a rule
	ruleID := uuid.New()
	rule := &models.AccessRule{
		ID:        ruleID,
		SenderID:  "sender1",
		ClientID:  "client1",
		GrantedAt: time.Now(),
		GrantedBy: "admin",
		Reason:    "initial",
		Permissions: models.PermissionSet{
			CanSendMessages: true,
		},
	}

	// Update non-existent rule should fail
	if ac.UpdateRule(rule) {
		t.Error("UpdateRule should return false for non-existent rule")
	}

	// Create rule first via GrantAccessWithDetails
	createdRule, err := ac.GrantAccessWithDetails("sender1", "client1", models.PermissionSend, "admin", "initial", nil)
	if err != nil {
		t.Fatalf("GrantAccessWithDetails failed: %v", err)
	}

	// Update the rule
	createdRule.Reason = "updated reason"
	createdRule.Permissions.CanScheduleMessages = true
	if !ac.UpdateRule(createdRule) {
		t.Error("UpdateRule should return true for successful update")
	}

	// Check updated permissions
	if !ac.CheckAccess("sender1", "client1", models.PermissionSend) {
		t.Error("CheckAccess should return true for send permission after update")
	}
	if !ac.CheckAccess("sender1", "client1", models.PermissionSchedule) {
		t.Error("CheckAccess should return true for schedule permission after update")
	}
}

func TestAccessControlImpl_RemoveRule(t *testing.T) {
	ac := NewAccessControl(nil)

	// Grant access first
	if !ac.GrantAccess("sender1", "client1", models.PermissionSend) {
		t.Error("GrantAccess should return true on success")
	}

	// Check access is granted
	if !ac.CheckAccess("sender1", "client1", models.PermissionSend) {
		t.Error("CheckAccess should return true after granting access")
	}

	// Remove rule
	if !ac.RemoveRule("sender1", "client1") {
		t.Error("RemoveRule should return true on success")
	}

	// Check access is removed
	if ac.CheckAccess("sender1", "client1", models.PermissionSend) {
		t.Error("CheckAccess should return false after removing rule")
	}
}

func TestAccessControlImpl_GetRule(t *testing.T) {
	ac := NewAccessControl(nil)

	// Grant access
	createdRule, err := ac.GrantAccessWithDetails("sender1", "client1", models.PermissionSend, "admin", "test", nil)
	if err != nil {
		t.Fatalf("GrantAccessWithDetails failed: %v", err)
	}

	// Get rule
	rule, found := ac.GetRule("sender1", "client1")
	if !found {
		t.Error("GetRule should find existing rule")
	}
	if rule == nil {
		t.Fatal("GetRule returned nil for existing rule")
	}
	if rule.ID != createdRule.ID {
		t.Error("GetRule returned wrong rule ID")
	}

	// Get non-existent rule
	rule, found = ac.GetRule("sender1", "client2")
	if found {
		t.Error("GetRule should not find non-existent rule")
	}
	if rule != nil {
		t.Error("GetRule should return nil for non-existent rule")
	}
}

func TestAccessControlImpl_ListRules(t *testing.T) {
	ac := NewAccessControl(nil)

	// Initially no rules
	rules := ac.ListRules()
	if len(rules) != 0 {
		t.Errorf("ListRules should return empty slice initially, got %d rules", len(rules))
	}

	// Add some rules
	ac.GrantAccess("sender1", "client1", models.PermissionSend)
	ac.GrantAccess("sender1", "client2", models.PermissionSchedule)
	ac.GrantAccess("sender2", "client1", models.PermissionSendToAllChats)

	// List rules
	rules = ac.ListRules()
	if len(rules) != 3 {
		t.Errorf("ListRules should return 3 rules, got %d", len(rules))
	}
}

func TestAccessControlImpl_Close(t *testing.T) {
	ac := NewAccessControl(nil)

	// Add some rules
	ac.GrantAccess("sender1", "client1", models.PermissionSend)
	ac.GrantAccess("sender1", "client2", models.PermissionSchedule)

	// Close
	err := ac.Close()
	if err != nil {
		t.Errorf("Close should not return error, got: %v", err)
	}

	// Note: After Close(), the audit log channel is closed, so logging will fail
	// but core functionality should still work. We don't test operations after Close
	// since the channel is closed and would cause panics.
}
