// Package domain defines the strategy engine's market-neutral contracts and values.
package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// Mode controls where a generated signal is published.
type Mode string

const (
	ModeShadow Mode = "shadow"
	ModeLive   Mode = "live"
)

// Direction is the side of a virtual position.
type Direction string

const (
	DirectionLong  Direction = "long"
	DirectionShort Direction = "short"
)

// DecisionKind is the action returned by a strategy evaluation.
type DecisionKind string

const (
	DecisionNone  DecisionKind = "NONE"
	DecisionEntry DecisionKind = "ENTRY"
	DecisionExit  DecisionKind = "EXIT"
)

// SignalType identifies the lifecycle event represented by a signal.
type SignalType string

const (
	SignalEntry  SignalType = "ENTRY"
	SignalExit   SignalType = "EXIT"
	SignalUpdate SignalType = "UPDATE"
)

// Candle is a normalized OHLCV observation. Times are explicit so callers can
// distinguish a candle's open from its close and reject unfinished candles.
type Candle struct {
	Market    string
	Symbol    string
	Timeframe string
	OpenTime  time.Time
	CloseTime time.Time
	Closed    bool
	Open      decimal.Decimal
	High      decimal.Decimal
	Low       decimal.Decimal
	Close     decimal.Decimal
	Volume    decimal.Decimal
}

// Snapshot combines a normalized candle with its calculated indicator values.
// Indicator keys include any needed output or parameter suffix.
type Snapshot struct {
	Candle     Candle
	Indicators map[string]decimal.Decimal
}

// SeriesKey uniquely identifies a market data series.
type SeriesKey struct {
	Market    string
	Symbol    string
	Timeframe string
}

// Reference binds a strategy-local alias to a reference series.
type Reference struct {
	Alias  string
	Market string
	Symbol string
}

// Subscription describes the series and timeframes a strategy consumes.
type Subscription struct {
	Market            string
	Symbols           []string
	TriggerTimeframe  string
	ContextTimeframes []string
	References        []Reference
}

// IndicatorRequirement declares an indicator key required by a strategy.
type IndicatorRequirement struct {
	Market    string
	Timeframe string
	Key       string
}

// StrategyDefinition is the common envelope for file and database definitions.
// Config remains opaque to the engine so definition formats can evolve separately.
type StrategyDefinition struct {
	SchemaVersion int             `json:"schema_version"`
	ID            string          `json:"id"`
	Version       int             `json:"version"`
	Enabled       bool            `json:"enabled"`
	Mode          Mode            `json:"mode"`
	Type          string          `json:"type"`
	Market        string          `json:"market"`
	Config        json.RawMessage `json:"config,omitempty"`
}

// Reason contains explainable evaluation evidence.
type Reason struct {
	Matched []string                   `json:"matched,omitempty"`
	Values  map[string]decimal.Decimal `json:"values,omitempty"`
	Text    string                     `json:"text,omitempty"`
}

// Decision is the result of evaluating one strategy against a snapshot.
type Decision struct {
	Kind       DecisionKind
	Direction  Direction
	StopLoss   *decimal.Decimal
	TakeProfit *decimal.Decimal
	Score      *float64
	Reason     Reason
}

// EvalContext provides the current and historical observations available to a
// strategy, together with context-series access, position state, and a clock.
type EvalContext struct {
	Context  context.Context
	Current  Snapshot
	History  []Snapshot
	Contexts SnapshotReader
	Position *Position
	Clock    Clock
}

// Position is an in-memory representation of an open virtual position.
type Position struct {
	EntrySignalID   string
	StrategyID      string
	StrategyVersion int
	StrategyMode    Mode
	StrategyConfig  json.RawMessage
	Market          string
	Symbol          string
	Timeframe       string
	Direction       Direction
	EntryTime       time.Time
	EntryPrice      decimal.Decimal
	StopLoss        decimal.Decimal
	TakeProfit      decimal.Decimal
	BarsHeld        int
	LastCandleOpen  time.Time
}

// ExitDetails records the outcome attached to an exit signal.
type ExitDetails struct {
	Reason        string          `json:"reason"`
	EntrySignalID string          `json:"entry_signal_id"`
	PnLPercent    decimal.Decimal `json:"pnl_pct"`
	BarsHeld      int             `json:"bars_held"`
	Recovered     bool            `json:"recovered"`
}

// Signal is the transport-neutral signal value persisted and published by the
// worker. Serialization and storage behavior are implemented in later steps.
type Signal struct {
	SchemaVersion   int              `json:"schema_version"`
	SignalID        string           `json:"signal_id"`
	StrategyID      string           `json:"strategy_id"`
	StrategyVersion int              `json:"strategy_version"`
	StrategyConfig  json.RawMessage  `json:"strategy_config,omitempty"`
	Mode            Mode             `json:"mode"`
	Market          string           `json:"market"`
	Symbol          string           `json:"symbol"`
	Timeframe       string           `json:"timeframe"`
	Type            SignalType       `json:"type"`
	Direction       Direction        `json:"direction"`
	CandleOpenTime  time.Time        `json:"candle_open_time"`
	CandleCloseTime time.Time        `json:"candle_close_time"`
	EmittedAt       time.Time        `json:"emitted_at"`
	Price           decimal.Decimal  `json:"price"`
	StopLoss        *decimal.Decimal `json:"stop_loss,omitempty"`
	TakeProfit      *decimal.Decimal `json:"take_profit,omitempty"`
	RiskPercent     *decimal.Decimal `json:"risk_pct,omitempty"`
	RewardPercent   *decimal.Decimal `json:"reward_pct,omitempty"`
	RiskReward      *decimal.Decimal `json:"risk_reward,omitempty"`
	ValidUntil      *time.Time       `json:"valid_until,omitempty"`
	Score           *float64         `json:"score,omitempty"`
	Reason          Reason           `json:"reason"`
	Exit            *ExitDetails     `json:"exit,omitempty"`
}

// ApplyStatus classifies a snapshot submitted to the market state store.
type ApplyStatus string

const (
	ApplyNew       ApplyStatus = "new"
	ApplyDuplicate ApplyStatus = "duplicate"
	ApplyLate      ApplyStatus = "late"
	ApplyGap       ApplyStatus = "gap"
	ApplyUnclosed  ApplyStatus = "unclosed"
	ApplyInactive  ApplyStatus = "inactive"
)

// ApplyResult describes how a market state store accepted a snapshot.
type ApplyResult struct {
	Status      ApplyStatus
	Key         SeriesKey
	Dirty       bool
	Recovered   bool
	MissingFrom time.Time
	MissingTo   time.Time
}

// ChangeSet groups strategy definition changes observed by a repository.
type ChangeSet struct {
	Added   []StrategyDefinition
	Updated []StrategyDefinition
	Removed []string
}
