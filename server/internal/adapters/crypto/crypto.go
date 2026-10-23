package crypto

import (
	"context"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/shopspring/decimal"
	"time"
)

type Candle struct {
	Key    string          `json:"key"`
	Time   int64           `json:"time"`
	Open   decimal.Decimal `json:"open"`
	High   decimal.Decimal `json:"high"`
	Low    decimal.Decimal `json:"low"`
	Close  decimal.Decimal `json:"close"`
	Volume decimal.Decimal `json:"volume"`
}

type Market struct {
	Name   string              `json:"name"`
	Status consts.MarketStatus `json:"status"`
}

type Crypto interface {
	GetID() uint
	GetCandles(key, market string, timeframe time.Duration) error
	GetCandleHistory(market string, timeframe time.Duration, to time.Time, count int) ([]Candle, error)
	UpdateMarkets(ctx context.Context) (markets []Market, error error)
}
