package indicators

import (
	"context"
	"sync"

	"github.com/shopspring/decimal"
)

type AverageVolume struct {
	indicator
	*AverageVolumeConfigs
	candles []Candle
}

type AverageVolumeConfigs struct{ BasicConfigs }

type AverageVolumeValue struct{ AverageVolume decimal.Decimal }

func (v *AverageVolumeValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{"average_volume": v.AverageVolume}
}

func NewAverageVolume(configs *AverageVolumeConfigs, providerID, marketID, timeframeID uint) (*AverageVolume, error) {
	ind := &AverageVolume{
		indicator:            indicator{lock: new(sync.Mutex), providerID: providerID, marketID: marketID, timeframeID: timeframeID},
		AverageVolumeConfigs: configs,
	}
	return ind, ind.hasValidConfigs()
}

func (ind *AverageVolume) GetType() IndicatorType      { return IndicatorTypeAverageVolume }
func (ind *AverageVolume) GetRequiredCandleCount() int { return ind.Period }
func (ind *AverageVolume) hasValidConfigs() error {
	if ind.AverageVolumeConfigs == nil {
		return ErrMissingIndicatorCfg
	}
	return validateBasicConfigs(&ind.BasicConfigs)
}

func (ind *AverageVolume) Calculate(_ context.Context, candles []Candle) []IndicatorValue {
	values := make([]IndicatorValue, 0, len(candles))
	if len(candles) == 0 {
		return values
	}

	sum := decimal.Zero
	for i, candle := range candles {
		sum = sum.Add(candle.Volume)
		if i >= ind.Period {
			sum = sum.Sub(candles[i-ind.Period].Volume)
		}
		windowSize := i + 1
		if windowSize > ind.Period {
			windowSize = ind.Period
		}
		values = append(values, IndicatorValue{Time: candle.Time, Value: &AverageVolumeValue{
			AverageVolume: sum.Div(decimal.NewFromInt(int64(windowSize))),
		}})
	}
	ind.candles = append([]Candle(nil), candles...)
	storeCalculatedValues(&ind.indicator, values, ind.StoredValueCount)
	return values
}

func (ind *AverageVolume) UpdateLast(ctx context.Context, candle Candle) {
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
