package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TelegramAccountStatus string

const (
	TelegramAccountStatusReady        TelegramAccountStatus = "ready"
	TelegramAccountStatusNeedsLogin   TelegramAccountStatus = "needs_login"
	TelegramAccountStatusPending      TelegramAccountStatus = "pending"
	TelegramAccountStatusDisconnected TelegramAccountStatus = "disconnected"
)

type TelegramAccount struct {
	ID                uuid.UUID `gorm:"primarykey"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt        `gorm:"index"`
	Role              string                `gorm:"not null"`
	Status            TelegramAccountStatus `gorm:"not null;default:'pending'"`
	DatabaseDirectory string                `gorm:"not null"`
	FilesDirectory    string                `gorm:"not null"`
	BotToken          *string               `gorm:"uniqueIndex:idx_unique_bot_token,where:bot_token IS NOT NULL"`
	Mobile            *string               `gorm:"uniqueIndex:idx_unique_mobile,where:mobile IS NOT NULL"`
	Username          string
	AccountName       string
	UserID            int64 `gorm:"uniqueIndex:idx_unique_user_id,where:user_id != 0" json:"user_id"`
}

// BeforeCreate و BeforeUpdate برای validation در سطح application
func (a *TelegramAccount) BeforeCreate(tx *gorm.DB) error {
	return a.validate()
}

func (a *TelegramAccount) BeforeUpdate(tx *gorm.DB) error {
	return a.validate()
}

func (a *TelegramAccount) validate() error {
	hasBotToken := a.BotToken != nil && *a.BotToken != ""
	hasMobile := a.Mobile != nil && *a.Mobile != ""

	// قانون ۱: نمی‌توانند همزمان وجود داشته باشند
	if hasBotToken && hasMobile {
		return errors.New("bot_token and mobile cannot both be set (mutually exclusive)")
	}

	// قانون ۲: حداقل یکی باید وجود داشته باشد
	if !hasBotToken && !hasMobile {
		return errors.New("either bot_token or mobile must be provided")
	}

	return nil
}
