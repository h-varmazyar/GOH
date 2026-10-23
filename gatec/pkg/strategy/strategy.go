package strategies

import (
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/pkg/indicators"
	"github.com/shopspring/decimal"
)

type MarketEvent struct {
	Market     models.Market
	Candle     models.Candle
	Indicators map[uint]indicators.IndicatorValue
}

type MarketContext struct {
	Event      MarketEvent
	Close      decimal.Decimal
	PrevClose  decimal.Decimal
	Indicators map[uint]indicators.IndicatorValue
	Memory     *IndicatorMemory
	State      *SymbolState
}

type Strategy interface {
	Name() string
	ExtractFeatures(ctx MarketContext) Features
	DetectRegime(ctx MarketContext, f Features) Regime
	GenerateSignal(ctx MarketContext, f Features, r Regime) Signal
	EvaluateRisk(ctx MarketContext, s Signal) RiskResult
}
