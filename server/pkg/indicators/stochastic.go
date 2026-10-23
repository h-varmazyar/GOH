package indicators

import (
	"context"
	"math"
	"sync"

	"github.com/shopspring/decimal"
)

type Stochastic struct {
	indicator
	*StochasticConfigs
	fasts       []decimal.Decimal
	lastCandles []Candle
	// lastValues  []*IndicatorValue
}

type StochasticConfigs struct {
	BasicConfigs
	SmoothK int64
	SmoothD int64
}

type StochasticValue struct {
	IndexK decimal.Decimal
	IndexD decimal.Decimal
}

func (s StochasticValue) Fields() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{
		"indexK": s.IndexK,
		"indexD": s.IndexD,
	}
}

func (*StochasticValue) isIndicatorValue() {}

func NewStochastic(configs *StochasticConfigs, providerID uint, marketID, timeframeID uint) (*Stochastic, error) {
	return &Stochastic{
		indicator: indicator{
			lock:        new(sync.Mutex),
			providerID:  providerID,
			marketID:    marketID,
			timeframeID: timeframeID,
		},
		StochasticConfigs: configs,
		fasts:             make([]decimal.Decimal, configs.SmoothK),
		lastCandles:       make([]Candle, configs.Period),
		// lastValues:        make([]*IndicatorValue, configs.SmoothD),
	}, nil
}

func (ind *Stochastic) GetType() IndicatorType {
	return IndicatorTypeStochastic
}

func (ind *Stochastic) GetRequiredCandleCount() int {
	return 2
}

func (ind *Stochastic) Calculate(ctx context.Context, candles []Candle) []IndicatorValue {
	ind.fasts = make([]decimal.Decimal, ind.SmoothK)
	ind.lastCandles = make([]Candle, ind.Period)

	values := make([]IndicatorValue, len(candles))

	for i := 0; i < ind.Period-1; i++ {
		values[i] = IndicatorValue{
			Time: candles[i].Time,
		}
	}

	for i := ind.Period - 1; i < len(candles); i++ {
		lowest := decimal.NewFromFloat(math.MaxFloat64)
		highest := decimal.Zero
		for j := i - (ind.Period - 1); j <= i; j++ {
			if candles[j].Low.LessThan(lowest) {
				lowest = candles[j].Low
			}
			if candles[j].High.GreaterThan(highest) {
				highest = candles[j].High
			}
		}

		fd1 := candles[i].Close.Sub(lowest)
		fd2 := highest.Sub(lowest)
		fm2 := fd1.Div(fd2)
		fm1 := decimal.NewFromInt(100)
		fastK := fm1.Mul(fm2)

		values[i] = IndicatorValue{
			Time:  candles[i].Time,
			Value: new(StochasticValue),
		}

		lastKIndex := decimal.Zero
		if values[i-1].Value != nil {
			lastKIndex = (values[i-1].Value.(*StochasticValue)).IndexK
		}

		kd1 := fastK.Sub(ind.fasts[0])
		kd2 := decimal.NewFromInt(ind.SmoothK)
		kd := kd1.Div(kd2)
		indexK := kd.Add(lastKIndex)

		(values[i].Value.(*StochasticValue)).IndexK = indexK
		(values[i].Value.(*StochasticValue)).IndexD = calculateIndexD(values[i-int(ind.SmoothD)+1 : i+1])

		ind.fasts = ind.fasts[1:]
		ind.fasts = append(ind.fasts, fastK)
	}

	ind.lastCandles = candles[len(candles)-ind.Period:]
	// ind.lastValues = values[len(values)-int(ind.SmoothD):]

	for _, v := range values {
		ind.updateLastValue(ctx, v)
	}

	return values
}

func (conf *Stochastic) UpdateLast(ctx context.Context, candle Candle) {
	lastValue := conf.GetValues(ctx, 1)[0]
	if candle.Time > conf.lastCandles[len(conf.lastCandles)-1].Time {
		conf.fasts = conf.fasts[1:]
		conf.fasts = append(conf.fasts, decimal.Zero)

		conf.lastCandles = conf.lastCandles[1:]
		conf.lastCandles = append(conf.lastCandles, candle)

		lastValue = IndicatorValue{
			Time:  candle.Time,
			Value: new(StochasticValue),
		}
	}

	lowest := decimal.NewFromFloat(math.MaxFloat64)
	highest := decimal.Zero
	for _, lastCandle := range conf.lastCandles {
		if lastCandle.Low.LessThan(lowest) {
			lowest = lastCandle.Low
		}
		if lastCandle.High.GreaterThan(highest) {
			highest = lastCandle.High
		}
	}

	fastK := (decimal.NewFromInt(100).Mul(candle.Close.Sub(lowest))).Div(highest.Sub(lowest))
	(lastValue.Value).(*StochasticValue).IndexK = (fastK.Sub(conf.fasts[0])).Div(decimal.NewFromInt(conf.SmoothK)).Add((lastValue.Value).(*StochasticValue).IndexK)
	(lastValue.Value).(*StochasticValue).IndexD = calculateIndexD(conf.GetValues(ctx, int(conf.SmoothD)))

	conf.fasts[len(conf.fasts)-1] = fastK
	conf.lastCandles[len(conf.lastCandles)-1] = candle

	conf.updateLastValue(ctx, lastValue)
}

func (conf *Stochastic) hasValidConfigs() error {
	return nil
}

func calculateIndexD(values []IndicatorValue) decimal.Decimal {
	sum := decimal.Zero
	for _, value := range values {
		if value.Value == nil {
			return decimal.Zero
		}
		k := (value.Value).(*StochasticValue).IndexK
		sum = sum.Add(k)
	}

	d2 := decimal.NewFromInt(int64(len(values)))
	return sum.Div(d2)
}
