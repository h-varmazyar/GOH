package indicators

import (
	"context"
	"sync"

	"github.com/shopspring/decimal"
)

type EMA struct {
	indicator
	*EMAConfigs
}

type EMAConfigs struct {
	BasicConfigs
	Source IndicatorSource
}

type EMAValue struct{ EMA decimal.Decimal }

func (v *EMAValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{"ema": v.EMA}
}

func NewEMA(configs *EMAConfigs, providerID, marketID, timeframeID uint) (*EMA, error) {
	ind := &EMA{indicator: indicator{lock: new(sync.Mutex), providerID: providerID, marketID: marketID, timeframeID: timeframeID}, EMAConfigs: configs}
	return ind, ind.hasValidConfigs()
}

func (ind *EMA) GetType() IndicatorType {
	return IndicatorTypeEMA
}

func (ind *EMA) GetRequiredCandleCount() int {
	return 1
}

func (ind *EMA) hasValidConfigs() error {
	return nil
}

func (ind *EMA) Calculate(ctx context.Context, candles []Candle) []IndicatorValue {
	vals := make([]IndicatorValue, 0, len(candles))
	if len(candles) == 0 {
		return vals
	}
	k := decimal.NewFromInt(2).Div(decimal.NewFromInt(int64(ind.Period + 1)))
	ema := candles[0].Close
	for _, c := range candles {
		ema = c.Close.Mul(k).Add(ema.Mul(decimal.NewFromInt(1).Sub(k)))
		v := IndicatorValue{Time: c.Time, Value: &EMAValue{EMA: ema}}
		vals = append(vals, v)
		ind.updateLastValue(ctx, v)
	}
	return vals
}
func (ind *EMA) UpdateLast(ctx context.Context, candle Candle) {
	ind.Calculate(ctx, []Candle{candle})
}
