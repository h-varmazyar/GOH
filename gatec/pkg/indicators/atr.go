package indicators

import (
	"context"
	"github.com/shopspring/decimal"
	"sync"
)

type ATR struct {
	indicator
	*ATRConfigs
}
type ATRConfigs struct{ BasicConfigs }
type ATRValue struct{ ATR decimal.Decimal }

func (v *ATRValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{"atr": v.ATR}
}
func NewATR(configs *ATRConfigs, providerID, marketID, timeframeID uint) (*ATR, error) {
	return &ATR{indicator: indicator{lock: new(sync.Mutex), providerID: providerID, marketID: marketID, timeframeID: timeframeID}, ATRConfigs: configs}, nil
}
func (ind *ATR) GetType() IndicatorType      { return IndicatorType("ATR") }
func (ind *ATR) GetRequiredCandleCount() int { return 2 }
func (ind *ATR) hasValidConfigs() error      { return nil }
func (ind *ATR) Calculate(ctx context.Context, candles []Candle) []IndicatorValue {
	out := make([]IndicatorValue, 0, len(candles))
	if len(candles) == 0 {
		return out
	}
	prev := candles[0].Close
	atr := decimal.Zero
	p := decimal.NewFromInt(int64(max(1, ind.Period)))
	for _, c := range candles {
		tr := c.High.Sub(c.Low)
		a := c.High.Sub(prev).Abs()
		b := c.Low.Sub(prev).Abs()
		if a.GreaterThan(tr) {
			tr = a
		}
		if b.GreaterThan(tr) {
			tr = b
		}
		atr = atr.Mul(p.Sub(decimal.NewFromInt(1))).Add(tr).Div(p)
		v := IndicatorValue{Time: c.Time, Value: &ATRValue{ATR: atr}}
		out = append(out, v)
		ind.updateLastValue(ctx, v)
		prev = c.Close
	}
	return out
}
func (ind *ATR) UpdateLast(ctx context.Context, candle Candle) { ind.Calculate(ctx, []Candle{candle}) }

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
