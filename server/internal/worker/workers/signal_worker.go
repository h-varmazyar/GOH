package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/internal/services/signal/definition"
	"github.com/h-varmazyar/gatec/internal/services/signal/domain"
	"github.com/h-varmazyar/gatec/internal/services/signal/engine"
	signalReplay "github.com/h-varmazyar/gatec/internal/services/signal/replay"
	"github.com/h-varmazyar/gatec/internal/services/signal/risk"
	"github.com/h-varmazyar/gatec/internal/services/signal/state"
	"github.com/h-varmazyar/gatec/internal/worker/events"
	indicatorsPkg "github.com/h-varmazyar/gatec/pkg/indicators"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type SignalWorker struct {
	cfg               *configs.SignalWorker
	eventBus          events.EventBus
	markets           *repositories.MarketRepository
	providers         *repositories.ProviderRepository
	candles           *repositories.CandleRepository
	indicators        *repositories.IndicatorDefinitionRepository
	strategies        *repositories.StrategyDefinitionRepository
	signals           *repositories.SignalRepository
	subscription      io.Closer
	cancel            context.CancelFunc
	mu                sync.Mutex
	started           bool
	done              sync.WaitGroup
	router            *engine.Router
	loader            *state.DatabaseSnapshotLoader
	marketInfo        map[uint]signalMarketInfo
	seriesIndicators  map[string][]models.IndicatorDefinition
	definitions       domain.StrategyRepository
	stats             signalWorkerStats
	statsLoaded       atomic.Int64
	replayMode        bool
	replayStrategyID  string
	replayDefinitions []domain.StrategyDefinition
	replayMu          sync.Mutex
	replaySignals     []domain.Signal
	replayClock       *virtualClock
}

type signalMarketInfo struct {
	market, symbol string
	timeframes     map[uint]signalTimeframeInfo
}
type signalTimeframeInfo struct {
	label    string
	duration time.Duration
}
type signalWorkerStats struct{ events, duplicate, late, stale, emitted, evalErrors atomic.Uint64 }

type ReplayOptions struct {
	StrategyID, Market                 string
	Symbols                            []string
	From, To                           time.Time
	FeePercent, SlippagePercent, Split float64
}

type virtualClock struct {
	mu  sync.RWMutex
	now time.Time
}

func (c *virtualClock) Now() time.Time      { c.mu.RLock(); defer c.mu.RUnlock(); return c.now }
func (c *virtualClock) Set(value time.Time) { c.mu.Lock(); c.now = value; c.mu.Unlock() }

func NewSignalWorker(cfg *configs.SignalWorker, bus events.EventBus, markets *repositories.MarketRepository, providers *repositories.ProviderRepository, candles *repositories.CandleRepository, indicators *repositories.IndicatorDefinitionRepository, strategies *repositories.StrategyDefinitionRepository, signals *repositories.SignalRepository) *SignalWorker {
	return &SignalWorker{
		cfg:        cfg,
		eventBus:   bus,
		markets:    markets,
		providers:  providers,
		candles:    candles,
		indicators: indicators,
		strategies: strategies,
		signals:    signals,
	}
}

func (w *SignalWorker) Name() string { return "signal-worker" }

func (w *SignalWorker) Start(parent context.Context) error {
	w.mu.Lock()
	if w.started {
		w.mu.Unlock()
		return nil
	}
	w.started = true
	w.mu.Unlock()
	if w.cfg == nil || (!w.replayMode && (w.eventBus == nil || w.signals == nil)) || w.markets == nil || w.providers == nil || w.candles == nil || w.indicators == nil || w.strategies == nil {
		w.resetStart()
		return fmt.Errorf("signal-worker dependencies are incomplete")
	}
	ctx, cancel := context.WithCancel(parent)
	w.cancel = cancel
	if err := w.initialize(ctx); err != nil {
		cancel()
		w.resetStart()
		return err
	}
	subscription, err := w.eventBus.SubscribeSnapshots(ctx, w.Name(), w.cfg.Shards, w.handleSnapshot)
	if err != nil {
		cancel()
		w.resetStart()
		return err
	}
	w.subscription = subscription
	watch, err := w.definitions.Watch(ctx)
	if err != nil {
		_ = subscription.Close()
		cancel()
		w.resetStart()
		return err
	}
	w.done.Add(2)
	go w.watchDefinitions(ctx, watch)
	go w.background(ctx)
	logrus.WithFields(logrus.Fields{"strategies_loaded": w.statsLoaded.Load(), "source": w.cfg.StrategySource}).Info("signal worker started")
	return nil
}

func (w *SignalWorker) resetStart() { w.mu.Lock(); w.started = false; w.mu.Unlock() }

func (w *SignalWorker) captureReplaySignal(_ context.Context, signal domain.Signal) error {
	w.replayMu.Lock()
	w.replaySignals = append(w.replaySignals, signal)
	w.replayMu.Unlock()
	return nil
}

// Replay feeds historical snapshots through the live router and position manager,
// while capturing output in memory instead of writing to production tables or NATS.
func (w *SignalWorker) Replay(ctx context.Context, options ReplayOptions) (signalReplay.Report, error) {
	if options.StrategyID == "" || options.Market == "" || options.From.IsZero() || !options.To.After(options.From) {
		return signalReplay.Report{}, fmt.Errorf("strategy, market, and an increasing from/to range are required")
	}
	if options.Split == 0 {
		options.Split = .7
	}
	w.replayMode = true
	w.replayStrategyID = options.StrategyID
	w.replaySignals = nil
	if err := w.initialize(ctx); err != nil {
		return signalReplay.Report{}, err
	}
	clock := &virtualClock{}
	w.replayClock = clock
	w.router.SetClock(clock)
	keys, err := w.replaySeries(options)
	if err != nil {
		return signalReplay.Report{}, err
	}
	stateStore := w.router.StateStore()
	for _, key := range keys {
		capacity := w.router.CapacityFor(key)
		if capacity < 1 {
			capacity = 2
		}
		if configurable, ok := stateStore.(interface {
			SetCapacity(domain.SeriesKey, int) error
		}); ok {
			if err := configurable.SetCapacity(key, capacity); err != nil {
				return signalReplay.Report{}, err
			}
		}
		duration := w.timeframeDuration(key)
		if duration <= 0 {
			continue
		}
		warmFrom := options.From.Add(-time.Duration(capacity) * duration)
		warm, loadErr := w.loader.LoadRange(ctx, key, warmFrom, options.From.Add(-time.Nanosecond))
		if loadErr != nil {
			logrus.WithError(loadErr).WithFields(logrus.Fields{"market": key.Market, "symbol": key.Symbol, "timeframe": key.Timeframe}).Warn("replay warm-up history unavailable")
			continue
		}
		sort.Slice(warm, func(i, j int) bool { return warm[i].Candle.OpenTime.Before(warm[j].Candle.OpenTime) })
		for _, snapshot := range warm {
			if _, err := stateStore.Apply(ctx, snapshot); err != nil {
				return signalReplay.Report{}, err
			}
		}
	}
	eventsToRun := make([]domain.Snapshot, 0)
	for _, key := range keys {
		items, loadErr := w.loader.LoadRange(ctx, key, options.From.Add(-w.timeframeDuration(key)), options.To)
		if loadErr != nil {
			return signalReplay.Report{}, loadErr
		}
		for _, item := range items {
			if item.Candle.CloseTime.Before(options.From) || item.Candle.CloseTime.After(options.To) {
				continue
			}
			eventsToRun = append(eventsToRun, item)
		}
	}
	if len(eventsToRun) == 0 {
		return signalReplay.Report{}, fmt.Errorf("no closed candle snapshots found in the requested replay range")
	}
	sort.SliceStable(eventsToRun, func(i, j int) bool {
		a, b := eventsToRun[i], eventsToRun[j]
		if !a.Candle.CloseTime.Equal(b.Candle.CloseTime) {
			return a.Candle.CloseTime.Before(b.Candle.CloseTime)
		}
		ad, bd := w.timeframeDuration(domain.SeriesKey{Market: a.Candle.Market, Symbol: a.Candle.Symbol, Timeframe: a.Candle.Timeframe}), w.timeframeDuration(domain.SeriesKey{Market: b.Candle.Market, Symbol: b.Candle.Symbol, Timeframe: b.Candle.Timeframe})
		if ad != bd {
			return ad > bd
		}
		if a.Candle.Symbol != b.Candle.Symbol {
			return a.Candle.Symbol < b.Candle.Symbol
		}
		return a.Candle.Timeframe < b.Candle.Timeframe
	})
	nextOpen := make(map[string]decimal.Decimal)
	lastBySeries := make(map[string]int)
	for index, event := range eventsToRun {
		key := seriesIdentity(event.Candle.Market, event.Candle.Symbol, event.Candle.Timeframe)
		if previous, ok := lastBySeries[key]; ok {
			nextOpen[replayEventKey(eventsToRun[previous])] = event.Candle.Open
		}
		lastBySeries[key] = index
	}
	for _, event := range eventsToRun {
		clock.Set(event.Candle.CloseTime)
		optionsForEvent := engine.ProcessOptions{SlippagePercent: decimal.NewFromFloat(options.SlippagePercent)}
		key := replayEventKey(event)
		if fill, ok := nextOpen[key]; ok {
			optionsForEvent.NextOpen = &fill
		} else {
			optionsForEvent.NoEntry = true
		}
		if _, err := w.router.Process(ctx, event, optionsForEvent); err != nil {
			return signalReplay.Report{}, err
		}
	}
	w.replayMu.Lock()
	captured := append([]domain.Signal(nil), w.replaySignals...)
	w.replayMu.Unlock()
	return signalReplay.BuildReport(captured, options.From, options.To, options.FeePercent, options.Split)
}

func replayEventKey(snapshot domain.Snapshot) string {
	return seriesIdentity(snapshot.Candle.Market, snapshot.Candle.Symbol, snapshot.Candle.Timeframe) + "\x00" + snapshot.Candle.OpenTime.UTC().Format(time.RFC3339Nano)
}

func (w *SignalWorker) replaySeries(options ReplayOptions) ([]domain.SeriesKey, error) {
	var strategy definition.Strategy
	if len(w.replayDefinitions) != 1 {
		return nil, fmt.Errorf("replay requires exactly one strategy definition")
	}
	if err := json.Unmarshal(w.replayDefinitions[0].Config, &strategy); err != nil {
		return nil, err
	}
	symbols := append([]string(nil), options.Symbols...)
	if len(symbols) == 0 {
		for _, info := range w.marketInfo {
			if info.market == options.Market {
				symbols = append(symbols, info.symbol)
			}
		}
		sort.Strings(symbols)
	}
	keys := make(map[domain.SeriesKey]struct{})
	for _, subscription := range []domain.Subscription{{Market: strategy.Market, Symbols: strategy.Universe.Symbols, TriggerTimeframe: strategy.Timeframes.Trigger, ContextTimeframes: strategy.Timeframes.Context}} {
		if subscription.Market != options.Market {
			continue
		}
		for _, symbol := range symbols {
			include := len(strategy.Universe.Symbols) == 0
			for _, pattern := range strategy.Universe.Symbols {
				if matched, _ := path.Match(pattern, symbol); matched {
					include = true
					break
				}
			}
			if !include {
				continue
			}
			for _, pattern := range strategy.Universe.Exclude {
				if matched, _ := path.Match(pattern, symbol); matched {
					include = false
					break
				}
			}
			if !include {
				continue
			}
			for _, timeframe := range append([]string{strategy.Timeframes.Trigger}, strategy.Timeframes.Context...) {
				keys[domain.SeriesKey{Market: options.Market, Symbol: symbol, Timeframe: timeframe}] = struct{}{}
			}
		}
		for _, reference := range strategy.References {
			for _, timeframe := range append([]string{strategy.Timeframes.Trigger}, strategy.Timeframes.Context...) {
				keys[domain.SeriesKey{Market: options.Market, Symbol: reference.Symbol, Timeframe: timeframe}] = struct{}{}
			}
		}
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("strategy %q has no series for market %q and requested symbols", options.StrategyID, options.Market)
	}
	out := make([]domain.SeriesKey, 0, len(keys))
	for key := range keys {
		out = append(out, key)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Symbol == out[j].Symbol {
			return out[i].Timeframe < out[j].Timeframe
		}
		return out[i].Symbol < out[j].Symbol
	})
	return out, nil
}

func (w *SignalWorker) initialize(ctx context.Context) error {
	providers, err := w.providers.Active(ctx)
	if err != nil {
		return fmt.Errorf("load active providers: %w", err)
	}
	providerIDs := make(map[string]uint)
	providerAliases := make(map[uint]string)
	for _, provider := range providers {
		aliases := marketAliases(provider.Name)
		for _, alias := range aliases {
			providerIDs[alias] = provider.ID
		}
		if len(aliases) > 0 {
			providerAliases[provider.ID] = aliases[0]
		}
	}
	markets, err := w.markets.List(ctx, nil)
	if err != nil {
		return fmt.Errorf("load configured markets: %w", err)
	}
	w.marketInfo = make(map[uint]signalMarketInfo, len(markets))
	for _, market := range markets {
		alias, ok := providerAliases[market.ProviderID]
		if !ok {
			continue
		}
		symbol := market.Label
		if symbol == "" {
			symbol = market.Name
		}
		info := signalMarketInfo{market: alias, symbol: symbol, timeframes: make(map[uint]signalTimeframeInfo)}
		for _, timeframe := range market.TimeFrames {
			info.timeframes[timeframe.ID] = signalTimeframeInfo{label: timeframe.Label, duration: timeframe.Duration}
		}
		w.marketInfo[market.ID] = info
	}
	definitions, err := w.indicators.List(ctx)
	if err != nil {
		return fmt.Errorf("load indicator definitions: %w", err)
	}
	w.seriesIndicators = make(map[string][]models.IndicatorDefinition)
	for _, indicator := range definitions {
		market, ok := w.marketInfo[indicator.MarketID]
		if !ok {
			continue
		}
		tf, ok := market.timeframes[indicator.TimeframeID]
		if !ok {
			continue
		}
		key := seriesIdentity(market.market, market.symbol, tf.label)
		w.seriesIndicators[key] = append(w.seriesIndicators[key], indicator)
	}
	catalog := definition.RuntimeCatalog{Markets: w.markets, Indicators: w.indicators, ProviderIDs: providerIDs}
	validator := definition.Validator{Markets: catalog, Indicators: catalog}
	source := strings.ToLower(strings.TrimSpace(w.cfg.StrategySource))
	if source == "" {
		source = "both"
	}
	var fileRepo domain.StrategyRepository
	var dbRepo domain.StrategyRepository
	if source == "file" || source == "both" {
		fileRepo = definition.NewFileStrategyRepository(w.cfg.StrategiesDir, validator, w.cfg.PollInterval, 250*time.Millisecond)
	}
	if source == "db" || source == "both" {
		dbRepo = definition.NewDBStrategyRepository(w.strategies, validator, w.cfg.PollInterval)
	}
	switch source {
	case "file":
		w.definitions = fileRepo
	case "db":
		w.definitions = dbRepo
	case "both":
		w.definitions = &definition.CompositeRepository{File: fileRepo, Database: dbRepo, PollInterval: w.cfg.PollInterval}
	default:
		return fmt.Errorf("unknown signal strategy source %q", source)
	}
	loaded, err := w.definitions.List(ctx)
	if err != nil {
		return fmt.Errorf("load signal strategies: %w", err)
	}
	if w.replayMode {
		filtered := make([]domain.StrategyDefinition, 0, 1)
		for _, item := range loaded {
			if item.ID == w.replayStrategyID {
				filtered = append(filtered, item)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("strategy %q was not found or is invalid", w.replayStrategyID)
		}
		loaded = filtered
	}
	var positions []domain.Position
	if !w.replayMode {
		positions, err = w.signals.OpenPositions(ctx)
		if err != nil {
			return fmt.Errorf("restore open virtual positions: %w", err)
		}
	}
	manager := risk.NewManager()
	manager.Restore(positions)
	registry := definition.NewFactoryRegistry()
	if err := registry.Register(&engine.DeclarativeFactory{Validator: &validator}); err != nil {
		return err
	}
	loader := &state.DatabaseSnapshotLoader{Candles: w.candles, Markets: w.markets, ProviderID: providerIDs, Calculator: state.SnapshotCalculatorFunc(w.calculateSnapshots)}
	w.loader = loader
	marketState, err := state.NewMemoryMarketStateStore(max(w.cfg.StateCapacity, 1), loader)
	if err != nil {
		return err
	}
	profiles := make(map[string]domain.MarketProfile)
	for alias := range providerIDs {
		profiles[alias] = signalMarketProfile{}
	}
	handler := engine.SignalHandler(w.persistAndPublish)
	if w.replayMode {
		handler = w.captureReplaySignal
	}
	w.router = engine.NewRouter(marketState, registry, catalog, manager, engine.RouterConfig{FailureThreshold: w.cfg.FailureThreshold, WarmupMargin: w.cfg.WarmupMargin, Handler: handler, Profiles: profiles})
	if err := w.router.Reload(ctx, loaded); err != nil {
		return fmt.Errorf("index signal strategies: %w", err)
	}
	w.statsLoaded.Store(int64(len(loaded)))
	w.replayDefinitions = loaded
	if w.replayMode {
		return nil
	}
	if err := w.reconcilePositions(ctx, positions); err != nil {
		logrus.WithError(err).Warn("reconcile persisted positions")
	}
	if err := w.router.Warmup(ctx); err != nil {
		return err
	}
	w.publishPending(ctx)
	return nil
}

func marketAliases(provider consts.Provider) []string {
	switch provider {
	case consts.ProviderNobitex:
		return []string{"crypto", "nobitex"}
	case consts.ProviderTSE:
		return []string{"bourse", "tse"}
	default:
		value := strings.ToLower(string(provider))
		if value == "" {
			return nil
		}
		return []string{value}
	}
}

func (w *SignalWorker) calculateSnapshots(ctx context.Context, key domain.SeriesKey, candles []models.Candle) ([]domain.Snapshot, error) {
	items := make([]domain.Snapshot, len(candles))
	input := make([]indicatorsPkg.Candle, len(candles))
	for i, candle := range candles {
		open := time.Unix(candle.Time, 0).UTC()
		items[i] = domain.Snapshot{Candle: domain.Candle{Market: key.Market, Symbol: key.Symbol, Timeframe: key.Timeframe, OpenTime: open, CloseTime: open.Add(w.timeframeDuration(key)), Closed: true, Open: candle.Open, High: candle.High, Low: candle.Low, Close: candle.Close, Volume: candle.Volume}}
		input[i] = indicatorsPkg.Candle{Time: candle.Time, Open: candle.Open, High: candle.High, Low: candle.Low, Close: candle.Close, Volume: candle.Volume}
	}
	for _, definition := range w.seriesIndicators[seriesIdentity(key.Market, key.Symbol, key.Timeframe)] {
		indicator, err := CreateIndicatorFromDefinition(definition)
		if err != nil {
			logrus.WithError(err).WithField("indicator_id", definition.ID).Warn("skip unsupported warm-up indicator")
			continue
		}
		values := indicator.Calculate(ctx, input)
		fieldValues := make(map[int]map[string]decimal.Decimal)
		for _, value := range values {
			for i, candle := range candles {
				if candle.Time == value.Time && value.Value != nil {
					fieldValues[i] = value.Value.Fields()
					break
				}
			}
		}
		indicatorKey := canonicalIndicatorKey(definition)
		base := stripIndicatorPeriod(indicatorKey)
		aliases := indicatorKeyAliases(base)
		for index, fields := range fieldValues {
			for field, value := range fields {
				if len(fields) == 1 {
					items[index].Indicators = ensureIndicatorMap(items[index].Indicators)
					items[index].Indicators[indicatorKey] = value
					for _, alias := range aliases {
						items[index].Indicators[alias] = value
						if field != alias {
							items[index].Indicators[alias+"."+field] = value
						}
					}
				} else {
					items[index].Indicators = ensureIndicatorMap(items[index].Indicators)
					for _, alias := range aliases {
						items[index].Indicators[alias+"."+field] = value
					}
					items[index].Indicators[indicatorKey+"."+field] = value
					if field == base {
						for _, alias := range aliases {
							items[index].Indicators[alias] = value
						}
						items[index].Indicators[indicatorKey] = value
					}
					if base == "macd" && field == "histogram" {
						items[index].Indicators["macd.hist"] = value
					}
				}
			}
		}
	}
	return items, nil
}

func ensureIndicatorMap(value map[string]decimal.Decimal) map[string]decimal.Decimal {
	if value == nil {
		return make(map[string]decimal.Decimal)
	}
	return value
}
func seriesIdentity(market, symbol, timeframe string) string {
	return market + "\x00" + strings.ToUpper(symbol) + "\x00" + timeframe
}
func (w *SignalWorker) timeframeDuration(key domain.SeriesKey) time.Duration {
	for _, info := range w.marketInfo {
		if info.market == key.Market && strings.EqualFold(info.symbol, key.Symbol) {
			for _, tf := range info.timeframes {
				if tf.label == key.Timeframe {
					return tf.duration
				}
			}
		}
	}
	return 0
}

func (w *SignalWorker) reconcilePositions(ctx context.Context, positions []domain.Position) error {
	for _, position := range positions {
		key := domain.SeriesKey{Market: position.Market, Symbol: position.Symbol, Timeframe: position.Timeframe}
		items, err := w.loader.LoadRange(ctx, key, position.EntryTime, time.Now().UTC())
		if err != nil {
			logrus.WithError(err).WithField("signal_id", position.EntrySignalID).Warn("cannot load candles for position reconciliation")
			continue
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Candle.CloseTime.Before(items[j].Candle.CloseTime) })
		for _, item := range items {
			if !item.Candle.CloseTime.After(position.EntryTime) {
				continue
			}
			if _, err := w.router.Process(ctx, item, engine.ProcessOptions{Stale: true, Recovered: true}); err != nil {
				return err
			}
			if _, stillOpen := w.router.PositionManager().Position(position.StrategyID, position.Market, position.Symbol); !stillOpen {
				break
			}
		}
	}
	return nil
}

func (w *SignalWorker) handleSnapshot(ctx context.Context, event events.CandleSnapshot) error {
	w.stats.events.Add(1)
	info, ok := w.marketInfo[event.Candle.MarketID]
	if !ok {
		logrus.WithField("market_id", event.Candle.MarketID).Debug("ignore signal candle for unknown market")
		return nil
	}
	tf, ok := info.timeframes[event.Candle.TimeframeID]
	if !ok || tf.duration <= 0 {
		return nil
	}
	open := time.Unix(event.Candle.Time, 0).UTC()
	closeAt := open.Add(tf.duration)
	now := time.Now().UTC()
	snapshot := domain.Snapshot{Candle: domain.Candle{Market: info.market, Symbol: info.symbol, Timeframe: tf.label, OpenTime: open, CloseTime: closeAt, Closed: !closeAt.After(now), Open: event.Candle.Open, High: event.Candle.High, Low: event.Candle.Low, Close: event.Candle.Close, Volume: event.Candle.Volume}, Indicators: event.Indicators}
	if !snapshot.Candle.Closed {
		return nil
	}
	maxAge := w.cfg.MaxEventAge
	if maxAge <= 0 {
		maxAge = 2 * tf.duration
	}
	stale := now.Sub(closeAt) > maxAge
	if stale {
		w.stats.stale.Add(1)
	}
	result, err := w.router.Process(ctx, snapshot, engine.ProcessOptions{Stale: stale})
	if err != nil {
		w.stats.evalErrors.Add(1)
		return err
	}
	switch result.Status {
	case domain.ApplyDuplicate:
		w.stats.duplicate.Add(1)
	case domain.ApplyLate:
		w.stats.late.Add(1)
	}
	return nil
}

func (w *SignalWorker) persistAndPublish(ctx context.Context, signal domain.Signal) error {
	row, err := w.signals.Get(ctx, signal.SignalID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil && row.Published {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := w.signals.Save(ctx, signal); err != nil {
			return err
		}
	}
	if signal.Type == domain.SignalExit && signal.Exit != nil {
		if err := w.signals.CloseEntry(ctx, signal.Exit.EntrySignalID, signal); err != nil {
			return err
		}
	}
	if err := w.eventBus.PublishSignal(ctx, signal); err != nil {
		// The signal row is durable and remains pending. Let JetStream ack the
		// input so a transient publisher outage cannot roll back an open virtual
		// position or replay a risk event twice; the outbox retry loop republishes.
		logrus.WithError(err).WithField("signal_id", signal.SignalID).Warn("signal publish deferred to outbox retry")
		return nil
	}
	if err := w.signals.MarkPublished(ctx, signal.SignalID); err != nil {
		logrus.WithError(err).WithField("signal_id", signal.SignalID).Warn("signal publish marker deferred to outbox retry")
		return nil
	}
	w.stats.emitted.Add(1)
	return nil
}

func (w *SignalWorker) publishPending(ctx context.Context) {
	pending, err := w.signals.Pending(ctx, 100)
	if err != nil {
		logrus.WithError(err).Warn("load pending signal publications")
		return
	}
	for _, signal := range pending {
		if err := w.persistAndPublish(ctx, signal); err != nil {
			logrus.WithError(err).WithField("signal_id", signal.SignalID).Warn("retry pending signal publication")
		}
	}
}

func (w *SignalWorker) watchDefinitions(ctx context.Context, changes <-chan domain.ChangeSet) {
	defer w.done.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-changes:
			if !ok {
				return
			}
			next, err := w.definitions.List(ctx)
			if err != nil {
				logrus.WithError(err).Error("reload signal definitions")
				continue
			}
			if err = w.router.Reload(ctx, next); err != nil {
				logrus.WithError(err).Error("swap signal definitions")
				continue
			}
			w.statsLoaded.Store(int64(len(next)))
			_ = w.router.Warmup(ctx)
			logrus.WithField("strategies_loaded", len(next)).Info("signal strategies reloaded")
		}
	}
}

func (w *SignalWorker) background(ctx context.Context) {
	defer w.done.Done()
	pending := time.NewTicker(5 * time.Second)
	stats := time.NewTicker(time.Minute)
	defer pending.Stop()
	defer stats.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-pending.C:
			w.publishPending(ctx)
		case <-stats.C:
			logrus.WithFields(logrus.Fields{"events_in": w.stats.events.Load(), "duplicate": w.stats.duplicate.Load(), "late": w.stats.late.Load(), "stale": w.stats.stale.Load(), "signals_emitted": w.stats.emitted.Load(), "eval_errors": w.stats.evalErrors.Load(), "strategies_loaded": w.statsLoaded.Load()}).Info("signal worker stats")
		}
	}
}

func (w *SignalWorker) Stop() error {
	w.mu.Lock()
	cancel := w.cancel
	subscription := w.subscription
	w.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if subscription != nil {
		_ = subscription.Close()
	}
	w.done.Wait()
	logrus.Info("signal worker stopped")
	return nil
}

type signalMarketProfile struct{}

func (signalMarketProfile) Location() *time.Location                         { return time.UTC }
func (signalMarketProfile) IsTradingTime(time.Time) bool                     { return true }
func (signalMarketProfile) RoundPrice(price decimal.Decimal) decimal.Decimal { return price }

var _ io.Closer = (*events.NATSBus)(nil)
