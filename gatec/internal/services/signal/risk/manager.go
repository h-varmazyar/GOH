package risk

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/h-varmazyar/gatec/internal/services/signal/definition"
	"github.com/h-varmazyar/gatec/internal/services/signal/domain"
	"github.com/shopspring/decimal"
)

type Exit struct {
	Reason  string
	Price   decimal.Decimal
	Details domain.ExitDetails
}

// Manager owns virtual positions, limits, and exit ordering. It never places orders.
type Manager struct {
	mu           sync.Mutex
	positions    map[string]domain.Position
	cooldown     map[string]int
	dailyEntries map[string]int
}

func NewManager() *Manager {
	return &Manager{positions: make(map[string]domain.Position), cooldown: make(map[string]int), dailyEntries: make(map[string]int)}
}

func positionKey(strategy, market, symbol string) string {
	return strategy + "\x00" + market + "\x00" + symbol
}
func dayKey(strategy string, at time.Time) string {
	return strategy + "\x00" + at.UTC().Format("2006-01-02")
}

func (m *Manager) Positions() []domain.Position {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.Position, 0, len(m.positions))
	for _, position := range m.positions {
		out = append(out, position)
	}
	return out
}

func (m *Manager) Position(strategy, market, symbol string) (domain.Position, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	position, ok := m.positions[positionKey(strategy, market, symbol)]
	return position, ok
}

func (m *Manager) Restore(positions []domain.Position) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, position := range positions {
		m.positions[positionKey(position.StrategyID, position.Market, position.Symbol)] = position
	}
}

func (m *Manager) CanEnter(strategy, market, symbol string, at time.Time, limits definition.Limits) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	currentDay := dayKey(strategy, at)
	for key := range m.dailyEntries {
		if strings.HasPrefix(key, strategy+"\x00") && key != currentDay {
			delete(m.dailyEntries, key)
		}
	}
	key := positionKey(strategy, market, symbol)
	if _, open := m.positions[key]; open || m.cooldown[key] > 0 {
		return false
	}
	if limits.MaxOpenPositions > 0 {
		count := 0
		for _, position := range m.positions {
			if position.StrategyID == strategy {
				count++
			}
		}
		if count >= limits.MaxOpenPositions {
			return false
		}
	}
	if limits.MaxSignalsPerDay > 0 && m.dailyEntries[currentDay] >= limits.MaxSignalsPerDay {
		return false
	}
	return true
}

func (m *Manager) Open(entrySignalID string, strategy definition.Strategy, snapshot domain.Snapshot, direction domain.Direction, profile domain.MarketProfile) (domain.Position, *decimal.Decimal, *decimal.Decimal, *decimal.Decimal, error) {
	candle := snapshot.Candle
	if strategy.Risk == nil || strategy.Risk.StopLoss == nil {
		return domain.Position{}, nil, nil, nil, fmt.Errorf("risk.stop_loss is required")
	}
	if candle.Close.IsZero() {
		return domain.Position{}, nil, nil, nil, fmt.Errorf("entry price must be non-zero")
	}
	if direction != domain.DirectionLong && direction != domain.DirectionShort {
		return domain.Position{}, nil, nil, nil, fmt.Errorf("unsupported position direction %q", direction)
	}
	stopDistance, stopPct, err := distance(snapshot, strategy.Risk.StopLoss)
	if err != nil {
		return domain.Position{}, nil, nil, nil, fmt.Errorf("stop loss: %w", err)
	}
	var takeDistance, rewardPct decimal.Decimal
	if strategy.Risk.TakeProfit != nil {
		takeDistance, rewardPct, err = distance(snapshot, strategy.Risk.TakeProfit)
		if err != nil {
			return domain.Position{}, nil, nil, nil, fmt.Errorf("take profit: %w", err)
		}
	}
	entry := candle.Close
	stop, take := decimal.Zero, decimal.Zero
	if direction == domain.DirectionLong {
		stop = entry.Sub(stopDistance)
		if strategy.Risk.TakeProfit != nil {
			take = entry.Add(takeDistance)
		}
	} else {
		stop = entry.Add(stopDistance)
		if strategy.Risk.TakeProfit != nil {
			take = entry.Sub(takeDistance)
		}
	}
	if profile != nil {
		entry = profile.RoundPrice(entry)
		stop = profile.RoundPrice(stop)
		if !take.IsZero() {
			take = profile.RoundPrice(take)
		}
	}
	key := positionKey(strategy.ID, candle.Market, candle.Symbol)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.positions[key]; exists {
		return domain.Position{}, nil, nil, nil, fmt.Errorf("position already open")
	}
	if m.cooldown[key] > 0 {
		return domain.Position{}, nil, nil, nil, fmt.Errorf("position is cooling down")
	}
	strategyConfig, _ := json.Marshal(strategy)
	m.positions[key] = domain.Position{EntrySignalID: entrySignalID, StrategyID: strategy.ID, StrategyVersion: strategy.Version, StrategyMode: domain.Mode(strategy.Mode), StrategyConfig: strategyConfig, Market: candle.Market, Symbol: candle.Symbol, Timeframe: candle.Timeframe, Direction: direction, EntryTime: candle.CloseTime, EntryPrice: entry, StopLoss: stop, TakeProfit: take}
	m.dailyEntries[dayKey(strategy.ID, candle.CloseTime)]++
	riskReward := decimal.Zero
	if !rewardPct.IsZero() {
		riskReward = rewardPct.Div(stopPct)
	}
	return m.positions[key], ptr(stopPct), decimalPtr(rewardPct), decimalPtr(riskReward), nil
}

func distance(snapshot domain.Snapshot, rule *definition.RiskRule) (decimal.Decimal, decimal.Decimal, error) {
	candle := snapshot.Candle
	if rule == nil || rule.Value <= 0 {
		return decimal.Zero, decimal.Zero, fmt.Errorf("positive risk rule is required")
	}
	value := decimal.NewFromFloat(rule.Value)
	pct := decimal.Zero
	switch rule.Type {
	case "percent":
		pct = value
	case "atr_multiple":
		atr, ok := indicatorValue(snapshot, "atr")
		if !ok {
			return decimal.Zero, decimal.Zero, fmt.Errorf("ATR value is missing")
		}
		pct = atr.Mul(value).Div(candle.Close).Mul(decimal.NewFromInt(100))
	default:
		return decimal.Zero, decimal.Zero, fmt.Errorf("unsupported rule type %q", rule.Type)
	}
	minPct, maxPct := decimal.NewFromFloat(rule.MinPct), decimal.NewFromFloat(rule.MaxPct)
	if rule.MinPct > 0 && pct.LessThan(minPct) {
		pct = minPct
	}
	if rule.MaxPct > 0 && pct.GreaterThan(maxPct) {
		pct = maxPct
	}
	if !pct.GreaterThan(decimal.Zero) {
		return decimal.Zero, decimal.Zero, fmt.Errorf("risk distance must be positive")
	}
	return candle.Close.Mul(pct).Div(decimal.NewFromInt(100)), pct, nil
}

func indicatorValue(snapshot domain.Snapshot, prefix string) (decimal.Decimal, bool) {
	if value, ok := snapshot.Indicators[prefix]; ok {
		return value, true
	}
	keys := make([]string, 0, len(snapshot.Indicators))
	for key := range snapshot.Indicators {
		if strings.HasPrefix(key, prefix+"_") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return decimal.Zero, false
	}
	return snapshot.Indicators[keys[0]], true
}

func (m *Manager) Manage(strategyID string, strategy definition.Strategy, snapshot domain.Snapshot, decision domain.Decision) (*Exit, bool) {
	if !snapshot.Candle.Closed {
		return nil, false
	}
	key := positionKey(strategyID, snapshot.Candle.Market, snapshot.Candle.Symbol)
	m.mu.Lock()
	defer m.mu.Unlock()
	position, exists := m.positions[key]
	if !exists {
		return nil, false
	}
	if !snapshot.Candle.CloseTime.After(position.EntryTime) {
		return nil, false
	}
	if !position.LastCandleOpen.Equal(snapshot.Candle.OpenTime) {
		position.BarsHeld++
		position.LastCandleOpen = snapshot.Candle.OpenTime
	}
	stopHit, takeHit := false, false
	if position.Direction == domain.DirectionLong {
		stopHit = snapshot.Candle.Low.LessThanOrEqual(position.StopLoss)
		takeHit = !position.TakeProfit.IsZero() && snapshot.Candle.High.GreaterThanOrEqual(position.TakeProfit)
	} else {
		stopHit = snapshot.Candle.High.GreaterThanOrEqual(position.StopLoss)
		takeHit = !position.TakeProfit.IsZero() && snapshot.Candle.Low.LessThanOrEqual(position.TakeProfit)
	}
	price, reason := decimal.Zero, ""
	if stopHit {
		price, reason = position.StopLoss, "stop_loss"
	} else if takeHit {
		price, reason = position.TakeProfit, "take_profit"
	} else if strategy.Risk != nil && strategy.Risk.TimeStop != nil && position.BarsHeld >= strategy.Risk.TimeStop.MaxBars {
		price, reason = snapshot.Candle.Close, "time_stop"
	} else if decision.Kind == domain.DecisionExit {
		price, reason = snapshot.Candle.Close, "rule"
	}
	if reason == "" {
		m.positions[key] = position
		return nil, false
	}
	m.positions[key] = position
	pnl := price.Sub(position.EntryPrice).Div(position.EntryPrice).Mul(decimal.NewFromInt(100))
	if position.Direction == domain.DirectionShort {
		pnl = position.EntryPrice.Sub(price).Div(position.EntryPrice).Mul(decimal.NewFromInt(100))
	}
	return &Exit{Reason: reason, Price: price, Details: domain.ExitDetails{Reason: reason, EntrySignalID: position.EntrySignalID, PnLPercent: pnl, BarsHeld: position.BarsHeld}}, true
}

func (m *Manager) Close(strategyID, market, symbol string, cooldownBars int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.positions, positionKey(strategyID, market, symbol))
	if cooldownBars > 0 {
		m.cooldown[positionKey(strategyID, market, symbol)] = cooldownBars + 1
	}
}

func (m *Manager) AdvanceCooldown(market, symbol string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key, remaining := range m.cooldown {
		if strings.HasSuffix(key, "\x00"+market+"\x00"+symbol) {
			if remaining <= 1 {
				delete(m.cooldown, key)
			} else {
				m.cooldown[key] = remaining - 1
			}
		}
	}
}

func ptr(value decimal.Decimal) *decimal.Decimal { return &value }
func decimalPtr(value decimal.Decimal) *decimal.Decimal {
	if value.IsZero() {
		return nil
	}
	return &value
}
