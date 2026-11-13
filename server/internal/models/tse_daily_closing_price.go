package models

import (
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
)

// TSEDailyClosingPrice is one trading day of a TSE market, stored by the
// tse-market-data-sync worker. The History fields are filled by the
// tse-trade-history worker from the individual trades of the day (stored in
// ClickHouse), so both sources of the day can be compared.
type TSEDailyClosingPrice struct {
	ID                      uint       `gorm:"primaryKey"`
	MarketID                uint       `gorm:"index"`
	Market                  *Market    `gorm:"foreignKey:MarketID"`
	TSEMarketID             uint       `gorm:"index;uniqueIndex:idx_tse_daily_tse_market_candle"`
	TSEMarket               *TSEMarket `gorm:"foreignKey:TSEMarketID"`
	CandleID                uint       `gorm:"uniqueIndex:idx_tse_daily_tse_market_candle"`
	Candle                  Candle     `gorm:"foreignKey:CandleID"`
	TradeCount              int64
	Volume                  float64
	Value                   int64
	First                   float64
	Yesterday               float64
	Last                    float64
	LastChange              float64
	LastChangePercent       float64
	Closing                 float64
	CloseChange             float64
	CloseChangePercent      float64
	Low                     float64
	High                    float64
	InstrumentStateCode     string
	BuyIndividualVolume     int64
	BuyIndividualValue      int64
	BuyIndividualCount      int64
	BuyInstitutionalVolume  int64
	BuyInstitutionalValue   int64
	BuyInstitutionalCount   int64
	SellIndividualVolume    int64
	SellIndividualValue     int64
	SellIndividualCount     int64
	SellInstitutionalVolume int64
	SellInstitutionalValue  int64
	SellInstitutionalCount  int64

	// HistoryStatus is the result of the last trade history fetch of the day;
	// nil when the trade history of the day was not fetched yet.
	HistoryStatus *consts.TSETradeDayStatus `gorm:"type:varchar(16)"`
	// HistoryTradeCount counts the received trades, canceled trades included.
	HistoryTradeCount    int64 `gorm:"not null;default:0"`
	HistoryCanceledCount int64 `gorm:"not null;default:0"`
	// HistoryVolume sums the quantity of every received trade, canceled included;
	// HistoryNetVolume and HistoryNetValue leave canceled trades out.
	HistoryVolume    int64 `gorm:"not null;default:0"`
	HistoryNetVolume int64 `gorm:"not null;default:0"`
	HistoryNetValue  int64 `gorm:"not null;default:0"`
	// HistoryAttempts counts the requests of all trade history fetches of the day.
	HistoryAttempts  int    `gorm:"not null;default:0"`
	HistoryError     string `gorm:"type:text;not null;default:''"`
	HistoryJobID     *uint
	HistoryFetchedAt *time.Time
}

func (TSEDailyClosingPrice) TableName() string { return "tse_daily_closing_prices" }
