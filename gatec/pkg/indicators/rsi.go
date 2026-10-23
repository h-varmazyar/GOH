package indicators

import (
	"context"
	"sync"

	"github.com/shopspring/decimal"
)

type RSI struct {
	indicator
	*RSIConfig
	lastCandle Candle
}

type RSIConfig struct {
	BasicConfigs
	Source IndicatorSource
}

type RSIValue struct {
	gain decimal.Decimal
	loss decimal.Decimal
	RSI  decimal.Decimal
}

func (r RSIValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{
		"rsi": r.RSI,
	}
}

func NewRSI(configs *RSIConfig, providerID, marketID, timeframeID uint) (*RSI, error) {
	ind := &RSI{
		indicator: indicator{
			lock:        new(sync.Mutex),
			providerID:  providerID,
			marketID:    marketID,
			timeframeID: timeframeID,
		},
		RSIConfig: configs,
	}

	if err := ind.hasValidConfigs(); err != nil {
		return nil, err
	}
	return ind, nil
}

func (ind *RSI) GetType() IndicatorType {
	return IndicatorTypeRSI
}

func (ind *RSI) GetRequiredCandleCount() int {
	return 1
}

func (ind *RSI) hasValidConfigs() error {
	if ind.RSIConfig == nil {
		return ErrMissingIndicatorCfg
	}

	if ind.RSIConfig.Period == 0 {
		return ErrInvalidPeriod
	}

	if !ind.RSIConfig.Source.IsValidSource() {
		return ErrInvalidSource
	}

	if ind.RSIConfig.StoredValueCount < 1 {
		return ErrInvalidValueStorageSize
	}

	return nil
}

func (ind *RSI) Calculate(ctx context.Context, candles []Candle) []IndicatorValue {
	values := make([]IndicatorValue, len(candles))

	//first RSI
	{
		gainSum := decimal.Zero
		lossSum := decimal.Zero
		for i := 1; i <= ind.Period; i++ {
			if change := candles[i].Close.Sub(candles[i-1].Close); change.GreaterThan(decimal.Zero) {
				gainSum = gainSum.Add(change)
			} else {
				lossSum = lossSum.Sub(change)
			}
		}

		gain := gainSum.Div(decimal.NewFromInt(int64(ind.Period)))
		loss := lossSum.Div(decimal.NewFromInt(int64(ind.Period)))
		rs := gain.Div(loss)
		rsiValue := decimal.NewFromInt(100).Sub(decimal.NewFromInt(100).Div(decimal.NewFromInt(1).Add(rs)))

		v := IndicatorValue{
			Time: candles[ind.Period].Time,
			Value: &RSIValue{
				gain: gain,
				loss: loss,
				RSI:  rsiValue,
			},
		}
		ind.updateLastValue(ctx, v)
		values[ind.Period] = v

	}

	for i := ind.Period + 1; i < len(candles); i++ {
		ind.lastCandle = candles[i-1]

		v := IndicatorValue{
			Time:  candles[i].Time,
			Value: ind.calculateRSIValue(ctx, candles[i]),
		}
		ind.updateLastValue(ctx, v)
		values[i] = v
	}

	return values
}

func (ind *RSI) UpdateLast(ctx context.Context, candle Candle) {
	value := ind.calculateRSIValue(ctx, candle)

	ind.lastCandle = candle

	lastValue := &IndicatorValue{
		Time:  candle.Time,
		Value: value,
	}

	ind.updateLastValue(ctx, *lastValue)
}

func (ind *RSI) calculateRSIValue(ctx context.Context, candle Candle) *RSIValue {
	gain, loss := decimal.Zero, decimal.Zero
	if change := candle.Close.Sub(ind.lastCandle.Close); change.GreaterThan(decimal.Zero) {
		gain = change
	} else {
		loss = change
	}

	lastValue := ind.GetValues(ctx, 1)[0]

	sumGain := (lastValue.Value).(*RSIValue).gain.Mul(decimal.NewFromInt(int64(ind.Period - 1))).Add(gain)
	subLoss := (lastValue.Value).(*RSIValue).loss.Mul(decimal.NewFromInt(int64(ind.Period - 1))).Sub(loss)

	gain = sumGain.Div(decimal.NewFromInt(int64(ind.Period)))
	loss = subLoss.Div(decimal.NewFromInt(int64(ind.Period)))

	rs := gain.Div(loss)
	rsi := decimal.NewFromInt(100).Sub(decimal.NewFromInt(100).Div(decimal.NewFromInt(1).Add(rs)))

	return &RSIValue{
		gain: gain,
		loss: loss,
		RSI:  rsi,
	}
}
