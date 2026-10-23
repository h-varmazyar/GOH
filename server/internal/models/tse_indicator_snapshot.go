package models

import "time"

// TSEIndicatorSnapshot stores all calculated indicator values for one market day.
type TSEIndicatorSnapshot struct {
	ID         uint64    `gorm:"primaryKey"`
	MarketID   uint64    `gorm:"not null;uniqueIndex:idx_tse_indicator_snapshot_market_date"`
	ProviderID uint      `gorm:"not null;index"`
	Date       time.Time `gorm:"type:date;not null;uniqueIndex:idx_tse_indicator_snapshot_market_date"`
	Results    []byte    `gorm:"type:jsonb;not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (TSEIndicatorSnapshot) TableName() string { return "tse_indicator_snapshots" }
