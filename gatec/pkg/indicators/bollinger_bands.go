package indicators

import (
	"context"
	"errors"
	"math"
	"sync"

	"github.com/shopspring/decimal"
)

type BollingerBands struct {
	indicator
	*BollingerBandsConfigs
	periodCandles []Candle
}

type BollingerBandsConfigs struct {
	BasicConfigs
	Multiplier int64
	Source     IndicatorSource
}

type BollingerBandsValue struct {
	UpperBand  decimal.Decimal
	LowerBand  decimal.Decimal
	MiddleBand decimal.Decimal
	PercentB   decimal.Decimal
	Bandwidth  decimal.Decimal
}

func (v *BollingerBandsValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{"upper_band": v.UpperBand, "lower_band": v.LowerBand, "middle_band": v.MiddleBand, "percent_b": v.PercentB, "bandwidth": v.Bandwidth}
}
func (*BollingerBandsValue) isIndicatorValue() {}

func NewBollingerBands(configs *BollingerBandsConfigs, providerID, marketID, timeframeID uint) (*BollingerBands, error) {
	if configs == nil || configs.Period < 1 || configs.Multiplier < 0 {
		return nil, errors.New("Bollinger period must be positive and multiplier cannot be negative")
	}
	return &BollingerBands{
		indicator: indicator{
			lock:        new(sync.Mutex),
			providerID:  providerID,
			marketID:    marketID,
			timeframeID: timeframeID,
		},
		BollingerBandsConfigs: configs,
	}, nil
}

func (conf *BollingerBands) GetRequiredCandleCount() int {
	return conf.Period
}

func (ind *BollingerBands) GetType() IndicatorType {
	return IndicatorTypeBollingerBands
}

func (ind *BollingerBands) Calculate(ctx context.Context, candles []Candle) []IndicatorValue {
	values := make([]IndicatorValue, len(candles))

	for i := range candles {
		values[i] = IndicatorValue{
			Time: candles[i].Time,
		}
		if ind.Period > 0 && i+1 >= ind.Period {
			values[i].Value = ind.calculateBands(candles[i+1-ind.Period : i+1])
		}
	}

	ind.periodCandles = append([]Candle(nil), candles...)
	if len(ind.periodCandles) > ind.Period {
		ind.periodCandles = ind.periodCandles[len(ind.periodCandles)-ind.Period:]
	}

	for _, v := range values {
		ind.updateLastValue(ctx, v)
	}

	return values
}

func (ind *BollingerBands) UpdateLast(ctx context.Context, candle Candle) {
	if len(ind.periodCandles) == 0 || candle.Time > ind.periodCandles[len(ind.periodCandles)-1].Time {
		ind.periodCandles = append(ind.periodCandles, candle)
	} else if candle.Time == ind.periodCandles[len(ind.periodCandles)-1].Time {
		ind.periodCandles[len(ind.periodCandles)-1] = candle
	} else {
		return
	}
	if len(ind.periodCandles) > ind.Period {
		ind.periodCandles = ind.periodCandles[len(ind.periodCandles)-ind.Period:]
	}
	if len(ind.periodCandles) < ind.Period {
		return
	}

	value := ind.calculateBands(ind.periodCandles)

	lastValue := IndicatorValue{
		Time:  candle.Time,
		Value: value,
	}

	ind.updateLastValue(ctx, lastValue)
}

func (ind *BollingerBands) hasValidConfigs() error {
	return nil
}

func (ind *BollingerBands) calculateBands(candles []Candle) *BollingerBandsValue {
	if len(candles) == 0 {
		return &BollingerBandsValue{}
	}
	mean := decimal.Zero
	prices := make([]decimal.Decimal, len(candles))
	for i := range candles {
		prices[i] = bollingerSource(candles[i], ind.Source)
		mean = mean.Add(prices[i])
	}
	mean = mean.Div(decimal.NewFromInt(int64(len(candles))))
	sumOfSquares := decimal.Zero
	for _, price := range prices {
		ps1 := price.Sub(mean)
		square := ps1.Mul(ps1)
		sumOfSquares = sumOfSquares.Add(square)
	}
	variance := sumOfSquares.Div(decimal.NewFromInt(int64(len(candles))))

	floatVariance, _ := variance.Float64()
	standardDeviation := math.Sqrt(floatVariance)
	value := decimal.NewFromFloat(float64(ind.Multiplier) * standardDeviation)
	upper := mean.Add(value)
	lower := mean.Sub(value)
	percentB, bandwidth := decimal.Zero, decimal.Zero
	width := upper.Sub(lower)
	if !width.IsZero() {
		percentB = candles[len(candles)-1].Close.Sub(lower).Div(width)
	}
	if !mean.IsZero() {
		bandwidth = width.Div(mean.Abs()).Mul(decimal.NewFromInt(100))
	}
	return &BollingerBandsValue{
		UpperBand:  upper,
		LowerBand:  lower,
		MiddleBand: mean,
		PercentB:   percentB,
		Bandwidth:  bandwidth,
	}
}

func bollingerSource(c Candle, source IndicatorSource) decimal.Decimal {
	switch source {
	case IndicatorSourceOpen:
		return c.Open
	case IndicatorSourceHigh:
		return c.High
	case IndicatorSourceLow:
		return c.Low
	case IndicatorSourceOHLC4:
		return c.Open.Add(c.High).Add(c.Low).Add(c.Close).Div(decimal.NewFromInt(4))
	case IndicatorSourceHLC3:
		return c.Low.Add(c.High).Add(c.Close).Div(decimal.NewFromInt(3))
	case IndicatorSourceHL2:
		return c.Low.Add(c.High).Div(decimal.NewFromInt(2))
	default:
		return c.Close
	}
}
