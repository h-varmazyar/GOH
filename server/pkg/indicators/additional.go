package indicators

import (
	"context"
	"errors"
	"sync"

	"github.com/shopspring/decimal"
)

// WindowIndicatorConfig configures indicators that need a finite candle window.
type WindowIndicatorConfig struct {
	BasicConfigs
}

type windowIndicator struct {
	indicator
	cfg       WindowIndicatorConfig
	kind      string
	candles   []Candle
	atrState  atrPercentState
	atrBefore atrPercentState
}

type atrPercentState struct {
	prevClose   decimal.Decimal
	seedSum     decimal.Decimal
	atr         decimal.Decimal
	trCount     int
	initialized bool
	ready       bool
}

type ATRPercent struct{ *windowIndicator }
type Donchian struct{ *windowIndicator }
type ROC struct{ *windowIndicator }
type MFI struct{ *windowIndicator }
type CandleFeatures struct{ *windowIndicator }

type ATRPercentValue struct{ Value decimal.Decimal }

func (v *ATRPercentValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{"value": v.Value}
}

type DonchianValue struct{ Upper, Lower, Mid decimal.Decimal }

func (v *DonchianValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{"upper": v.Upper, "lower": v.Lower, "mid": v.Mid}
}

type ROCValue struct{ Value decimal.Decimal }

func (v *ROCValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{"value": v.Value}
}

type MFIValue struct{ Value decimal.Decimal }

func (v *MFIValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{"value": v.Value}
}

type CandleFeaturesValue struct{ BodyPct, UpperWickPct, LowerWickPct, RangePct, CloseLocation decimal.Decimal }

func (v *CandleFeaturesValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{"body_pct": v.BodyPct, "upper_wick_pct": v.UpperWickPct, "lower_wick_pct": v.LowerWickPct, "range_pct": v.RangePct, "close_location": v.CloseLocation}
}

func NewATRPercent(cfg *WindowIndicatorConfig, providerID, marketID, timeframeID uint) (*ATRPercent, error) {
	if cfg == nil || cfg.Period < 1 {
		return nil, errors.New("ATR percent period must be positive")
	}
	return &ATRPercent{newWindowIndicator("atr_pct", cfg, providerID, marketID, timeframeID)}, nil
}
func NewDonchian(cfg *WindowIndicatorConfig, providerID, marketID, timeframeID uint) (*Donchian, error) {
	if cfg == nil || cfg.Period < 1 {
		return nil, errors.New("Donchian period must be positive")
	}
	return &Donchian{newWindowIndicator("donchian", cfg, providerID, marketID, timeframeID)}, nil
}
func NewROC(cfg *WindowIndicatorConfig, providerID, marketID, timeframeID uint) (*ROC, error) {
	if cfg == nil || cfg.Period < 1 {
		return nil, errors.New("ROC period must be positive")
	}
	return &ROC{newWindowIndicator("roc", cfg, providerID, marketID, timeframeID)}, nil
}
func NewMFI(cfg *WindowIndicatorConfig, providerID, marketID, timeframeID uint) (*MFI, error) {
	if cfg == nil || cfg.Period < 1 {
		return nil, errors.New("MFI period must be positive")
	}
	return &MFI{newWindowIndicator("mfi", cfg, providerID, marketID, timeframeID)}, nil
}
func NewCandleFeatures(cfg *WindowIndicatorConfig, providerID, marketID, timeframeID uint) (*CandleFeatures, error) {
	if cfg == nil {
		return nil, errors.New("candle features config is required")
	}
	return &CandleFeatures{newWindowIndicator("candle_features", cfg, providerID, marketID, timeframeID)}, nil
}

func newWindowIndicator(kind string, cfg *WindowIndicatorConfig, providerID, marketID, timeframeID uint) *windowIndicator {
	return &windowIndicator{indicator: indicator{lock: new(sync.Mutex), providerID: providerID, marketID: marketID, timeframeID: timeframeID}, cfg: *cfg, kind: kind}
}
func (ind *windowIndicator) GetType() IndicatorType { return IndicatorType(ind.kind) }
func (ind *windowIndicator) GetRequiredCandleCount() int {
	switch ind.kind {
	case "atr_pct", "mfi", "roc":
		return ind.cfg.Period + 1
	case "donchian":
		return ind.cfg.Period
	default:
		return 1
	}
}
func (ind *windowIndicator) hasValidConfigs() error { return nil }
func (ind *windowIndicator) Calculate(ctx context.Context, candles []Candle) []IndicatorValue {
	out := make([]IndicatorValue, len(candles))
	if ind.kind == "atr_pct" {
		state := atrPercentState{}
		before := atrPercentState{}
		for i, c := range candles {
			before = state
			state = advanceATRPercent(state, c, ind.cfg.Period)
			out[i] = IndicatorValue{Time: c.Time}
			if state.ready && !c.Close.IsZero() {
				out[i].Value = &ATRPercentValue{Value: state.atr.Div(c.Close.Abs()).Mul(decimal.NewFromInt(100))}
			}
		}
		ind.lock.Lock()
		ind.candles = append([]Candle(nil), candles...)
		ind.StoredValue = append([]IndicatorValue(nil), out...)
		ind.atrState, ind.atrBefore = state, before
		ind.lock.Unlock()
		return out
	}
	for i, c := range candles {
		out[i] = IndicatorValue{Time: c.Time}
		if v, ok := ind.value(candles[:i+1]); ok {
			out[i].Value = v
		}
	}
	ind.lock.Lock()
	ind.candles = append([]Candle(nil), candles...)
	ind.StoredValue = append([]IndicatorValue(nil), out...)
	ind.lock.Unlock()
	return out
}
func (ind *windowIndicator) UpdateLast(_ context.Context, c Candle) {
	ind.lock.Lock()
	defer ind.lock.Unlock()
	if ind.kind == "atr_pct" {
		if n := len(ind.candles); n > 0 && c.Time < ind.candles[n-1].Time {
			return
		} else if n > 0 && c.Time == ind.candles[n-1].Time {
			ind.atrState = ind.atrBefore
			ind.candles[n-1] = c
		} else {
			ind.atrBefore = ind.atrState
			ind.candles = append(ind.candles, c)
		}
		if len(ind.candles) > 2 {
			ind.candles = append([]Candle(nil), ind.candles[len(ind.candles)-2:]...)
		}
		ind.atrState = advanceATRPercent(ind.atrState, c, ind.cfg.Period)
		var value IsIndicatorValue
		if ind.atrState.ready && !c.Close.IsZero() {
			value = &ATRPercentValue{Value: ind.atrState.atr.Div(c.Close.Abs()).Mul(decimal.NewFromInt(100))}
		}
		ind.storeWindowValue(IndicatorValue{Time: c.Time, Value: value})
		return
	}
	if n := len(ind.candles); n > 0 && c.Time < ind.candles[n-1].Time {
		return
	} else if n > 0 && ind.candles[n-1].Time == c.Time {
		ind.candles[n-1] = c
	} else {
		ind.candles = append(ind.candles, c)
	}
	keep := ind.GetRequiredCandleCount() + 1
	if keep < 2 {
		keep = 2
	}
	if len(ind.candles) > keep {
		ind.candles = append([]Candle(nil), ind.candles[len(ind.candles)-keep:]...)
	}
	var v IsIndicatorValue
	if value, ok := ind.value(ind.candles); ok {
		v = value
	}
	entry := IndicatorValue{Time: c.Time, Value: v}
	ind.storeWindowValue(entry)
}
func (ind *windowIndicator) value(cs []Candle) (IsIndicatorValue, bool) {
	p := ind.cfg.Period
	if p < 1 {
		p = 1
	}
	last := cs[len(cs)-1]
	switch ind.kind {
	case "donchian":
		if len(cs) < p {
			return nil, false
		}
		start := len(cs) - p
		upper, lower := cs[start].High, cs[start].Low
		for _, c := range cs[start+1:] {
			if c.High.GreaterThan(upper) {
				upper = c.High
			}
			if c.Low.LessThan(lower) {
				lower = c.Low
			}
		}
		return &DonchianValue{Upper: upper, Lower: lower, Mid: upper.Add(lower).Div(decimal.NewFromInt(2))}, true
	case "roc":
		if len(cs) <= p {
			return nil, false
		}
		prior := cs[len(cs)-1-p].Close
		if prior.IsZero() {
			return nil, false
		}
		return &ROCValue{Value: last.Close.Sub(prior).Div(prior).Mul(decimal.NewFromInt(100))}, true
	case "mfi":
		if len(cs) <= p {
			return nil, false
		}
		positive, negative := decimal.Zero, decimal.Zero
		for i := len(cs) - p; i < len(cs); i++ {
			prevTP := typicalPrice(cs[i-1])
			tp := typicalPrice(cs[i])
			flow := tp.Mul(cs[i].Volume)
			if tp.GreaterThan(prevTP) {
				positive = positive.Add(flow)
			} else if tp.LessThan(prevTP) {
				negative = negative.Add(flow)
			}
		}
		if positive.IsZero() && negative.IsZero() {
			return &MFIValue{Value: decimal.Zero}, true
		}
		if negative.IsZero() {
			return &MFIValue{Value: decimal.NewFromInt(100)}, true
		}
		if positive.IsZero() {
			return &MFIValue{Value: decimal.Zero}, true
		}
		ratio := positive.Div(negative)
		return &MFIValue{Value: decimal.NewFromInt(100).Sub(decimal.NewFromInt(100).Div(decimal.NewFromInt(1).Add(ratio)))}, true
	case "candle_features":
		rng := last.High.Sub(last.Low).Abs()
		if rng.IsZero() {
			return &CandleFeaturesValue{CloseLocation: decimal.NewFromFloat(.5)}, true
		}
		body := last.Close.Sub(last.Open).Abs()
		upper := last.High.Sub(maxDecimal2(last.Open, last.Close))
		lower := minDecimal(last.Open, last.Close).Sub(last.Low)
		loc := last.Close.Sub(last.Low).Div(rng)
		rangePct := decimal.Zero
		if !last.Close.IsZero() {
			rangePct = rng.Div(last.Close.Abs()).Mul(decimal.NewFromInt(100))
		}
		hundred := decimal.NewFromInt(100)
		return &CandleFeaturesValue{BodyPct: body.Div(rng).Mul(hundred), UpperWickPct: upper.Div(rng).Mul(hundred), LowerWickPct: lower.Div(rng).Mul(hundred), RangePct: rangePct, CloseLocation: loc}, true
	}
	return nil, false
}

func advanceATRPercent(state atrPercentState, candle Candle, period int) atrPercentState {
	if period < 1 {
		period = 1
	}
	if !state.initialized {
		state.prevClose = candle.Close
		state.initialized = true
		return state
	}
	tr := maxDecimal(candle.High.Sub(candle.Low), candle.High.Sub(state.prevClose).Abs(), candle.Low.Sub(state.prevClose).Abs())
	state.trCount++
	if state.trCount < period {
		state.seedSum = state.seedSum.Add(tr)
	} else if state.trCount == period {
		state.seedSum = state.seedSum.Add(tr)
		state.atr = state.seedSum.Div(decimal.NewFromInt(int64(period)))
		state.ready = true
	} else {
		p := decimal.NewFromInt(int64(period))
		state.atr = state.atr.Mul(p.Sub(decimal.NewFromInt(1))).Add(tr).Div(p)
	}
	state.prevClose = candle.Close
	return state
}

func (ind *windowIndicator) storeWindowValue(value IndicatorValue) {
	if n := len(ind.StoredValue); n > 0 && ind.StoredValue[n-1].Time == value.Time {
		ind.StoredValue[n-1] = value
	} else {
		ind.StoredValue = append(ind.StoredValue, value)
	}
	if n := ind.cfg.StoredValueCount; n > 0 && len(ind.StoredValue) > n {
		ind.StoredValue = ind.StoredValue[len(ind.StoredValue)-n:]
	}
}
func typicalPrice(c Candle) decimal.Decimal {
	return c.High.Add(c.Low).Add(c.Close).Div(decimal.NewFromInt(3))
}
func maxDecimal(a, b, c decimal.Decimal) decimal.Decimal {
	if b.GreaterThan(a) {
		a = b
	}
	if c.GreaterThan(a) {
		a = c
	}
	return a
}
func maxDecimal2(a, b decimal.Decimal) decimal.Decimal {
	if b.GreaterThan(a) {
		return b
	}
	return a
}
func minDecimal(a, b decimal.Decimal) decimal.Decimal {
	if b.LessThan(a) {
		return b
	}
	return a
}
func (ind *ATRPercent) GetType() IndicatorType     { return IndicatorType("ATR_PCT") }
func (ind *Donchian) GetType() IndicatorType       { return IndicatorType("DONCHIAN") }
func (ind *ROC) GetType() IndicatorType            { return IndicatorType("ROC") }
func (ind *MFI) GetType() IndicatorType            { return IndicatorType("MFI") }
func (ind *CandleFeatures) GetType() IndicatorType { return IndicatorType("CANDLE_FEATURES") }
