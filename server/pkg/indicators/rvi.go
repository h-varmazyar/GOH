package indicators

import (
	"context"
	"math"
	"sync"

	"github.com/shopspring/decimal"
)

// RVI is the Relative Volatility Index. It compares Wilder-smoothed upward
// and downward standard deviations and returns a value between zero and 100.
type RVI struct {
	indicator
	*RVIConfigs
	candles []Candle
}

type RVIConfigs struct {
	BasicConfigs
	SmoothingPeriod int
	Source          IndicatorSource
}

type RVIValue struct{ RVI decimal.Decimal }

func (v *RVIValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{"rvi": v.RVI}
}

func NewRVI(configs *RVIConfigs, providerID, marketID, timeframeID uint) (*RVI, error) {
	ind := &RVI{
		indicator:  indicator{lock: new(sync.Mutex), providerID: providerID, marketID: marketID, timeframeID: timeframeID},
		RVIConfigs: configs,
	}
	return ind, ind.hasValidConfigs()
}

func (ind *RVI) GetType() IndicatorType      { return IndicatorTypeRVI }
func (ind *RVI) GetRequiredCandleCount() int { return ind.Period + ind.SmoothingPeriod }
func (ind *RVI) hasValidConfigs() error {
	if ind.RVIConfigs == nil {
		return ErrMissingIndicatorCfg
	}
	if err := validateBasicConfigs(&ind.BasicConfigs); err != nil {
		return err
	}
	if ind.SmoothingPeriod <= 0 {
		return ErrInvalidSmoothingPeriod
	}
	if !ind.Source.IsValidSource() {
		return ErrInvalidSource
	}
	return nil
}

func (ind *RVI) Calculate(_ context.Context, candles []Candle) []IndicatorValue {
	values := make([]IndicatorValue, 0, len(candles))
	if len(candles) == 0 {
		return values
	}

	up, down := decimal.Zero, decimal.Zero
	smoothingPeriod := decimal.NewFromInt(int64(ind.SmoothingPeriod))
	previousWeight := decimal.NewFromInt(int64(ind.SmoothingPeriod - 1))
	for i, candle := range candles {
		standardDeviation := rollingStandardDeviation(candles, i, ind.Period, ind.Source)
		upInput, downInput := decimal.Zero, decimal.Zero
		if i > 0 {
			change := sourceValue(candle, ind.Source).Sub(sourceValue(candles[i-1], ind.Source))
			if change.GreaterThan(decimal.Zero) {
				upInput = standardDeviation
			} else if change.LessThan(decimal.Zero) {
				downInput = standardDeviation
			}
		}
		if i == 0 {
			up, down = upInput, downInput
		} else {
			up = up.Mul(previousWeight).Add(upInput).Div(smoothingPeriod)
			down = down.Mul(previousWeight).Add(downInput).Div(smoothingPeriod)
		}
		rvi := decimal.NewFromInt(50)
		if denominator := up.Add(down); !denominator.IsZero() {
			rvi = up.Div(denominator).Mul(decimal.NewFromInt(100))
		}
		values = append(values, IndicatorValue{Time: candle.Time, Value: &RVIValue{RVI: rvi}})
	}
	ind.candles = append([]Candle(nil), candles...)
	storeCalculatedValues(&ind.indicator, values, ind.StoredValueCount)
	return values
}

func (ind *RVI) UpdateLast(ctx context.Context, candle Candle) {
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

func rollingStandardDeviation(candles []Candle, end, period int, source IndicatorSource) decimal.Decimal {
	start := end - period + 1
	if start < 0 {
		start = 0
	}
	count := end - start + 1
	mean := decimal.Zero
	for i := start; i <= end; i++ {
		mean = mean.Add(sourceValue(candles[i], source))
	}
	mean = mean.Div(decimal.NewFromInt(int64(count)))
	variance := decimal.Zero
	for i := start; i <= end; i++ {
		difference := sourceValue(candles[i], source).Sub(mean)
		variance = variance.Add(difference.Mul(difference))
	}
	variance = variance.Div(decimal.NewFromInt(int64(count)))
	floatVariance, _ := variance.Float64()
	return decimal.NewFromFloat(math.Sqrt(floatVariance))
}
