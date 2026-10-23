package models

import (
	"time"

	"gorm.io/gorm"
)

type IndicatorDefinition struct {
	ID          uint `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	MarketID    uint           `gorm:"not null;index"`
	ProviderID  uint           `gorm:"not null;index"`
	TimeframeID uint           `gorm:"not null;index"`
	Type        string         `gorm:"type:varchar(50);not null;index"`
	Params      []byte         `gorm:"type:jsonb;not null"`
	Fingerprint string         `gorm:"type:varchar(128);not null;uniqueIndex"`
}

func (IndicatorDefinition) TableName() string {
	return "indicator_definitions"
}
