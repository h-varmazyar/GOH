package state

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/internal/services/signal/domain"
)

type SnapshotCalculator interface {
	CalculateSnapshots(context.Context, domain.SeriesKey, []models.Candle) ([]domain.Snapshot, error)
}

type SnapshotCalculatorFunc func(context.Context, domain.SeriesKey, []models.Candle) ([]domain.Snapshot, error)

func (f SnapshotCalculatorFunc) CalculateSnapshots(ctx context.Context, key domain.SeriesKey, candles []models.Candle) ([]domain.Snapshot, error) {
	return f(ctx, key, candles)
}

// DatabaseSnapshotLoader reads stored candles and delegates indicator rebuilds
// to the existing indicator package through SnapshotCalculator.
type DatabaseSnapshotLoader struct {
	Candles    *repositories.CandleRepository
	Markets    *repositories.MarketRepository
	ProviderID map[string]uint
	Calculator SnapshotCalculator
	Now        func() time.Time
}

func (l *DatabaseSnapshotLoader) LoadRecent(ctx context.Context, key domain.SeriesKey, count int) ([]domain.Snapshot, error) {
	marketID, timeframeID, duration, err := l.resolve(ctx, key)
	if err != nil {
		return nil, err
	}
	candles, err := l.Candles.List(ctx, marketID, timeframeID, count)
	if err != nil {
		return nil, err
	}
	return l.snapshots(ctx, key, candles, duration)
}

func (l *DatabaseSnapshotLoader) LoadRange(ctx context.Context, key domain.SeriesKey, from, to time.Time) ([]domain.Snapshot, error) {
	marketID, timeframeID, duration, err := l.resolve(ctx, key)
	if err != nil {
		return nil, err
	}
	candles, err := l.Candles.Range(ctx, marketID, timeframeID, from, to)
	if err != nil {
		return nil, err
	}
	return l.snapshots(ctx, key, candles, duration)
}

func (l *DatabaseSnapshotLoader) resolve(ctx context.Context, key domain.SeriesKey) (uint, uint, time.Duration, error) {
	providerID, ok := l.ProviderID[key.Market]
	if !ok {
		return 0, 0, 0, fmt.Errorf("no provider configured for market %q", key.Market)
	}
	markets, err := l.Markets.List(ctx, map[string]any{"provider_id = ?": providerID})
	if err != nil {
		return 0, 0, 0, err
	}
	var marketID, timeframeID uint
	var duration time.Duration
	for _, market := range markets {
		if !strings.EqualFold(market.Label, key.Symbol) && !strings.EqualFold(market.Name, key.Symbol) {
			continue
		}
		marketID = market.ID
		for _, timeframe := range market.TimeFrames {
			if timeframe.Label == key.Timeframe {
				timeframeID, duration = timeframe.ID, timeframe.Duration
				break
			}
		}
		break
	}
	if marketID == 0 {
		return 0, 0, 0, fmt.Errorf("market %q has no configured symbol %q", key.Market, key.Symbol)
	}
	if timeframeID == 0 || duration <= 0 {
		return 0, 0, 0, fmt.Errorf("timeframe %q is not configured for %s/%s", key.Timeframe, key.Market, key.Symbol)
	}
	return marketID, timeframeID, duration, nil
}

func (l *DatabaseSnapshotLoader) snapshots(ctx context.Context, key domain.SeriesKey, candles []models.Candle, duration time.Duration) ([]domain.Snapshot, error) {
	now := time.Now().UTC()
	if l.Now != nil {
		now = l.Now().UTC()
	}
	filtered := make([]models.Candle, 0, len(candles))
	for _, candle := range candles {
		if !time.Unix(candle.Time, 0).UTC().Add(duration).After(now) {
			filtered = append(filtered, candle)
		}
	}
	if l.Calculator != nil {
		return l.Calculator.CalculateSnapshots(ctx, key, filtered)
	}
	out := make([]domain.Snapshot, 0, len(filtered))
	for _, candle := range filtered {
		open := time.Unix(candle.Time, 0).UTC()
		out = append(out, domain.Snapshot{Candle: domain.Candle{Market: key.Market, Symbol: key.Symbol, Timeframe: key.Timeframe, OpenTime: open, CloseTime: open.Add(duration), Closed: true, Open: candle.Open, High: candle.High, Low: candle.Low, Close: candle.Close, Volume: candle.Volume}})
	}
	return out, nil
}

var _ SnapshotLoader = (*DatabaseSnapshotLoader)(nil)
