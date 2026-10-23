package models

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Candle struct {
	gorm.Model
	MarketID    uint    `gorm:"not null;uniqueIndex:idx_uq"`
	Market      *Market `gorm:"foreignkey:MarketID"`
	TimeframeID uint
	Timeframe   *Timeframe `gorm:"foreignkey:timeframeID"`
	Time        int64      `gorm:"not null;uniqueIndex:idx_uq"`
	Open        decimal.Decimal
	High        decimal.Decimal
	Low         decimal.Decimal
	Close       decimal.Decimal
	Volume      decimal.Decimal
}
