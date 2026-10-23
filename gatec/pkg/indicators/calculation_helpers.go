package indicators

import "github.com/shopspring/decimal"

func sourceValue(candle Candle, source IndicatorSource) decimal.Decimal {
	switch source {
	case IndicatorSourceOpen:
		return candle.Open
	case IndicatorSourceHigh:
		return candle.High
	case IndicatorSourceLow:
		return candle.Low
	case IndicatorSourceHL2:
		return candle.High.Add(candle.Low).Div(decimal.NewFromInt(2))
	case IndicatorSourceHLC3:
		return candle.High.Add(candle.Low).Add(candle.Close).Div(decimal.NewFromInt(3))
	case IndicatorSourceOHLC4:
		return candle.Open.Add(candle.High).Add(candle.Low).Add(candle.Close).Div(decimal.NewFromInt(4))
	default:
		return candle.Close
	}
}

func emaSeries(candles []Candle, period int, source IndicatorSource) []decimal.Decimal {
	values := make([]decimal.Decimal, len(candles))
	if len(candles) == 0 || period <= 0 {
		return values
	}

	multiplier := decimal.NewFromInt(2).Div(decimal.NewFromInt(int64(period + 1)))
	oneMinusMultiplier := decimal.NewFromInt(1).Sub(multiplier)
	ema := sourceValue(candles[0], source)
	for i, candle := range candles {
		if i > 0 {
			ema = sourceValue(candle, source).Mul(multiplier).Add(ema.Mul(oneMinusMultiplier))
		}
		values[i] = ema
	}
	return values
}

func validateBasicConfigs(configs *BasicConfigs) error {
	if configs == nil {
		return ErrMissingIndicatorCfg
	}
	if configs.Period <= 0 {
		return ErrInvalidPeriod
	}
	if configs.StoredValueCount < 1 {
		return ErrInvalidValueStorageSize
	}
	return nil
}

func storeCalculatedValues(ind *indicator, values []IndicatorValue, count int) {
	if len(values) == 0 {
		return
	}
	if count > len(values) {
		count = len(values)
	}
	ind.lock.Lock()
	ind.StoredValue = append([]IndicatorValue(nil), values[len(values)-count:]...)
	ind.lock.Unlock()
}
