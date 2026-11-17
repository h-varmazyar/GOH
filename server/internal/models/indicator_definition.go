package models

import (
	"time"

	"gorm.io/gorm"
)

type IndicatorDefinition struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	// MarketID is the market of a definition of one market; 0 for every
	// market of the provider and for a group of markets.
	MarketID uint `gorm:"not null;index"`
	// MarketIDs are the markets of a definition of a group of markets, sorted;
	// empty otherwise.
	MarketIDs   []uint `gorm:"type:jsonb;serializer:json"`
	ProviderID  uint   `gorm:"not null;index"`
	TimeframeID uint   `gorm:"not null;index"`
	Type        string `gorm:"type:varchar(50);not null;index"`
	Params      []byte `gorm:"type:jsonb;not null"`
	Fingerprint string `gorm:"type:varchar(128);not null;uniqueIndex"`
	// Enabled definitions are calculated; a disabled one is kept with its
	// params but skipped.
	Enabled bool `gorm:"not null;default:true;index"`
}

func (IndicatorDefinition) TableName() string {
	return "indicator_definitions"
}
