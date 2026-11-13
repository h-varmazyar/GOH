package models

import (
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
)

// TSETradeHistoryJob is a trade history collection requested from the panel. The
// API only queues jobs; the tse-trade-history worker claims and runs them.
type TSETradeHistoryJob struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Status    consts.TSETradeHistoryJobStatus `gorm:"type:varchar(16);not null;index"`
	// Title describes the job in the panel.
	Title string
	// AllMarkets selects every market whose data collection is enabled, read when the job starts.
	AllMarkets   bool
	TSEMarketIDs []uint `gorm:"type:jsonb;serializer:json"`
	// Dates are fetched whatever their status. When empty, the days between
	// FromDate and ToDate whose status is in DayStatuses are fetched. The days of
	// a market are its daily closing price rows.
	Dates       []string `gorm:"type:jsonb;serializer:json"`
	FromDate    *time.Time `gorm:"type:date"`
	ToDate      *time.Time `gorm:"type:date"`
	DayStatuses []consts.TSETradeDayStatus `gorm:"type:jsonb;serializer:json"`

	TotalMarkets int `gorm:"not null;default:0"`
	DoneMarkets  int `gorm:"not null;default:0"`
	// PlannedDays grows as the worker plans the days of each market.
	PlannedDays   int64 `gorm:"not null;default:0"`
	ProcessedDays int64 `gorm:"not null;default:0"`
	FetchedDays   int64 `gorm:"not null;default:0"`
	EmptyDays     int64 `gorm:"not null;default:0"`
	FailedDays    int64 `gorm:"not null;default:0"`
	// CurrentMarket is the label of a market being fetched, for the panel.
	CurrentMarket   string
	CancelRequested bool `gorm:"not null;default:false"`
	Error           string `gorm:"type:text"`
	Worker          string
	StartedAt       *time.Time
	HeartbeatAt     *time.Time
	FinishedAt      *time.Time
}

func (TSETradeHistoryJob) TableName() string { return "tse_trade_history_jobs" }
