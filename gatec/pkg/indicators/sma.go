package indicators

import (
	"context"
	"sync"

	"github.com/shopspring/decimal"
)

type SMA struct {
	indicator
	*SMAConfigs
	periodCandles []Candle
}

type SMAConfigs struct {
	BasicConfigs
	Source IndicatorSource
}

type SMAValue struct {
	SMA decimal.Decimal
}

func (v *SMAValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{"sma": v.SMA}
}
func (v *SMAValue) isIndicatorValue() {}

func NewSMA(configs *SMAConfigs, providerID, marketID, timeframeID uint) (*SMA, error) {
	return &SMA{
		indicator: indicator{
			lock:        new(sync.Mutex),
			providerID:  providerID,
			marketID:    marketID,
			timeframeID: timeframeID,
		},
		SMAConfigs: configs,
	}, nil
}

func (ind *SMA) GetRequiredCandleCount() int {
	return 1
}

func (ind *SMA) GetType() IndicatorType {
	return IndicatorTypeSMA
}

func (ind *SMA) Calculate(ctx context.Context, candles []Candle) []IndicatorValue {
	values := make([]IndicatorValue, len(candles))

	baseNum := ind.getSmaNumber(candles[0])
	values[0] = IndicatorValue{
		Value: &SMAValue{SMA: baseNum},
		Time:  candles[0].Time,
	}

	for i := 1; i < len(candles); i++ {

		newNum := ind.getSmaNumber(candles[i])
		var changeableNum decimal.Decimal
		if i >= int(ind.Period) {
			changeableNum = ind.getSmaNumber(candles[i-int(ind.Period)])
		}

		baseNum = baseNum.Add(newNum).Sub(changeableNum)
		values[i] = IndicatorValue{
			Value: &SMAValue{SMA: baseNum},
			Time:  candles[i].Time,
		}

	}

	ind.periodCandles = candles[len(candles)-ind.Period:]

	for _, v := range values {
		ind.updateLastValue(ctx, v)
	}

	return values
}

func (ind *SMA) UpdateLast(ctx context.Context, candle Candle) {
	var changeableCandle Candle
	lastValue := ind.GetValues(ctx, 1)[0]
	if lastValue.Time == candle.Time {
		changeableCandle = ind.periodCandles[ind.Period-1]
	} else if lastValue.Time < candle.Time {
		changeableCandle = ind.periodCandles[0]
		ind.periodCandles = ind.periodCandles[1:] //todo: must be check
	}
	ind.periodCandles[ind.Period-1] = candle

	newNum := ind.getSmaNumber(candle)
	changeableNum := ind.getSmaNumber(changeableCandle)

	newValue := (lastValue.Value).(*SMAValue).SMA.Add(newNum).Sub(changeableNum)

	lastValue.Time = candle.Time
	lastValue.Value = &SMAValue{SMA: newValue}

	ind.updateLastValue(ctx, lastValue)
}

func (ind *SMA) hasValidConfigs() error {
	return nil
}

func (ind *SMA) getSmaNumber(candle Candle) decimal.Decimal {
	calculatorPeriod := ind.Period
	switch ind.Source {
	case IndicatorSourceClose:
		return candle.Close.Div(decimal.NewFromInt(int64(ind.Period)))
	case IndicatorSourceOpen:
		return candle.Open.Div(decimal.NewFromInt(int64(ind.Period)))
	case IndicatorSourceHigh:
		return candle.High.Div(decimal.NewFromInt(int64(ind.Period)))
	case IndicatorSourceLow:
		return candle.Low.Div(decimal.NewFromInt(int64(ind.Period)))
	case IndicatorSourceHL2:
		return candle.High.Add(candle.Low).Div(decimal.NewFromInt(int64(2 * calculatorPeriod)))
	case IndicatorSourceHLC3:
		return (candle.High.Add(candle.Low).Add(candle.Close)).Div(decimal.NewFromInt(int64(3 * calculatorPeriod)))
	case IndicatorSourceOHLC4:
		return (candle.Open.Add(candle.Close).Add(candle.High).Add(candle.Low)).Div(decimal.NewFromInt(int64(4 * calculatorPeriod)))
	}

	return decimal.Zero
}
