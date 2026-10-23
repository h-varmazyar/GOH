package indicators

import (
	"context"
	"github.com/shopspring/decimal"
	"sync"
)

type VWAP struct {
	indicator
	*VWAPConfigs
}
type VWAPConfigs struct{ BasicConfigs }
type VWAPValue struct{ VWAP decimal.Decimal }

func (v *VWAPValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{"value": v.VWAP, "vwap": v.VWAP}
}
func NewVWAP(configs *VWAPConfigs, providerID, marketID, timeframeID uint) (*VWAP, error) {
	return &VWAP{indicator: indicator{lock: new(sync.Mutex), providerID: providerID, marketID: marketID, timeframeID: timeframeID}, VWAPConfigs: configs}, nil
}
func (ind *VWAP) GetType() IndicatorType      { return IndicatorType("VWAP") }
func (ind *VWAP) GetRequiredCandleCount() int { return 1 }
func (ind *VWAP) hasValidConfigs() error      { return nil }
func (ind *VWAP) Calculate(ctx context.Context, candles []Candle) []IndicatorValue {
	out := make([]IndicatorValue, 0, len(candles))
	pv := decimal.Zero
	vol := decimal.Zero
	for _, c := range candles {
		tp := c.High.Add(c.Low).Add(c.Close).Div(decimal.NewFromInt(3))
		pv = pv.Add(tp.Mul(c.Volume))
		vol = vol.Add(c.Volume)
		vwap := decimal.Zero
		if !vol.IsZero() {
			vwap = pv.Div(vol)
		}
		v := IndicatorValue{Time: c.Time, Value: &VWAPValue{VWAP: vwap}}
		out = append(out, v)
		ind.updateLastValue(ctx, v)
	}
	return out
}
func (ind *VWAP) UpdateLast(ctx context.Context, candle Candle) { ind.Calculate(ctx, []Candle{candle}) }
