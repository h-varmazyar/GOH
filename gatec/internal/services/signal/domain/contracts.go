package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// Strategy evaluates normalized snapshots without depending on a concrete
// market or strategy implementation.
type Strategy interface {
	ID() string
	Version() int
	Mode() Mode
	Subscriptions() []Subscription
	Requirements() []IndicatorRequirement
	Evaluate(EvalContext) (Decision, error)
}

// StrategyFactory builds strategies registered under a type name.
type StrategyFactory interface {
	Type() string
	Build(StrategyDefinition) (Strategy, error)
}

// StrategyRepository loads definitions and reports changes until its context ends.
type StrategyRepository interface {
	List(context.Context) ([]StrategyDefinition, error)
	Watch(context.Context) (<-chan ChangeSet, error)
}

// IndicatorCatalog answers whether an indicator is configured and how much
// history it needs before a value is ready.
type IndicatorCatalog interface {
	Has(context.Context, string, string, string) (bool, error)
	Warmup(context.Context, string, string, string) (int, error)
}

// MarketStateStore maintains bounded, ordered snapshots for each data series.
type MarketStateStore interface {
	Apply(context.Context, Snapshot) (ApplyResult, error)
	Window(context.Context, SeriesKey, int) ([]Snapshot, error)
	LatestClosedAtOrBefore(context.Context, SeriesKey, time.Time) (Snapshot, bool, error)
	Ready(SeriesKey) bool
}

// SnapshotReader exposes historical and context snapshots to strategy evaluation.
type SnapshotReader interface {
	Window(context.Context, SeriesKey, int) ([]Snapshot, error)
	LatestClosedAtOrBefore(context.Context, SeriesKey, time.Time) (Snapshot, bool, error)
}

// SignalPublisher publishes a signal to its configured transport.
type SignalPublisher interface {
	Publish(context.Context, Signal) error
}

// SignalStore persists signals and virtual position lifecycle changes.
type SignalStore interface {
	Save(context.Context, Signal) error
	MarkPublished(context.Context, string) error
	OpenPositions(context.Context) ([]Position, error)
	CloseEntry(context.Context, string, Signal) error
}

// MarketProfile provides market-specific calendar and price rounding behavior
// behind a generic interface.
type MarketProfile interface {
	Location() *time.Location
	IsTradingTime(time.Time) bool
	RoundPrice(decimal.Decimal) decimal.Decimal
}

// Clock abstracts wall-clock and replay time.
type Clock interface {
	Now() time.Time
}
