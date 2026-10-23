package workers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/internal/worker/events"
	indicatorsPkg "github.com/h-varmazyar/gatec/pkg/indicators"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

type IndicatorCalculator struct {
	candleRepo    *repositories.CandleRepository
	indicatorRepo *repositories.IndicatorDefinitionRepository
	marketRepo    *repositories.MarketRepository
	cfg           *configs.IndicatorCalculator
	// candlePollingWorker CandlePolling

	mu            *sync.Mutex
	pendingMu     *sync.Mutex
	pending       map[string]models.Candle
	seriesLocks   map[string]*sync.Mutex
	indicators    map[string][]indicatorsPkg.Indicator // The key is combination of marketID:timeframeID
	indicatorKeys map[uint]string
	eventBus      events.EventBus
	// signalDispatchers map[uint]*SignalPipeline
	subscription io.Closer
}

func NewIndicatorCalculator(
	cfg *configs.IndicatorCalculator,
	candleRepo *repositories.CandleRepository,
	indicatorRepo *repositories.IndicatorDefinitionRepository,
	marketRepo *repositories.MarketRepository,
	eventBus events.EventBus,
) *IndicatorCalculator {
	if cfg.SignalPipelineCount == 0 {
		cfg.SignalPipelineCount = 1
	}
	w := &IndicatorCalculator{
		candleRepo:    candleRepo,
		indicatorRepo: indicatorRepo,
		marketRepo:    marketRepo,
		cfg:           cfg,
		// candlePollingWorker: candlePollingWorker,
		mu:            new(sync.Mutex),
		pendingMu:     new(sync.Mutex),
		pending:       make(map[string]models.Candle),
		seriesLocks:   make(map[string]*sync.Mutex),
		indicators:    make(map[string][]indicatorsPkg.Indicator),
		indicatorKeys: make(map[uint]string),
		eventBus:      eventBus,
		// signalDispatchers: make(map[uint]*SignalPipeline),
	}

	// for i := range cfg.SignalPipelineCount {
	// 	w.signalDispatchers[i] = NewSignalPipelineWorker(*marketRepo)
	// 	w.signalDispatchers[i].Start()
	// }

	return w
}

func (w *IndicatorCalculator) Name() string {
	return "indicator-calculator"
}

func (w *IndicatorCalculator) Start(ctx context.Context) error {
	logrus.Info("******************** starting indicator calculator")
	if w.cfg.CryptoP0Enabled {
		if err := w.ensureCryptoP0Definitions(ctx); err != nil {
			return fmt.Errorf("enable crypto P0 indicators: %w", err)
		}
	}
	w.prepareIndicatorsValue(ctx)

	if w.eventBus != nil {
		subscription, err := w.eventBus.SubscribeCandles(ctx, w.Name(), w.handleCandle)
		if err != nil {
			return fmt.Errorf("subscribe to candle events: %w", err)
		}
		w.subscription = subscription
	}

	// dbIndicators, err := w.indicatorRepo.List(ctx, nil)
	// if err != nil {
	// 	return err
	// }

	// for _, dbIndicator := range dbIndicators {
	// 	indicator, err := w.createIndicatorFromDB(ctx, dbIndicator)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	logrus.Infof("******************** calculating for %v:%v:%v", indicator.GetProviderID(), indicator.GetMarketID(), indicator.GetTimeframeID())
	// 	go w.calculate(ctx, indicator)
	// }

	return nil
}

func (w *IndicatorCalculator) ensureCryptoP0Definitions(ctx context.Context) error {
	if w.cfg.CryptoATRPercentPeriod < 1 || w.cfg.CryptoDonchianPeriod < 1 || w.cfg.CryptoMFIPeriod < 1 || w.cfg.CryptoBollingerPeriod < 1 || w.cfg.CryptoBollingerDeviation < 1 {
		return errors.New("crypto P0 indicator periods and Bollinger deviation must be positive")
	}
	// Resolve the provider from loaded market relations because provider IDs vary by database.
	allMarkets, err := w.marketRepo.List(ctx, map[string]any{})
	if err != nil {
		return err
	}
	definitions := []struct {
		kind   string
		params map[string]any
	}{
		{"ATR_PCT", map[string]any{"period": w.cfg.CryptoATRPercentPeriod}},
		{"DONCHIAN", map[string]any{"period": w.cfg.CryptoDonchianPeriod}},
		{"MFI", map[string]any{"period": w.cfg.CryptoMFIPeriod}},
		{"CANDLE_FEATURES", map[string]any{}},
		{"BOLLINGER_BANDS", map[string]any{"period": w.cfg.CryptoBollingerPeriod, "deviation": w.cfg.CryptoBollingerDeviation}},
	}
	for _, period := range w.cfg.CryptoROCPeriods {
		if period < 1 {
			return errors.New("crypto ROC periods must be positive")
		}
		definitions = append(definitions, struct {
			kind   string
			params map[string]any
		}{"ROC", map[string]any{"period": period}})
	}
	for _, market := range allMarkets {
		if market.Provider == nil || market.Provider.Name != consts.ProviderNobitex || market.Provider.Status != consts.ProviderStatusEnable {
			continue
		}
		for _, tf := range market.TimeFrames {
			if tf.Label != "15m" && tf.Label != "1h" {
				continue
			}
			for _, item := range definitions {
				params, err := json.Marshal(item.params)
				if err != nil {
					return err
				}
				fingerprintInput := fmt.Sprintf("%s|%d|%d|%d|%s", item.kind, market.ID, market.ProviderID, tf.ID, params)
				sum := sha256.Sum256([]byte(fingerprintInput))
				definition := &models.IndicatorDefinition{MarketID: uint(market.ID), ProviderID: market.ProviderID, TimeframeID: tf.ID, Type: item.kind, Params: params, Fingerprint: hex.EncodeToString(sum[:])}
				if err := w.indicatorRepo.Upsert(ctx, definition); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (w *IndicatorCalculator) generateKey(marketID, timeframeID uint) string {
	return fmt.Sprintf("%v:%v", marketID, timeframeID)
}

func CreateIndicatorFromDefinition(dbIndicator models.IndicatorDefinition) (indicatorsPkg.Indicator, error) {
	var indicator indicatorsPkg.Indicator
	var err error

	params := map[string]any{}
	if len(dbIndicator.Params) > 0 {
		if err = json.Unmarshal(dbIndicator.Params, &params); err != nil {
			return nil, err
		}
	}

	indicatorType := strings.ToUpper(dbIndicator.Type)
	switch indicatorType {
	case string(indicatorsPkg.IndicatorTypeRSI):
		period := intParam(params, "length", "period")
		if period <= 0 {
			return nil, errors.New("RSI length is required")
		}
		cfg := &indicatorsPkg.RSIConfig{
			BasicConfigs: indicatorsPkg.BasicConfigs{
				Period:           period,
				StoredValueCount: 2,
			},
			Source: indicatorsPkg.IndicatorSourceClose,
		}
		indicator, err = indicatorsPkg.NewRSI(cfg, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
		if err != nil {
			return nil, err
		}
	case string(indicatorsPkg.IndicatorTypeSMA):
		period := intParam(params, "length", "period")
		if period <= 0 {
			return nil, errors.New("SMA length is required")
		}
		cfg := &indicatorsPkg.SMAConfigs{
			BasicConfigs: indicatorsPkg.BasicConfigs{
				Period:           period,
				StoredValueCount: 2,
			},
			Source: indicatorsPkg.IndicatorSourceClose,
		}
		indicator, err = indicatorsPkg.NewSMA(cfg, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
		if err != nil {
			return nil, err
		}
	case string(indicatorsPkg.IndicatorTypeStochastic):

		period := intParam(params, "length", "period")
		smoothK := intParam(params, "smooth_k", "smoothK")
		smoothD := intParam(params, "smooth_d", "smoothD")
		if period <= 0 || smoothK <= 0 || smoothD <= 0 {
			return nil, errors.New("Stochastic params length/smooth_k/smooth_d are required")
		}
		cfg := &indicatorsPkg.StochasticConfigs{
			BasicConfigs: indicatorsPkg.BasicConfigs{
				Period:           period,
				StoredValueCount: 2,
			},
			SmoothK: int64(smoothK),
			SmoothD: int64(smoothD),
		}
		indicator, err = indicatorsPkg.NewStochastic(cfg, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
		if err != nil {
			return nil, err
		}
	case string(indicatorsPkg.IndicatorTypeBollingerBands):
		period := intParam(params, "length", "period")
		deviation := intParam(params, "deviation", "multiplier")
		if period <= 0 || deviation <= 0 {
			return nil, errors.New("Bollinger params length/deviation are required")
		}
		cfg := &indicatorsPkg.BollingerBandsConfigs{
			BasicConfigs: indicatorsPkg.BasicConfigs{
				Period:           period,
				StoredValueCount: 2,
			},
			Multiplier: int64(deviation),
			Source:     indicatorsPkg.IndicatorSourceClose,
		}
		indicator, err = indicatorsPkg.NewBollingerBands(cfg, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
		if err != nil {
			return nil, err
		}
	case string(indicatorsPkg.IndicatorTypeATRPercent), "ATR_PERCENT":
		period := intParam(params, "length", "period")
		if period <= 0 {
			return nil, errors.New("ATR percent period must be positive")
		}
		indicator, err = indicatorsPkg.NewATRPercent(&indicatorsPkg.WindowIndicatorConfig{BasicConfigs: indicatorsPkg.BasicConfigs{Period: period, StoredValueCount: 2}}, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
	case string(indicatorsPkg.IndicatorTypeDonchian):
		period := intParam(params, "length", "period")
		if period <= 0 {
			return nil, errors.New("Donchian period must be positive")
		}
		indicator, err = indicatorsPkg.NewDonchian(&indicatorsPkg.WindowIndicatorConfig{BasicConfigs: indicatorsPkg.BasicConfigs{Period: period, StoredValueCount: 2}}, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
	case string(indicatorsPkg.IndicatorTypeROC):
		period := intParam(params, "length", "period")
		if period <= 0 {
			return nil, errors.New("ROC period must be positive")
		}
		indicator, err = indicatorsPkg.NewROC(&indicatorsPkg.WindowIndicatorConfig{BasicConfigs: indicatorsPkg.BasicConfigs{Period: period, StoredValueCount: 2}}, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
	case string(indicatorsPkg.IndicatorTypeMFI), "MONEY_FLOW_INDEX":
		period := intParam(params, "length", "period")
		if period <= 0 {
			return nil, errors.New("MFI period must be positive")
		}
		indicator, err = indicatorsPkg.NewMFI(&indicatorsPkg.WindowIndicatorConfig{BasicConfigs: indicatorsPkg.BasicConfigs{Period: period, StoredValueCount: 2}}, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
	case string(indicatorsPkg.IndicatorTypeCandleFeatures):
		indicator, err = indicatorsPkg.NewCandleFeatures(&indicatorsPkg.WindowIndicatorConfig{BasicConfigs: indicatorsPkg.BasicConfigs{Period: 1, StoredValueCount: 2}}, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
	case string(indicatorsPkg.IndicatorTypeEMA):
		period := intParam(params, "length", "period")
		if period <= 0 {
			return nil, errors.New("EMA length is required")
		}
		cfg := &indicatorsPkg.EMAConfigs{
			BasicConfigs: indicatorsPkg.BasicConfigs{
				Period:           period,
				StoredValueCount: 2,
			},
			Source: indicatorsPkg.IndicatorSourceClose,
		}
		indicator, err = indicatorsPkg.NewEMA(cfg, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
		if err != nil {
			return nil, err
		}
	case string(indicatorsPkg.IndicatorTypeRVI):
		period := intParam(params, "length", "period")
		smoothingPeriod := intParam(params, "smoothing_period", "smoothingPeriod")
		if period <= 0 || smoothingPeriod <= 0 {
			return nil, errors.New("RVI params length/smoothing_period are required")
		}
		cfg := &indicatorsPkg.RVIConfigs{
			BasicConfigs:    indicatorsPkg.BasicConfigs{Period: period, StoredValueCount: 2},
			SmoothingPeriod: smoothingPeriod,
			Source:          sourceParam(params),
		}
		indicator, err = indicatorsPkg.NewRVI(cfg, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
	case string(indicatorsPkg.IndicatorTypeEMASlope):
		period := intParam(params, "ema_period", "emaPeriod", "length", "period")
		slopePeriod := intParam(params, "slope_period", "slopePeriod")
		if period <= 0 || slopePeriod <= 0 {
			return nil, errors.New("EMA slope params ema_period/slope_period are required")
		}
		cfg := &indicatorsPkg.EMASlopeConfigs{
			BasicConfigs: indicatorsPkg.BasicConfigs{Period: period, StoredValueCount: 2},
			SlopePeriod:  slopePeriod,
			Source:       sourceParam(params),
		}
		indicator, err = indicatorsPkg.NewEMASlope(cfg, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
	case string(indicatorsPkg.IndicatorTypeEMADistance):
		period := intParam(params, "ema_period", "emaPeriod", "length", "period")
		if period <= 0 {
			return nil, errors.New("EMA distance length is required")
		}
		cfg := &indicatorsPkg.EMADistanceConfigs{
			BasicConfigs: indicatorsPkg.BasicConfigs{Period: period, StoredValueCount: 2},
			Source:       sourceParam(params),
		}
		indicator, err = indicatorsPkg.NewEMADistance(cfg, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
	case string(indicatorsPkg.IndicatorTypeAverageVolume):
		period := intParam(params, "length", "period")
		if period <= 0 {
			return nil, errors.New("average volume length is required")
		}
		cfg := &indicatorsPkg.AverageVolumeConfigs{BasicConfigs: indicatorsPkg.BasicConfigs{Period: period, StoredValueCount: 2}}
		indicator, err = indicatorsPkg.NewAverageVolume(cfg, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
	case string(indicatorsPkg.IndicatorTypeVolumeSpike):
		period := intParam(params, "length", "period")
		multiplier := decimalParam(params, "multiplier", "threshold")
		if period <= 0 || !multiplier.GreaterThan(decimal.Zero) {
			return nil, errors.New("volume spike params length/multiplier are required")
		}
		cfg := &indicatorsPkg.VolumeSpikeConfigs{
			BasicConfigs: indicatorsPkg.BasicConfigs{Period: period, StoredValueCount: 2},
			Multiplier:   multiplier,
		}
		indicator, err = indicatorsPkg.NewVolumeSpike(cfg, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
	case "MACD":
		fastPeriod := intParam(params, "fast_period", "fastPeriod")
		slowPeriod := intParam(params, "slow_period", "slowPeriod")
		signalPeriod := intParam(params, "signal_period", "signalPeriod")
		if fastPeriod <= 0 || slowPeriod <= 0 || signalPeriod <= 0 {
			return nil, errors.New("MACD params fast_period/slow_period/signal_period are required")
		}
		cfg := &indicatorsPkg.MACDConfigs{
			BasicConfigs: indicatorsPkg.BasicConfigs{StoredValueCount: 2},
			FastPeriod:   fastPeriod,
			SlowPeriod:   slowPeriod,
			SignalPeriod: signalPeriod,
		}
		indicator, err = indicatorsPkg.NewMACD(cfg, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
		if err != nil {
			return nil, err
		}
	case "ATR":
		period := intParam(params, "length", "period")
		if period <= 0 {
			return nil, errors.New("ATR length is required")
		}
		cfg := &indicatorsPkg.ATRConfigs{BasicConfigs: indicatorsPkg.BasicConfigs{Period: period, StoredValueCount: 2}}
		indicator, err = indicatorsPkg.NewATR(cfg, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
		if err != nil {
			return nil, err
		}
	case "ADX":
		period := intParam(params, "length", "period")
		if period <= 0 {
			return nil, errors.New("ADX length is required")
		}
		cfg := &indicatorsPkg.ADXConfigs{BasicConfigs: indicatorsPkg.BasicConfigs{Period: period, StoredValueCount: 2}}
		indicator, err = indicatorsPkg.NewADX(cfg, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
		if err != nil {
			return nil, err
		}
	case "VWAP":
		cfg := &indicatorsPkg.VWAPConfigs{BasicConfigs: indicatorsPkg.BasicConfigs{Period: 1, StoredValueCount: 2}}
		indicator, err = indicatorsPkg.NewVWAP(cfg, dbIndicator.ProviderID, dbIndicator.MarketID, dbIndicator.TimeframeID)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported indicator type: %s", dbIndicator.Type)
	}

	if err != nil {
		return nil, err
	}
	indicator.SetID(dbIndicator.ID)
	return indicator, nil
}

func intParam(params map[string]any, keys ...string) int {
	for _, key := range keys {
		if val, ok := params[key]; ok {
			switch v := val.(type) {
			case int:
				return v
			case int64:
				return int(v)
			case float64:
				return int(v)
			case string:
				parsed, err := decimal.NewFromString(v)
				if err == nil {
					return int(parsed.IntPart())
				}
			}
		}
	}
	return 0
}

func decimalParam(params map[string]any, keys ...string) decimal.Decimal {
	for _, key := range keys {
		if val, ok := params[key]; ok {
			switch v := val.(type) {
			case int:
				return decimal.NewFromInt(int64(v))
			case int64:
				return decimal.NewFromInt(v)
			case float64:
				return decimal.NewFromFloat(v)
			case string:
				parsed, err := decimal.NewFromString(v)
				if err == nil {
					return parsed
				}
			}
		}
	}
	return decimal.Zero
}

func sourceParam(params map[string]any) indicatorsPkg.IndicatorSource {
	value, ok := params["source"].(string)
	if !ok || value == "" {
		return indicatorsPkg.IndicatorSourceClose
	}
	source := indicatorsPkg.IndicatorSource(strings.ToUpper(value))
	if !source.IsValidSource() {
		return indicatorsPkg.IndicatorSourceClose
	}
	return source
}

func (w *IndicatorCalculator) handleCandle(ctx context.Context, candle models.Candle) error {
	seriesKey := w.generateKey(candle.MarketID, candle.TimeframeID)
	w.pendingMu.Lock()
	seriesLock := w.seriesLocks[seriesKey]
	if seriesLock == nil {
		seriesLock = new(sync.Mutex)
		w.seriesLocks[seriesKey] = seriesLock
	}
	w.pendingMu.Unlock()
	seriesLock.Lock()
	defer seriesLock.Unlock()
	w.pendingMu.Lock()
	previous, exists := w.pending[seriesKey]
	if !exists {
		w.pending[seriesKey] = candle
		w.pendingMu.Unlock()
		return nil
	}
	if candle.Time == previous.Time {
		w.pending[seriesKey] = candle
		w.pendingMu.Unlock()
		return nil
	}
	if candle.Time < previous.Time {
		w.pendingMu.Unlock()
		return nil
	}
	w.pending[seriesKey] = candle
	w.pendingMu.Unlock()
	return w.processCandle(ctx, previous)
}

// processCandle receives only a candle superseded by the next candle in its
// series, which makes the prior bar closed and safe for strategy evaluation.
func (w *IndicatorCalculator) processCandle(ctx context.Context, candle models.Candle) error {
	marketID := candle.MarketID
	timeframeID := candle.TimeframeID

	key := w.generateKey(marketID, timeframeID)

	w.mu.Lock()
	candleIndicators, ok := w.indicators[key]
	if !ok {
		w.mu.Unlock()
		return nil
	}
	indicatorsSnapshot := append([]indicatorsPkg.Indicator(nil), candleIndicators...)
	indicatorKeys := make(map[uint]string, len(indicatorsSnapshot))
	for _, indicator := range indicatorsSnapshot {
		indicatorKeys[indicator.GetID()] = w.indicatorKeys[indicator.GetID()]
	}
	w.mu.Unlock()

	indCandle := indicatorsPkg.Candle{
		Time:   candle.Time,
		Open:   candle.Open,
		High:   candle.High,
		Low:    candle.Low,
		Close:  candle.Close,
		Volume: candle.Volume,
	}

	indicatorsValue := make(map[string]decimal.Decimal)
	for _, indicator := range indicatorsSnapshot {
		indicatorCtx, cancelFunc := context.WithTimeout(ctx, time.Second*5)
		indicator.UpdateLast(indicatorCtx, indCandle)
		values := indicator.GetValues(ctx, 1)
		cancelFunc()
		if len(values) == 0 || values[0].Value == nil {
			continue
		}
		fields := values[0].Value.Fields()
		key := indicatorKeys[indicator.GetID()]
		if key == "" {
			continue
		}
		baseKey := stripIndicatorPeriod(key)
		aliases := indicatorKeyAliases(baseKey)
		if len(fields) == 1 {
			for field, value := range fields {
				indicatorsValue[key] = value
				for _, alias := range aliases {
					indicatorsValue[alias] = value
					if field != alias {
						indicatorsValue[alias+"."+field] = value
					}
				}
			}
			continue
		}
		for field, value := range fields {
			for _, alias := range aliases {
				indicatorsValue[alias+"."+field] = value
			}
			indicatorsValue[key+"."+field] = value
			if field == baseKey {
				indicatorsValue[key] = value
				for _, alias := range aliases {
					indicatorsValue[alias] = value
				}
			}
			if key == "macd" && field == "histogram" {
				indicatorsValue["macd.hist"] = value
			}
			if field == key {
				indicatorsValue[key] = value
			}
		}
	}
	if w.eventBus != nil && len(indicatorsValue) > 0 {
		if err := w.eventBus.PublishSnapshot(ctx, events.CandleSnapshot{Candle: candle, Indicators: indicatorsValue}); err != nil {
			return fmt.Errorf("publish calculated candle snapshot: %w", err)
		}
	}

	// item := marketID % w.cfg.SignalPipelineCount
	// w.signalDispatchers[item].EnqueueNewEvent(marketID, candle, indicatorsValue)

	return nil
}

func stripIndicatorPeriod(key string) string {
	index := strings.LastIndexByte(key, '_')
	if index < 0 || index == len(key)-1 {
		return key
	}
	for _, r := range key[index+1:] {
		if r < '0' || r > '9' {
			return key
		}
	}
	return key[:index]
}

func indicatorKeyAliases(base string) []string {
	if base == "stochastic" {
		return []string{"stochastic", "stochastics"}
	}
	return []string{base}
}

// func (w *IndicatorCalculator) calculate(ctx context.Context, indicator indicatorsPkg.Indicator) {
// key := w.generateKey(indicator.GetMarketID(), indicator.GetTimeframeID())
// w.mu.Lock()

// if indicators, ok := w.indicators[key]; ok {
// 	for _, ind := range indicators {
// 		if ind.GetID() == indicator.GetID() {
// 			w.mu.Unlock()
// 			return
// 		}
// 	}
// }
// w.mu.Unlock()

// candles, err := w.candlePollingWorker.GetCandles(indicator.GetProviderID(), indicator.GetMarketID(), indicator.GetTimeframeID(), 100)
// if err != nil {
// 	logrus.WithError(err).Errorf("failed to get candles for %v:%v:%v", indicator.GetProviderID(), indicator.GetMarketID(), indicator.GetTimeframeID())
// 	return
// }

// indCandles := make([]indicatorsPkg.Candle, len(candles))
// for i := 0; i < len(candles); i++ {
// 	indCandles[i] = indicatorsPkg.Candle{
// 		Time:  candles[i].Time,
// 		Open:  candles[i].Open,
// 		High:  candles[i].High,
// 		Low:   candles[i].Low,
// 		Close: candles[i].Close,
// 	}
// }

// indicator.Calculate(ctx, indCandles)

// w.mu.Lock()
// defer w.mu.Unlock()

// if list, ok := w.indicators[key]; !ok {
// 	w.indicators[key] = make([]indicatorsPkg.Indicator, 0)
// } else {
// 	for _, i := range list {
// 		if i.GetID() == indicator.GetID() {
// 			return
// 		}
// 	}
// }
// w.indicators[key] = append(w.indicators[key], indicator)
// }

func (w *IndicatorCalculator) prepareIndicatorsValue(ctx context.Context) {
	dbIndicators, err := w.indicatorRepo.List(ctx)
	if err != nil {
		return
	}

	logrus.Infof("ind count: %v", len(dbIndicators))

	for _, dbIndicator := range dbIndicators {
		indicator, err := CreateIndicatorFromDefinition(dbIndicator)
		if err != nil {
			continue
		}
		key := w.generateKey(indicator.GetMarketID(), indicator.GetTimeframeID())
		indicatorKey := canonicalIndicatorKey(dbIndicator)

		candles, err := w.candleRepo.List(ctx, indicator.GetMarketID(), indicator.GetTimeframeID(), 500)
		if err != nil {
			logrus.WithError(err).Errorf("failed to get candles for %v:%v:%v", indicator.GetProviderID(), indicator.GetMarketID(), indicator.GetTimeframeID())
			continue
		}

		indCandles := make([]indicatorsPkg.Candle, len(candles))
		if len(candles) > 0 {
			// The newest stored candle may still be forming; wait for the next
			// market event before admitting it to indicator history. Retain it as
			// pending so the first later candle can close it in order.
			latest := candles[len(candles)-1]
			seriesKey := w.generateKey(indicator.GetMarketID(), indicator.GetTimeframeID())
			w.pendingMu.Lock()
			if pending, ok := w.pending[seriesKey]; !ok || latest.Time > pending.Time {
				w.pending[seriesKey] = latest
			}
			w.pendingMu.Unlock()
			candles = candles[:len(candles)-1]
			indCandles = make([]indicatorsPkg.Candle, len(candles))
		}
		for i := range candles {
			indCandles[i] = indicatorsPkg.Candle{
				Time:   candles[i].Time,
				Open:   candles[i].Open,
				High:   candles[i].High,
				Low:    candles[i].Low,
				Close:  candles[i].Close,
				Volume: candles[i].Volume,
			}
		}

		indicator.Calculate(ctx, indCandles)

		w.mu.Lock()
		w.indicatorKeys[indicator.GetID()] = indicatorKey
		list := w.indicators[key]
		duplicate := false
		for _, existing := range list {
			if existing.GetID() == indicator.GetID() {
				duplicate = true
				break
			}
		}
		if !duplicate {
			w.indicators[key] = append(list, indicator)
		}
		w.mu.Unlock()
	}
}

func canonicalIndicatorKey(value models.IndicatorDefinition) string {
	typeName := strings.ToLower(value.Type)
	base := map[string]string{"bollinger_bands": "bollinger", "average_volume": "average_volume", "volume_spike": "volume_spike", "ema_distance": "ema_distance", "ema_slope": "ema_slope"}[typeName]
	if base == "" {
		base = typeName
	}
	params := map[string]any{}
	_ = json.Unmarshal(value.Params, &params)
	period := intParam(params, "length", "period")
	switch base {
	case "ema", "sma", "rsi", "atr", "atr_pct", "donchian", "roc", "mfi", "adx", "bollinger", "stochastic", "average_volume", "volume_spike":
		if period > 0 {
			return fmt.Sprintf("%s_%d", base, period)
		}
	}
	return base
}

// Stop gracefully stops the worker
func (w *IndicatorCalculator) Stop() error {
	if w.subscription != nil {
		_ = w.subscription.Close()
	}
	logrus.Info("indicator calculator worker stopped")
	return nil
}
