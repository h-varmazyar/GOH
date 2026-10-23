package indicators

import (
	"context"
	"fmt"
	"sync"

	"github.com/shopspring/decimal"
)

type ADX struct {
	indicator
	*ADXConfigs
}
type ADXConfigs struct{ BasicConfigs }
type ADXValue struct{ ADX decimal.Decimal }

func (v *ADXValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{"adx": v.ADX}
}
func NewADX(configs *ADXConfigs, providerID, marketID, timeframeID uint) (*ADX, error) {
	return &ADX{indicator: indicator{lock: new(sync.Mutex), providerID: providerID, marketID: marketID, timeframeID: timeframeID}, ADXConfigs: configs}, nil
}
func (ind *ADX) GetType() IndicatorType      { return IndicatorType("ADX") }
func (ind *ADX) GetRequiredCandleCount() int { return 2 }
func (ind *ADX) hasValidConfigs() error      { return nil }
func (ind *ADX) Calculate(ctx context.Context, candles []Candle) []IndicatorValue {
	out := make([]IndicatorValue, 0, len(candles))
	for _, c := range candles {
		rng := c.High.Sub(c.Low)
		adx := decimal.Zero
		if !c.Close.IsZero() {
			adx = rng.Div(c.Close).Mul(decimal.NewFromInt(100))
		}
		v := IndicatorValue{Time: c.Time, Value: &ADXValue{ADX: adx}}
		out = append(out, v)
		ind.updateLastValue(ctx, v)

		fmt.Println(c, v.Time, v.Value)
	}
	return out
}
func (ind *ADX) UpdateLast(ctx context.Context, candle Candle) { ind.Calculate(ctx, []Candle{candle}) }
