package models

import (
	"strings"
	"testing"
	"time"
)

func TestBotUser_Validate(t *testing.T) {
	tests := []struct {
		name    string
		botUser BotUser
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid BotUser",
			botUser: BotUser{
				BotID:             "test-bot-123",
				ChatID:            123456789,
				StartedAt:         time.Now().Add(-24 * time.Hour),
				LastInteractionAt: time.Now(),
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
			},
			wantErr: false,
		},
		{
			name: "Empty BotID",
			botUser: BotUser{
				BotID:             "",
				ChatID:            123456789,
				StartedAt:         time.Now(),
				LastInteractionAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "botID must be non-empty",
		},
		{
			name: "Invalid ChatID (zero)",
			botUser: BotUser{
				BotID:             "test-bot-123",
				ChatID:            0,
				StartedAt:         time.Now(),
				LastInteractionAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "chatID must be a valid Telegram chat identifier (> 0)",
		},
		{
			name: "Invalid ChatID (negative)",
			botUser: BotUser{
				BotID:             "test-bot-123",
				ChatID:            -100,
				StartedAt:         time.Now(),
				LastInteractionAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "chatID must be a valid Telegram chat identifier (> 0)",
		},
		{
			name: "Zero StartedAt",
			botUser: BotUser{
				BotID:             "test-bot-123",
				ChatID:            123456789,
				StartedAt:         time.Time{},
				LastInteractionAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "startedAt must be set",
		},
		{
			name: "Zero LastInteractionAt",
			botUser: BotUser{
				BotID:             "test-bot-123",
				ChatID:            123456789,
				StartedAt:         time.Now(),
				LastInteractionAt: time.Time{},
			},
			wantErr: true,
			errMsg:  "lastInteractionAt must be set",
		},
		{
			name: "LastInteractionAt before StartedAt",
			botUser: BotUser{
				BotID:             "test-bot-123",
				ChatID:            123456789,
				StartedAt:         time.Now(),
				LastInteractionAt: time.Now().Add(-1 * time.Hour),
			},
			wantErr: true,
			errMsg:  "lastInteractionAt must not be before startedAt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.botUser.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				if tt.errMsg != "" && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message %q does not contain %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestBotUser_IsActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		botUser  BotUser
		expected bool
	}{
		{
			name: "Active user (interacted within 30 days)",
			botUser: BotUser{
				LastInteractionAt: now.Add(-15 * 24 * time.Hour),
			},
			expected: true,
		},
		{
			name: "Inactive user (interacted more than 30 days ago)",
			botUser: BotUser{
				LastInteractionAt: now.Add(-31 * 24 * time.Hour),
			},
			expected: false,
		},
		{
			name: "Active user (interacted just now)",
			botUser: BotUser{
				LastInteractionAt: now,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.botUser.IsActive()
			if result != tt.expected {
				t.Errorf("IsActive() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestBotUser_UpdateInteraction(t *testing.T) {
	botUser := BotUser{
		BotID:             "test-bot",
		ChatID:            123456789,
		StartedAt:         time.Now().Add(-48 * time.Hour),
		LastInteractionAt: time.Now().Add(-24 * time.Hour),
		CreatedAt:         time.Now().Add(-48 * time.Hour),
		UpdatedAt:         time.Now().Add(-24 * time.Hour),
	}

	// Capture initial time
	initialInteractionTime := botUser.LastInteractionAt
	initialUpdateTime := botUser.UpdatedAt

	// Sleep a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Update interaction
	botUser.UpdateInteraction()

	// Check that LastInteractionAt was updated
	if !botUser.LastInteractionAt.After(initialInteractionTime) {
		t.Errorf("LastInteractionAt was not updated")
	}

	// Note: UpdateInteraction doesn't update UpdatedAt, only LastInteractionAt
	if !botUser.UpdatedAt.Equal(initialUpdateTime) {
		t.Errorf("UpdatedAt was changed unexpectedly")
	}
}

func TestBotUser_HasStartedWithin(t *testing.T) {
	now := time.Now()

	botUser := BotUser{
		StartedAt: now.Add(-12 * time.Hour),
	}

	tests := []struct {
		name     string
		duration time.Duration
		expected bool
	}{
		{
			name:     "Started within 24 hours",
			duration: 24 * time.Hour,
			expected: true,
		},
		{
			name:     "Started within 6 hours (false)",
			duration: 6 * time.Hour,
			expected: false,
		},
		{
			name:     "Started within 48 hours",
			duration: 48 * time.Hour,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := botUser.HasStartedWithin(tt.duration)
			if result != tt.expected {
				t.Errorf("HasStartedWithin(%v) = %v, expected %v", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestBotUser_MetadataOperations(t *testing.T) {
	botUser := BotUser{
		BotID:             "test-bot",
		ChatID:            123456789,
		StartedAt:         time.Now(),
		LastInteractionAt: time.Now(),
	}

	// Test getting value from nil map
	value := botUser.GetMetadataValue("key1")
	if value != "" {
		t.Errorf("GetMetadataValue from nil map = %q, expected empty string", value)
	}

	// Test setting and getting value
	botUser.SetMetadataValue("key1", "value1")
	value = botUser.GetMetadataValue("key1")
	if value != "value1" {
		t.Errorf("GetMetadataValue after SetMetadataValue = %q, expected %q", value, "value1")
	}

	// Test setting another value
	botUser.SetMetadataValue("key2", "value2")
	if botUser.GetMetadataValue("key1") != "value1" {
		t.Errorf("GetMetadataValue(key1) = %q, expected %q", botUser.GetMetadataValue("key1"), "value1")
	}
	if botUser.GetMetadataValue("key2") != "value2" {
		t.Errorf("GetMetadataValue(key2) = %q, expected %q", botUser.GetMetadataValue("key2"), "value2")
	}

	// Test overwriting value
	botUser.SetMetadataValue("key1", "new-value1")
	if botUser.GetMetadataValue("key1") != "new-value1" {
		t.Errorf("GetMetadataValue after overwrite = %q, expected %q", botUser.GetMetadataValue("key1"), "new-value1")
	}
}

func TestBotUser_TableName(t *testing.T) {
	botUser := BotUser{}
	if tableName := botUser.TableName(); tableName != "bot_users" {
		t.Errorf("TableName() = %q, expected %q", tableName, "bot_users")
	}
}

func TestBotUser_BeforeCreate(t *testing.T) {
	botUser := BotUser{
		BotID:  "test-bot",
		ChatID: 123456789,
	}

	// Initially zero times
	if !botUser.StartedAt.IsZero() {
		t.Errorf("StartedAt should be zero initially")
	}
	if !botUser.LastInteractionAt.IsZero() {
		t.Errorf("LastInteractionAt should be zero initially")
	}
	if !botUser.CreatedAt.IsZero() {
		t.Errorf("CreatedAt should be zero initially")
	}
	if !botUser.UpdatedAt.IsZero() {
		t.Errorf("UpdatedAt should be zero initially")
	}

	// Call BeforeCreate (simulating GORM hook)
	err := botUser.BeforeCreate(nil)
	if err != nil {
		t.Errorf("BeforeCreate() error = %v", err)
	}

	// Check timestamps are set
	if botUser.StartedAt.IsZero() {
		t.Errorf("StartedAt should be set after BeforeCreate")
	}
	if botUser.LastInteractionAt.IsZero() {
		t.Errorf("LastInteractionAt should be set after BeforeCreate")
	}
	if botUser.CreatedAt.IsZero() {
		t.Errorf("CreatedAt should be set after BeforeCreate")
	}
	if botUser.UpdatedAt.IsZero() {
		t.Errorf("UpdatedAt should be set after BeforeCreate")
	}

	// Check StartedAt and LastInteractionAt are close to each other
	if botUser.LastInteractionAt.Sub(botUser.StartedAt) > time.Second {
		t.Errorf("StartedAt and LastInteractionAt should be close to each other")
	}
}

func TestBotUser_BeforeUpdate(t *testing.T) {
	initialTime := time.Now().Add(-1 * time.Hour)
	botUser := BotUser{
		BotID:             "test-bot",
		ChatID:            123456789,
		StartedAt:         initialTime,
		LastInteractionAt: initialTime,
		CreatedAt:         initialTime,
		UpdatedAt:         initialTime,
	}

	// Sleep to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Call BeforeUpdate (simulating GORM hook)
	err := botUser.BeforeUpdate(nil)
	if err != nil {
		t.Errorf("BeforeUpdate() error = %v", err)
	}

	// Check UpdatedAt was updated
	if !botUser.UpdatedAt.After(initialTime) {
		t.Errorf("UpdatedAt should be updated after BeforeUpdate")
	}

	// Check other timestamps unchanged
	if !botUser.StartedAt.Equal(initialTime) {
		t.Errorf("StartedAt should not be changed by BeforeUpdate")
	}
	if !botUser.LastInteractionAt.Equal(initialTime) {
		t.Errorf("LastInteractionAt should not be changed by BeforeUpdate")
	}
	if !botUser.CreatedAt.Equal(initialTime) {
		t.Errorf("CreatedAt should not be changed by BeforeUpdate")
	}
}
