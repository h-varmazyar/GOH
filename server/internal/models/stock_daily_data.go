package models

import (
	"time"

	"gorm.io/gorm"
)

// StockDailyData stores daily stock market data from TSETMC
type StockDailyData struct {
	gorm.Model
	MarketID     uint    `gorm:"not null;uniqueIndex:idx_market_date"`
	Market       *Market `gorm:"foreignkey:MarketID"`
	Date         string  `gorm:"type:varchar(10);not null;uniqueIndex:idx_market_date"` // Shamsi date e.g. "1404/07/12"
	Name         string  `gorm:"type:varchar(255)"`
	TradeCount   int64
	Volume       int64
	Value        int64
	Yesterday    float64
	First        float64
	Last         float64
	LastChange   float64
	LastPercent  float64
	Close        float64
	CloseChange  float64
	ClosePercent float64
	Low          float64
	High         float64
	FetchedAt    time.Time `gorm:"not null"`
}

func (StockDailyData) TableName() string {
	return "stock_daily_data"
}
