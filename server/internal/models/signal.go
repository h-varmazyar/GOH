package models

import "time"

type SignalRecord struct {
	SignalID     string    `gorm:"primaryKey;type:char(64)"`
	StrategyID   string    `gorm:"type:varchar(128);not null;index:idx_signal_strategy_status,priority:1"`
	Market       string    `gorm:"type:varchar(64);not null;index"`
	Symbol       string    `gorm:"type:varchar(128);not null;index"`
	Timeframe    string    `gorm:"type:varchar(16);not null"`
	Type         string    `gorm:"type:varchar(16);not null;index"`
	Mode         string    `gorm:"type:varchar(16);not null;index"`
	Status       string    `gorm:"type:varchar(16);not null;index:idx_signal_strategy_status,priority:2"`
	ExitSignalID *string   `gorm:"type:char(64)"`
	PnLPercent   *string   `gorm:"type:numeric(20,8)"`
	Published    bool      `gorm:"not null;default:false;index"`
	Payload      []byte    `gorm:"type:jsonb;not null"`
	CreatedAt    time.Time `gorm:"not null;index"`
	UpdatedAt    time.Time `gorm:"not null"`
}

func (SignalRecord) TableName() string { return "signals" }
