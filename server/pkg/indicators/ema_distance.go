package indicators

import (
	"context"
	"sync"

	"github.com/shopspring/decimal"
)

type EMADistance struct {
	indicator
	*EMADistanceConfigs
	candles []Candle
}

type EMADistanceConfigs struct {
	BasicConfigs
	Source IndicatorSource
}

type EMADistanceValue struct {
	EMA             decimal.Decimal
	Distance        decimal.Decimal
	DistancePercent decimal.Decimal
}

func (v *EMADistanceValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{"ema": v.EMA, "distance": v.Distance, "distance_percent": v.DistancePercent}
}

func NewEMADistance(configs *EMADistanceConfigs, providerID, marketID, timeframeID uint) (*EMADistance, error) {
	ind := &EMADistance{
		indicator:          indicator{lock: new(sync.Mutex), providerID: providerID, marketID: marketID, timeframeID: timeframeID},
		EMADistanceConfigs: configs,
	}
	return ind, ind.hasValidConfigs()
}

func (ind *EMADistance) GetType() IndicatorType      { return IndicatorTypeEMADistance }
func (ind *EMADistance) GetRequiredCandleCount() int { return ind.Period }
func (ind *EMADistance) hasValidConfigs() error {
	if ind.EMADistanceConfigs == nil {
		return ErrMissingIndicatorCfg
	}
	if err := validateBasicConfigs(&ind.BasicConfigs); err != nil {
		return err
	}
	if !ind.Source.IsValidSource() {
		return ErrInvalidSource
	}
	return nil
}

func (ind *EMADistance) Calculate(_ context.Context, candles []Candle) []IndicatorValue {
	values := make([]IndicatorValue, 0, len(candles))
	emas := emaSeries(candles, ind.Period, ind.Source)
	for i, candle := range candles {
		price := sourceValue(candle, ind.Source)
		distance := price.Sub(emas[i])
		distancePercent := decimal.Zero
		if !emas[i].IsZero() {
			distancePercent = distance.Div(emas[i]).Mul(decimal.NewFromInt(100))
		}
		values = append(values, IndicatorValue{Time: candle.Time, Value: &EMADistanceValue{
			EMA: emas[i], Distance: distance, DistancePercent: distancePercent,
		}})
	}
	ind.candles = append([]Candle(nil), candles...)
	storeCalculatedValues(&ind.indicator, values, ind.StoredValueCount)
	return values
}

func (ind *EMADistance) UpdateLast(ctx context.Context, candle Candle) {
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
