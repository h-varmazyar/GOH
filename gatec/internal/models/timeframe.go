package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	ColumnTimeframeProviderID = "provider_id"
)

type Timeframe struct {
	gorm.Model
	Label      string
	Duration   time.Duration
	ProviderID uint
	Provider   *Provider `gorm:"foreignkey:ProviderID"`
}
