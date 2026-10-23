package models

import "gorm.io/gorm"

type TelegramSubscriptionFilter struct {
	gorm.Model
	AccountID string `gorm:"not null;index" json:"account_id"`
	ChatID    int64  `gorm:"not null;index" json:"chat_id"`
	IsActive  bool   `gorm:"not null;default:true" json:"is_active"`
}
