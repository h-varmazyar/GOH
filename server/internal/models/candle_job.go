package models

import (
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
)

// CandleJob builds the candles of markets from their stored trades, queued from
// the panel or by the job-scheduler worker and run by the candle-builder
// worker. The markets and timeframes belong to the provider of the job.
type CandleJob struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Status    consts.JobStatus `gorm:"type:varchar(16);not null;index"`
	// Title describes the job in the panel.
	Title      string
	ProviderID uint `gorm:"not null;index"`
	// AllMarkets selects every market of the provider whose data collection is
	// enabled, read when the job starts; otherwise MarketIDs are built.
	AllMarkets   bool
	MarketIDs    []uint `gorm:"type:jsonb;serializer:json"`
	TimeframeIDs []uint `gorm:"type:jsonb;serializer:json"`
	// FromDate and ToDate bound the trading days whose candles are built; nil
	// is unbounded. The range is widened to whole candles, so a weekly candle
	// is built from every day of its week even when the job selects one day.
	FromDate *time.Time `gorm:"type:date"`
	ToDate   *time.Time `gorm:"type:date"`
	// AfterTradeHistoryJobID holds the job until that trade history job has
	// finished, so its candles are built from the trades it fetched.
	AfterTradeHistoryJobID *uint `gorm:"index"`

	TotalMarkets int `gorm:"not null;default:0"`
	DoneMarkets  int `gorm:"not null;default:0"`
	// EmptyMarkets had no trades in the range; FailedMarkets could not be built.
	EmptyMarkets  int   `gorm:"not null;default:0"`
	FailedMarkets int   `gorm:"not null;default:0"`
	Candles       int64 `gorm:"not null;default:0"`
	// CurrentMarket is the label of a market being built, for the panel.
	CurrentMarket   string
	CancelRequested bool   `gorm:"not null;default:false"`
	Error           string `gorm:"type:text"`
	Worker          string
	StartedAt       *time.Time
	HeartbeatAt     *time.Time
	FinishedAt      *time.Time
}

func (CandleJob) TableName() string { return "candle_jobs" }

// ScheduledRun records a run of a daily schedule, so a run is made once per day
// whatever the number of processes, and a missed run can be made up.
type ScheduledRun struct {
	Name      string    `gorm:"primaryKey;type:varchar(64)"`
	RunDate   time.Time `gorm:"primaryKey;type:date"`
	CreatedAt time.Time
	// Detail lists what the run queued.
	Detail string `gorm:"type:text"`
}

func (ScheduledRun) TableName() string { return "scheduled_runs" }

// MarketCandle is a candle stored in ClickHouse, built from the trades of a
// market. Time is the Unix time the candle opens at; Closed is false while the
// candle is still open, such as the weekly candle of the current week, and the
// candle is replaced when it is built again.
type MarketCandle struct {
	ProviderID  uint64  `json:"provider_id"`
	MarketID    uint64  `json:"market_id"`
	TimeframeID uint64  `json:"timeframe_id"`
	Time        int64   `json:"time"`
	Open        float64 `json:"open"`
	High        float64 `json:"high"`
	Low         float64 `json:"low"`
	Close       float64 `json:"close"`
	Volume      float64 `json:"volume"`
	Value       float64 `json:"value"`
	TradeCount  uint64  `json:"trade_count"`
	Closed      bool    `json:"closed"`
}

func (MarketCandle) TableName() string { return "market_candles" }
