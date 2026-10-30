package models

import "time"

type TSEMarket struct {
	ID                  uint    `gorm:"primaryKey"`
	MarketID            uint    `gorm:"not null;index"`
	Market              *Market `gorm:"foreignKey:MarketID"`
	InsCode             string  `gorm:"column:ins_code;unique"`
	Label               string
	Nav                 *float64 `gorm:"column:nav"`
	EPS                 *float64 `gorm:"column:eps"`
	InstrumentStateCode string   `gorm:"column:instrument_state_code"`
	MarketType          string
	SymbolType          string
	IndustryGroupCode   string
	IndustrySubCode     string
	IPODate             time.Time
}

func (TSEMarket) TableName() string { return "tse_markets" }
