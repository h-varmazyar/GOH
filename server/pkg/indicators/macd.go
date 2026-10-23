package indicators

import (
	"context"
	"github.com/shopspring/decimal"
	"sync"
)

type MACD struct {
	indicator
	*MACDConfigs
}
type MACDConfigs struct {
	BasicConfigs
	FastPeriod   int
	SlowPeriod   int
	SignalPeriod int
}
type MACDValue struct{ MACD, Signal, Histogram decimal.Decimal }

func (v *MACDValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{"macd": v.MACD, "signal": v.Signal, "histogram": v.Histogram}
}
func NewMACD(configs *MACDConfigs, providerID, marketID, timeframeID uint) (*MACD, error) {
	return &MACD{indicator: indicator{lock: new(sync.Mutex), providerID: providerID, marketID: marketID, timeframeID: timeframeID}, MACDConfigs: configs}, nil
}
func (ind *MACD) GetType() IndicatorType      { return IndicatorType("MACD") }
func (ind *MACD) GetRequiredCandleCount() int { return 1 }
func (ind *MACD) hasValidConfigs() error      { return nil }
func (ind *MACD) Calculate(ctx context.Context, candles []Candle) []IndicatorValue {
	out := make([]IndicatorValue, 0, len(candles))
	if len(candles) == 0 {
		return out
	}
	fast := candles[0].Close
	slow := candles[0].Close
	sig := decimal.Zero
	kf := decimal.NewFromInt(2).Div(decimal.NewFromInt(int64(ind.FastPeriod + 1)))
	ks := decimal.NewFromInt(2).Div(decimal.NewFromInt(int64(ind.SlowPeriod + 1)))
	ksi := decimal.NewFromInt(2).Div(decimal.NewFromInt(int64(ind.SignalPeriod + 1)))
	for _, c := range candles {
		fast = c.Close.Mul(kf).Add(fast.Mul(decimal.NewFromInt(1).Sub(kf)))
		slow = c.Close.Mul(ks).Add(slow.Mul(decimal.NewFromInt(1).Sub(ks)))
		m := fast.Sub(slow)
		sig = m.Mul(ksi).Add(sig.Mul(decimal.NewFromInt(1).Sub(ksi)))
		h := m.Sub(sig)
		v := IndicatorValue{Time: c.Time, Value: &MACDValue{MACD: m, Signal: sig, Histogram: h}}
		out = append(out, v)
		ind.updateLastValue(ctx, v)
	}
	return out
}
func (ind *MACD) UpdateLast(ctx context.Context, candle Candle) { ind.Calculate(ctx, []Candle{candle}) }
