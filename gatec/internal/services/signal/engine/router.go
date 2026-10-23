package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/h-varmazyar/gatec/internal/services/signal/definition"
	"github.com/h-varmazyar/gatec/internal/services/signal/domain"
	"github.com/h-varmazyar/gatec/internal/services/signal/risk"
	"github.com/h-varmazyar/gatec/internal/services/signal/signals"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

type SignalHandler func(context.Context, domain.Signal) error

type RouterConfig struct {
	FailureThreshold int
	WarmupMargin     int
	Handler          SignalHandler
	Profiles         map[string]domain.MarketProfile
	Clock            domain.Clock
}

type routeEntry struct {
	definition definition.Strategy
	strategy   domain.Strategy
	config     domain.StrategyDefinition
	capacity   int
}

type routeTable struct {
	bySeries   map[string][]*routeEntry
	all        []*routeEntry
	capacities map[domain.SeriesKey]int
}

type Router struct {
	state     domain.MarketStateStore
	registry  *definition.FactoryRegistry
	manager   *risk.Manager
	catalog   domain.IndicatorCatalog
	config    RouterConfig
	table     atomic.Pointer[routeTable]
	locks     sync.Map
	failureMu sync.Mutex
	failures  map[string]int
	disabled  map[string]bool
}

func (r *Router) PositionManager() *risk.Manager      { return r.manager }
func (r *Router) StateStore() domain.MarketStateStore { return r.state }

func NewRouter(state domain.MarketStateStore, registry *definition.FactoryRegistry, catalog domain.IndicatorCatalog, positions *risk.Manager, cfg RouterConfig) *Router {
	if cfg.FailureThreshold < 1 {
		cfg.FailureThreshold = 3
	}
	if cfg.WarmupMargin < 0 {
		cfg.WarmupMargin = 0
	}
	if cfg.Clock == nil {
		cfg.Clock = wallClock{}
	}
	if positions == nil {
		positions = risk.NewManager()
	}
	router := &Router{state: state, registry: registry, catalog: catalog, manager: positions, config: cfg, failures: make(map[string]int), disabled: make(map[string]bool)}
	router.table.Store(&routeTable{bySeries: make(map[string][]*routeEntry), capacities: make(map[domain.SeriesKey]int)})
	return router
}

type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now() }

func (r *Router) Reload(ctx context.Context, definitions []domain.StrategyDefinition) error {
	previous := r.table.Load()
	known := make(map[string]struct{}, len(definitions))
	for _, value := range definitions {
		known[value.ID] = struct{}{}
	}
	for _, position := range r.manager.Positions() {
		if _, ok := known[position.StrategyID]; ok || len(position.StrategyConfig) == 0 {
			continue
		}
		var saved definition.Strategy
		if err := json.Unmarshal(position.StrategyConfig, &saved); err != nil {
			logrus.WithError(err).WithField("strategy", position.StrategyID).Error("cannot restore deleted strategy configuration")
			continue
		}
		saved.Enabled = false
		envelope, err := saved.Domain()
		if err != nil {
			continue
		}
		definitions = append(definitions, envelope)
	}
	next := &routeTable{bySeries: make(map[string][]*routeEntry), capacities: make(map[domain.SeriesKey]int)}
	byID := make(map[string]*routeEntry)
	for _, envelope := range definitions {
		if envelope.Type == "" || (!envelope.Enabled && !openStrategy(r.manager, envelope.ID)) {
			continue
		}
		factory, ok := r.registry.Get(envelope.Type)
		if !ok {
			logrus.WithFields(logrus.Fields{"strategy": envelope.ID, "type": envelope.Type}).Warn("strategy factory is not registered")
			continue
		}
		strategy, err := factory.Build(envelope)
		if err != nil {
			logrus.WithError(err).WithField("strategy", envelope.ID).Error("strategy definition rejected")
			continue
		}
		var cfg definition.Strategy
		if err := json.Unmarshal(envelope.Config, &cfg); err != nil {
			logrus.WithError(err).WithField("strategy", envelope.ID).Error("strategy config decode failed")
			continue
		}
		entry := &routeEntry{definition: cfg, strategy: strategy, config: envelope, capacity: r.capacity(ctx, strategy)}
		byID[envelope.ID] = entry
	}
	openIDs := make(map[string]bool)
	for _, position := range r.manager.Positions() {
		openIDs[position.StrategyID] = true
	}
	if previous != nil {
		for _, old := range previous.all {
			if openIDs[old.strategy.ID()] {
				if replacement, exists := byID[old.strategy.ID()]; !exists || !replacement.definition.Enabled {
					draining := *old
					draining.definition.Enabled = false
					draining.config.Enabled = false
					byID[old.strategy.ID()] = &draining
				}
			}
		}
	}
	for _, entry := range byID {
		next.all = append(next.all, entry)
	}
	sort.Slice(next.all, func(i, j int) bool { return next.all[i].strategy.ID() < next.all[j].strategy.ID() })
	for _, entry := range next.all {
		for _, sub := range entry.strategy.Subscriptions() {
			keys := append([]string{sub.TriggerTimeframe}, sub.ContextTimeframes...)
			for _, symbol := range sub.Symbols {
				if symbol == "" || strings.ContainsAny(symbol, "*?[") {
					continue
				}
				for _, timeframe := range keys {
					series := domain.SeriesKey{Market: sub.Market, Symbol: symbol, Timeframe: timeframe}
					next.capacities[series] = max(next.capacities[series], entry.capacity)
				}
			}
			for _, reference := range sub.References {
				for _, timeframe := range keys {
					series := domain.SeriesKey{Market: reference.Market, Symbol: reference.Symbol, Timeframe: timeframe}
					next.capacities[series] = max(next.capacities[series], entry.capacity)
				}
			}
			indexKey := sub.Market + "\x00" + sub.TriggerTimeframe
			next.bySeries[indexKey] = append(next.bySeries[indexKey], entry)
		}
	}
	if state, ok := r.state.(interface {
		ReconcileActiveMatching(map[domain.SeriesKey]int, func(domain.SeriesKey) (int, bool)) error
	}); ok {
		if err := state.ReconcileActiveMatching(next.capacities, func(key domain.SeriesKey) (int, bool) { return r.activeCapacity(next, key) }); err != nil {
			return err
		}
	} else if state, ok := r.state.(interface {
		ReconcileActive(map[domain.SeriesKey]int) error
	}); ok {
		if err := state.ReconcileActive(next.capacities); err != nil {
			return err
		}
	}
	r.table.Store(next)
	r.failureMu.Lock()
	r.failures = make(map[string]int)
	r.disabled = make(map[string]bool)
	r.failureMu.Unlock()
	return nil
}

func openStrategy(manager *risk.Manager, id string) bool {
	for _, position := range manager.Positions() {
		if position.StrategyID == id {
			return true
		}
	}
	return false
}

func (r *Router) Warmup(ctx context.Context) error {
	warmer, ok := r.state.(interface {
		Warmup(context.Context, domain.SeriesKey, int) error
	})
	if !ok {
		return fmt.Errorf("market state does not support warm-up")
	}
	for key, count := range r.table.Load().capacities {
		if err := warmer.Warmup(ctx, key, count); err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{"market": key.Market, "symbol": key.Symbol, "timeframe": key.Timeframe}).Warn("signal market state warm-up failed; series will wait for live candles")
		}
	}
	return nil
}

func (r *Router) capacity(ctx context.Context, strategy domain.Strategy) int {
	want := 2 + r.config.WarmupMargin
	if r.catalog != nil {
		for _, requirement := range strategy.Requirements() {
			if count, err := r.catalog.Warmup(ctx, requirement.Market, requirement.Timeframe, requirement.Key); err == nil && count > want {
				want = count
			}
		}
	}
	return want
}

type ProcessOptions struct {
	Stale           bool
	Recovered       bool
	NextOpen        *decimal.Decimal
	SlippagePercent decimal.Decimal
	NoEntry         bool
}

func (r *Router) SetClock(clock domain.Clock) {
	if clock != nil {
		r.config.Clock = clock
	}
}
func (r *Router) SetSignalHandler(handler SignalHandler) { r.config.Handler = handler }
func (r *Router) CapacityFor(key domain.SeriesKey) int {
	table := r.table.Load()
	if table == nil {
		return 0
	}
	if value, ok := table.capacities[key]; ok {
		return value
	}
	if value, ok := r.activeCapacity(table, key); ok {
		return value
	}
	return 0
}

func (r *Router) Process(ctx context.Context, snapshot domain.Snapshot, options ProcessOptions) (domain.ApplyResult, error) {
	key := domain.SeriesKey{Market: snapshot.Candle.Market, Symbol: snapshot.Candle.Symbol, Timeframe: snapshot.Candle.Timeframe}
	table := r.table.Load()
	entries := r.matching(table, key, snapshot.Candle.Symbol)
	capacity, relevant := r.activeCapacity(table, key)
	if !relevant {
		return domain.ApplyResult{Status: domain.ApplyInactive, Key: key}, nil
	}
	if settable, ok := r.state.(interface {
		SetCapacity(domain.SeriesKey, int) error
	}); ok {
		if err := settable.SetCapacity(key, capacity); err != nil {
			return domain.ApplyResult{}, err
		}
	}
	lockValue, _ := r.locks.LoadOrStore(key.Market+"\x00"+key.Symbol, &sync.Mutex{})
	seriesLock := lockValue.(*sync.Mutex)
	seriesLock.Lock()
	defer seriesLock.Unlock()
	result, err := r.state.Apply(ctx, snapshot)
	if err != nil {
		return result, err
	}
	if result.Status != domain.ApplyNew && result.Status != domain.ApplyGap && result.Status != domain.ApplyDuplicate {
		return result, nil
	}
	if !snapshot.Candle.Closed {
		return result, nil
	}
	if options.Stale && !options.Recovered {
		return result, nil
	}
	if len(entries) == 0 {
		return result, nil
	}
	if result.Status != domain.ApplyDuplicate {
		r.manager.AdvanceCooldown(key.Market, key.Symbol)
	}
	for _, entry := range entries {
		r.processStrategy(ctx, entry, snapshot, options)
	}
	return result, nil
}

func (r *Router) activeCapacity(table *routeTable, key domain.SeriesKey) (int, bool) {
	if table == nil {
		return 0, false
	}
	capacity := 0
	active := false
	for _, entry := range table.all {
		for _, sub := range entry.strategy.Subscriptions() {
			covered := sub.Market == key.Market && matchesUniverse(key.Symbol, entry.definition.Universe) && (sub.TriggerTimeframe == key.Timeframe || contains(sub.ContextTimeframes, key.Timeframe))
			if !covered {
				for _, reference := range sub.References {
					if reference.Market == key.Market && strings.EqualFold(reference.Symbol, key.Symbol) && (sub.TriggerTimeframe == key.Timeframe || contains(sub.ContextTimeframes, key.Timeframe)) {
						covered = true
						break
					}
				}
			}
			if covered {
				active = true
				capacity = max(capacity, entry.capacity)
			}
		}
	}
	return capacity, active
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func (r *Router) matching(table *routeTable, key domain.SeriesKey, symbol string) []*routeEntry {
	if table == nil {
		return nil
	}
	entries := table.bySeries[key.Market+"\x00"+key.Timeframe]
	out := make([]*routeEntry, 0, len(entries))
	for _, entry := range entries {
		for _, sub := range entry.strategy.Subscriptions() {
			if sub.Market != key.Market || sub.TriggerTimeframe != key.Timeframe {
				continue
			}
			if matchesUniverse(symbol, entry.definition.Universe) {
				out = append(out, entry)
			}
		}
	}
	return out
}

func matchesUniverse(symbol string, universe definition.Universe) bool {
	include := len(universe.Symbols) == 0
	for _, pattern := range universe.Symbols {
		if matched, _ := path.Match(pattern, symbol); matched {
			include = true
			break
		}
	}
	if !include {
		return false
	}
	for _, pattern := range universe.Exclude {
		if matched, _ := path.Match(pattern, symbol); matched {
			return false
		}
	}
	return true
}

func (r *Router) processStrategy(ctx context.Context, entry *routeEntry, snapshot domain.Snapshot, options ProcessOptions) {
	strategyID := entry.strategy.ID()
	if recovered := recoverStrategy(func() { r.evaluate(ctx, entry, snapshot, options) }); recovered != nil {
		r.failure(strategyID, fmt.Errorf("strategy panic: %v", recovered))
	}
}

func recoverStrategy(call func()) (panicValue any) {
	defer func() { panicValue = recover() }()
	call()
	return nil
}

func (r *Router) evaluate(ctx context.Context, entry *routeEntry, snapshot domain.Snapshot, options ProcessOptions) {
	candle := snapshot.Candle
	position, hasPosition := r.manager.Position(entry.strategy.ID(), candle.Market, candle.Symbol)
	ready := r.state.Ready(domain.SeriesKey{Market: candle.Market, Symbol: candle.Symbol, Timeframe: candle.Timeframe})
	if !ready && !hasPosition {
		return
	}
	window, err := r.state.Window(ctx, domain.SeriesKey{Market: candle.Market, Symbol: candle.Symbol, Timeframe: candle.Timeframe}, 0)
	if err != nil {
		r.failure(entry.strategy.ID(), err)
		return
	}
	evalContext := domain.EvalContext{Context: ctx, Current: snapshot, History: window, Contexts: r.snapshotReader(), Clock: r.config.Clock}
	if hasPosition {
		evalContext.Position = &position
	}
	decision := domain.Decision{Kind: domain.DecisionNone}
	strategyDisabled := r.isDisabled(entry.strategy.ID())
	if ready && !strategyDisabled {
		decision, err = entry.strategy.Evaluate(evalContext)
		if err != nil {
			r.failure(entry.strategy.ID(), err)
			if !hasPosition {
				return
			}
		} else {
			r.success(entry.strategy.ID())
		}
	}
	if hasPosition {
		exit, shouldClose := r.manager.Manage(entry.strategy.ID(), entry.definition, snapshot, decision)
		if !shouldClose || exit == nil {
			return
		}
		signal := domain.Signal{SchemaVersion: 1, SignalID: signals.DeterministicID(entry.strategy.ID(), candle.Market, candle.Symbol, candle.Timeframe, candle.OpenTime, domain.SignalExit), StrategyID: entry.strategy.ID(), StrategyVersion: entry.strategy.Version(), Mode: entry.strategy.Mode(), Market: candle.Market, Symbol: candle.Symbol, Timeframe: candle.Timeframe, Type: domain.SignalExit, Direction: position.Direction, CandleOpenTime: candle.OpenTime, CandleCloseTime: candle.CloseTime, EmittedAt: r.config.Clock.Now(), Price: exit.Price, Reason: decision.Reason, Exit: &exit.Details}
		signal.Exit.Recovered = options.Recovered
		if err := r.emit(ctx, signal); err != nil {
			r.failure(entry.strategy.ID(), err)
			return
		}
		r.manager.Close(entry.strategy.ID(), candle.Market, candle.Symbol, entry.definition.Limits.CooldownBars)
		return
	}
	if options.Stale || options.NoEntry || !entry.definition.Enabled || decision.Kind != domain.DecisionEntry {
		return
	}
	if !r.manager.CanEnter(entry.strategy.ID(), candle.Market, candle.Symbol, candle.CloseTime, entry.definition.Limits) {
		return
	}
	id := signals.DeterministicID(entry.strategy.ID(), candle.Market, candle.Symbol, candle.Timeframe, candle.OpenTime, domain.SignalEntry)
	entrySnapshot := snapshot
	if options.NextOpen != nil && !options.NextOpen.IsZero() {
		entry := *options.NextOpen
		if options.SlippagePercent.GreaterThan(decimal.Zero) {
			slippage := options.SlippagePercent.Div(decimal.NewFromInt(100))
			if decision.Direction == domain.DirectionShort {
				entry = entry.Mul(decimal.NewFromInt(1).Sub(slippage))
			} else {
				entry = entry.Mul(decimal.NewFromInt(1).Add(slippage))
			}
		}
		entrySnapshot.Candle.Close = entry
	}
	created, stopPct, rewardPct, riskReward, err := r.manager.Open(id, entry.definition, entrySnapshot, decision.Direction, r.config.Profiles[candle.Market])
	if err != nil {
		r.failure(entry.strategy.ID(), err)
		return
	}
	signal := domain.Signal{SchemaVersion: 1, SignalID: id, StrategyID: entry.strategy.ID(), StrategyVersion: entry.strategy.Version(), StrategyConfig: append([]byte(nil), created.StrategyConfig...), Mode: entry.strategy.Mode(), Market: candle.Market, Symbol: candle.Symbol, Timeframe: candle.Timeframe, Type: domain.SignalEntry, Direction: created.Direction, CandleOpenTime: candle.OpenTime, CandleCloseTime: candle.CloseTime, EmittedAt: r.config.Clock.Now(), Price: created.EntryPrice, StopLoss: &created.StopLoss, TakeProfit: decimalPointer(created.TakeProfit), RiskPercent: stopPct, RewardPercent: rewardPct, RiskReward: riskReward, Score: decision.Score, Reason: decision.Reason}
	if entry.definition.Risk != nil && entry.definition.Risk.TimeStop != nil {
		valid := candle.CloseTime.Add(time.Duration(entry.definition.Risk.TimeStop.MaxBars) * timeframeApprox(candle.Timeframe))
		signal.ValidUntil = &valid
	}
	if err := r.emit(ctx, signal); err != nil {
		r.manager.Close(entry.strategy.ID(), candle.Market, candle.Symbol, 0)
		r.failure(entry.strategy.ID(), err)
	}
}

type latestReader struct{ store domain.MarketStateStore }

func (l latestReader) Window(ctx context.Context, key domain.SeriesKey, n int) ([]domain.Snapshot, error) {
	return l.store.Window(ctx, key, n)
}
func (l latestReader) LatestClosedAtOrBefore(ctx context.Context, key domain.SeriesKey, at time.Time) (domain.Snapshot, bool, error) {
	return l.store.LatestClosedAtOrBefore(ctx, key, at)
}
func (r *Router) snapshotReader() domain.SnapshotReader { return latestReader{store: r.state} }

func (r *Router) emit(ctx context.Context, signal domain.Signal) error {
	if r.config.Handler == nil {
		return fmt.Errorf("signal handler is not configured")
	}
	return r.config.Handler(ctx, signal)
}
func (r *Router) isDisabled(id string) bool {
	r.failureMu.Lock()
	defer r.failureMu.Unlock()
	return r.disabled[id]
}
func (r *Router) failure(id string, err error) {
	r.failureMu.Lock()
	r.failures[id]++
	count := r.failures[id]
	if count >= r.config.FailureThreshold {
		r.disabled[id] = true
	}
	r.failureMu.Unlock()
	logrus.WithError(err).WithFields(logrus.Fields{"strategy": id, "consecutive_errors": count}).Error("signal strategy evaluation failed")
}
func (r *Router) success(id string) { r.failureMu.Lock(); delete(r.failures, id); r.failureMu.Unlock() }

func timeframeApprox(value string) time.Duration {
	if len(value) < 2 {
		return time.Hour
	}
	n, err := time.ParseDuration(value[:len(value)-1] + map[byte]string{'m': "m", 'h': "h", 'd': "h"}[value[len(value)-1]])
	if err != nil {
		return time.Hour
	}
	if value[len(value)-1] == 'd' {
		return n * 24
	}
	return n
}

func decimalPointer(value decimal.Decimal) *decimal.Decimal {
	if value.IsZero() {
		return nil
	}
	return &value
}
