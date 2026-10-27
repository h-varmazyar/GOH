package models

import (
	"gorm.io/gorm"
)

type TabdealAirdrop struct {
	gorm.Model
	Phrase string `gorm:"uniqueIndex;not null"`
	Route  string `gorm:"not null"`
}

type TabdealAirdropGift struct {
	gorm.Model
	TelegramMessageID int64  `gorm:"index"`
	TelegramChatID    int64  `gorm:"index"`
	RawMessage        string `gorm:"type:text;not null"`
	SearchPhrase      string `gorm:"index"`
	Route             string
	GiftCode          string                  `gorm:"index"`
	Status            string                  `gorm:"index;not null"`
	ErrorMessage      string                  `gorm:"type:text"`
	Allocations       []*TabdealGitAllocation `gorm:"foreignKey:GiftID"`
}

type TabdealGitAllocation struct {
	gorm.Model
	GiftID       uint
	Gift         *TabdealAirdropGift `gorm:"foreignKey:GiftID"`
	Currency     string
	Status       string `gorm:"index;not null"`
	ErrorMessage string `gorm:"type:text"`
	UserID       uint
}
