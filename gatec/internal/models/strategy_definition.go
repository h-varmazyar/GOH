package models

import "time"

// StrategyDefinition stores the shared strict JSON schema used by file and DB sources.
type StrategyDefinition struct {
	ID         string    `gorm:"primaryKey;type:varchar(128)"`
	Version    int       `gorm:"not null"`
	Enabled    bool      `gorm:"not null;default:false;index"`
	Market     string    `gorm:"type:varchar(64);not null;index"`
	Definition []byte    `gorm:"type:jsonb;not null"`
	UpdatedAt  time.Time `gorm:"not null;index"`
}

func (StrategyDefinition) TableName() string { return "strategy_definitions" }
