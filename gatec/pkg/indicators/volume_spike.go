package indicators

import (
	"context"
	"sync"

	"github.com/shopspring/decimal"
)

type VolumeSpike struct {
	indicator
	*VolumeSpikeConfigs
	candles []Candle
}

type VolumeSpikeConfigs struct {
	BasicConfigs
	Multiplier decimal.Decimal
}

type VolumeSpikeValue struct {
	Volume        decimal.Decimal
	AverageVolume decimal.Decimal
	Ratio         decimal.Decimal
	IsSpike       decimal.Decimal
}

func (v *VolumeSpikeValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{
		"volume": v.Volume, "average_volume": v.AverageVolume,
		"ratio": v.Ratio, "is_spike": v.IsSpike,
	}
}

func NewVolumeSpike(configs *VolumeSpikeConfigs, providerID, marketID, timeframeID uint) (*VolumeSpike, error) {
	ind := &VolumeSpike{
		indicator:          indicator{lock: new(sync.Mutex), providerID: providerID, marketID: marketID, timeframeID: timeframeID},
		VolumeSpikeConfigs: configs,
	}
	return ind, ind.hasValidConfigs()
}

func (ind *VolumeSpike) GetType() IndicatorType      { return IndicatorTypeVolumeSpike }
func (ind *VolumeSpike) GetRequiredCandleCount() int { return ind.Period + 1 }
func (ind *VolumeSpike) hasValidConfigs() error {
	if ind.VolumeSpikeConfigs == nil {
		return ErrMissingIndicatorCfg
	}
	if err := validateBasicConfigs(&ind.BasicConfigs); err != nil {
		return err
	}
	if !ind.Multiplier.GreaterThan(decimal.Zero) {
		return ErrInvalidMultiplier
	}
	return nil
}

func (ind *VolumeSpike) Calculate(_ context.Context, candles []Candle) []IndicatorValue {
	values := make([]IndicatorValue, 0, len(candles))
	if len(candles) == 0 {
		return values
	}

	previousVolumeSum := decimal.Zero
	for i, candle := range candles {
		windowSize := i
		if windowSize > ind.Period {
			windowSize = ind.Period
		}
		average, ratio, isSpike := decimal.Zero, decimal.Zero, decimal.Zero
		if windowSize > 0 {
			average = previousVolumeSum.Div(decimal.NewFromInt(int64(windowSize)))
			if !average.IsZero() {
				ratio = candle.Volume.Div(average)
			}
			if !average.IsZero() && candle.Volume.GreaterThanOrEqual(average.Mul(ind.Multiplier)) {
				isSpike = decimal.NewFromInt(1)
			}
		}
		values = append(values, IndicatorValue{Time: candle.Time, Value: &VolumeSpikeValue{
			Volume: candle.Volume, AverageVolume: average, Ratio: ratio, IsSpike: isSpike,
		}})
		previousVolumeSum = previousVolumeSum.Add(candle.Volume)
		if i >= ind.Period {
			previousVolumeSum = previousVolumeSum.Sub(candles[i-ind.Period].Volume)
		}
	}
	ind.candles = append([]Candle(nil), candles...)
	storeCalculatedValues(&ind.indicator, values, ind.StoredValueCount)
	return values
}

func (ind *VolumeSpike) UpdateLast(ctx context.Context, candle Candle) {
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
