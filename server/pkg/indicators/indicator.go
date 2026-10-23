package indicators

import (
	"context"
	"sync"

	"github.com/shopspring/decimal"
)

type IndicatorSource string

const (
	IndicatorSourceOHLC4 = "OHLC4"
	IndicatorSourceClose = "CLOSE"
	IndicatorSourceOpen  = "OPEN"
	IndicatorSourceHigh  = "HIGH"
	IndicatorSourceHLC3  = "HLC3"
	IndicatorSourceLow   = "LOW"
	IndicatorSourceHL2   = "HL2"
)

func (s IndicatorSource) IsValidSource() bool {
	switch s {
	case IndicatorSourceOHLC4,
		IndicatorSourceClose,
		IndicatorSourceOpen,
		IndicatorSourceHigh,
		IndicatorSourceHLC3,
		IndicatorSourceLow,
		IndicatorSourceHL2:
		return true

	}
	return false
}

type IndicatorType string

const (
	IndicatorTypeRSI            = "RSI"
	IndicatorTypeStochastic     = "STOCHASTIC"
	IndicatorTypeSMA            = "SMA"
	IndicatorTypeEMA            = "EMA"
	IndicatorTypeBollingerBands = "BOLLINGER_BANDS"
	IndicatorTypeRVI            = "RVI"
	IndicatorTypeEMASlope       = "EMA_SLOPE"
	IndicatorTypeEMADistance    = "EMA_DISTANCE"
	IndicatorTypeVolumeSpike    = "VOLUME_SPIKE"
	IndicatorTypeAverageVolume  = "AVERAGE_VOLUME"
	IndicatorTypeATRPercent     = "ATR_PCT"
	IndicatorTypeDonchian       = "DONCHIAN"
	IndicatorTypeROC            = "ROC"
	IndicatorTypeMFI            = "MFI"
	IndicatorTypeCandleFeatures = "CANDLE_FEATURES"
)

type Indicator interface {
	GetID() uint
	SetID(id uint)
	GetMarketID() uint
	GetProviderID() uint
	GetTimeframeID() uint
	GetRequiredCandleCount() int
	GetType() IndicatorType
	Calculate(ctx context.Context, candles []Candle) []IndicatorValue
	UpdateLast(ctx context.Context, candle Candle)
	GetValues(ctx context.Context, count int) []IndicatorValue
	hasValidConfigs() error
}

type indicator struct {
	id          uint
	lock        *sync.Mutex
	providerID  uint
	marketID    uint
	timeframeID uint
	StoredValue []IndicatorValue
}

type BasicConfigs struct {
	Period           int
	StoredValueCount int
}

type Candle struct {
	Time   int64
	Open   decimal.Decimal
	High   decimal.Decimal
	Low    decimal.Decimal
	Close  decimal.Decimal
	Volume decimal.Decimal
}

type IsIndicatorValue interface {
	Fields() map[string]decimal.Decimal
}

type IndicatorValue struct {
	Time  int64
	Value IsIndicatorValue
}

func (ind *indicator) GetProviderID() uint {
	return ind.providerID
}

func (ind *indicator) GetMarketID() uint {
	return ind.marketID
}

func (ind *indicator) GetTimeframeID() uint {
	return ind.timeframeID
}

func (ind *indicator) GetID() uint {
	return ind.id
}

func (ind *indicator) SetID(id uint) {
	ind.id = id
}

func (ind *indicator) GetValues(_ context.Context, count int) []IndicatorValue {
	ind.lock.Lock()
	defer ind.lock.Unlock()

	if count <= 0 || len(ind.StoredValue) == 0 {
		return []IndicatorValue{}
	}
	if count > len(ind.StoredValue) {
		count = len(ind.StoredValue)
	}
	from := len(ind.StoredValue) - count

	return append([]IndicatorValue{}, ind.StoredValue[from:]...)
}

func (ind *indicator) updateLastValue(_ context.Context, value IndicatorValue) {
	ind.lock.Lock()
	defer ind.lock.Unlock()

	if len(ind.StoredValue) == 0 {
		ind.StoredValue = append(ind.StoredValue, value)
		return
	}
	last := len(ind.StoredValue) - 1
	if ind.StoredValue[last].Time == value.Time {
		ind.StoredValue[last] = value
	} else if ind.StoredValue[last].Time < value.Time {
		ind.StoredValue = append(ind.StoredValue, value)
	}
}
