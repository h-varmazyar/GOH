package models

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
}

func (TSEDailyClosingPrice) TableName() string { return "tse_daily_closing_prices" }
