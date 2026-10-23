package indicators

import (
	"context"
	"sync"

	"github.com/shopspring/decimal"
)

type EMASlope struct {
	indicator
	*EMASlopeConfigs
	candles []Candle
}

type EMASlopeConfigs struct {
	BasicConfigs
	SlopePeriod int
	Source      IndicatorSource
}

type EMASlopeValue struct {
	EMA          decimal.Decimal
	Slope        decimal.Decimal
	SlopePercent decimal.Decimal
}

func (v *EMASlopeValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{"ema": v.EMA, "slope": v.Slope, "slope_percent": v.SlopePercent}
}

func NewEMASlope(configs *EMASlopeConfigs, providerID, marketID, timeframeID uint) (*EMASlope, error) {
	ind := &EMASlope{
		indicator:       indicator{lock: new(sync.Mutex), providerID: providerID, marketID: marketID, timeframeID: timeframeID},
		EMASlopeConfigs: configs,
	}
	return ind, ind.hasValidConfigs()
}

func (ind *EMASlope) GetType() IndicatorType      { return IndicatorTypeEMASlope }
func (ind *EMASlope) GetRequiredCandleCount() int { return ind.Period + ind.SlopePeriod }
func (ind *EMASlope) hasValidConfigs() error {
	if ind.EMASlopeConfigs == nil {
		return ErrMissingIndicatorCfg
	}
	if err := validateBasicConfigs(&ind.BasicConfigs); err != nil {
		return err
	}
	if ind.SlopePeriod <= 0 {
		return ErrInvalidSlopePeriod
	}
	if !ind.Source.IsValidSource() {
		return ErrInvalidSource
	}
	return nil
}

func (ind *EMASlope) Calculate(_ context.Context, candles []Candle) []IndicatorValue {
	values := make([]IndicatorValue, 0, len(candles))
	emas := emaSeries(candles, ind.Period, ind.Source)
	slopeDivisor := decimal.NewFromInt(int64(ind.SlopePeriod))
	for i, candle := range candles {
		slope, slopePercent := decimal.Zero, decimal.Zero
		if i >= ind.SlopePeriod {
			base := emas[i-ind.SlopePeriod]
			slope = emas[i].Sub(base).Div(slopeDivisor)
			if !base.IsZero() {
				slopePercent = slope.Div(base).Mul(decimal.NewFromInt(100))
			}
		}
		values = append(values, IndicatorValue{Time: candle.Time, Value: &EMASlopeValue{
			EMA: emas[i], Slope: slope, SlopePercent: slopePercent,
		}})
	}
	ind.candles = append([]Candle(nil), candles...)
	storeCalculatedValues(&ind.indicator, values, ind.StoredValueCount)
	return values
}

func (ind *EMASlope) UpdateLast(ctx context.Context, candle Candle) {
	if len(ind.candles) == 0 {
		ind.Calculate(ctx, []Candle{candle})
		return
	}
	last := len(ind.candles) - 1
	if candle.Time == ind.candles[last].Time {
		ind.candles[last] = candle
	} else if candle.Time > ind.candles[last].Time {
		ind.candles = append(ind.candles, candle)
	} else {
		return
	}
	ind.Calculate(ctx, ind.candles)
}
