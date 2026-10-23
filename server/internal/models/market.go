package models

import (
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
	"gorm.io/gorm"
)

type Market struct {
	gorm.Model
	Provider       *Provider `gorm:"foreignkey:ProviderID"`
	ProviderID     uint      `gorm:"uniqueIndex:idx_markets_provider_label,priority:1"`
	PricingAsset   *Asset    `gorm:"foreignkey:PricingAssetID"`
	PricingAssetID uint
	TradingAsset   *Asset `gorm:"foreignkey:TradingAssetID"`
	TradingAssetID uint
	Name           string
	Status         consts.MarketStatus
	TimeFrames     []Timeframe `gorm:"many2many:market_timeframes"`
	TSEMarkets     []TSEMarket `gorm:"foreignKey:MarketID"`
	Label          string      `gorm:"uniqueIndex:idx_markets_provider_label,priority:2"`
	LastPriceDate  *time.Time  `gorm:"column:last_price_date;type:date"`
	StateUpdatedAt *time.Time  `gorm:"column:state_updated_at"`
}
