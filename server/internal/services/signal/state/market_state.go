package state

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/h-varmazyar/gatec/internal/services/signal/domain"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// SnapshotLoader is implemented by the application layer that can read stored
// candles and calculate indicator values when the database has no snapshots.
type SnapshotLoader interface {
	LoadRecent(context.Context, domain.SeriesKey, int) ([]domain.Snapshot, error)
	LoadRange(context.Context, domain.SeriesKey, time.Time, time.Time) ([]domain.Snapshot, error)
}

// GapPolicy lets a market calendar account for closed sessions and holidays.
type GapPolicy interface {
	NextOpen(domain.SeriesKey, time.Time) (time.Time, bool)
}

// IntervalGapPolicy assumes a continuous fixed-duration market series.
type IntervalGapPolicy struct{}

func (IntervalGapPolicy) NextOpen(key domain.SeriesKey, lastOpen time.Time) (time.Time, bool) {
	interval, ok := timeframeDuration(key.Timeframe)
	if !ok {
		return time.Time{}, false
	}
	return lastOpen.Add(interval), true
}

type seriesState struct {
	mu       sync.RWMutex
	values   []domain.Snapshot
	head     int
	size     int
	capacity int
	dirtyAt  time.Time
}

func (s *seriesState) oldestIndex() int { return (s.head - s.size + s.capacity) % s.capacity }
func (s *seriesState) at(index int) domain.Snapshot {
	return s.values[(s.oldestIndex()+index)%s.capacity]
}
func (s *seriesState) append(snapshot domain.Snapshot) {
	s.values[s.head] = cloneSnapshot(snapshot)
	s.head = (s.head + 1) % s.capacity
	if s.size < s.capacity {
		s.size++
	}
	if !s.dirtyAt.IsZero() && s.size > 0 && s.at(0).Candle.OpenTime.After(s.dirtyAt) {
		s.dirtyAt = time.Time{}
	}
}

// MemoryMarketStateStore keeps a bounded chronological ring for active series.
type MemoryMarketStateStore struct {
	mu         sync.RWMutex
	series     map[domain.SeriesKey]*seriesState
	active     map[domain.SeriesKey]int
	activeOnly bool
	capacity   int
	loader     SnapshotLoader
	gapPolicy  GapPolicy
}

func NewMemoryMarketStateStore(capacity int, loader SnapshotLoader, policies ...GapPolicy) (*MemoryMarketStateStore, error) {
	if capacity < 1 {
		return nil, fmt.Errorf("market state capacity must be positive")
	}
	policy := GapPolicy(IntervalGapPolicy{})
	if len(policies) > 0 && policies[0] != nil {
		policy = policies[0]
	}
	return &MemoryMarketStateStore{series: make(map[domain.SeriesKey]*seriesState), active: make(map[domain.SeriesKey]int), capacity: capacity, loader: loader, gapPolicy: policy}, nil
}

// ReconcileActive sets the only series eligible for storage and each series'
// maximum lookback. Removing a series drops its buffer immediately.
func (s *MemoryMarketStateStore) ReconcileActive(capacities map[domain.SeriesKey]int) error {
	return s.ReconcileActiveMatching(capacities, func(key domain.SeriesKey) (int, bool) { capacity, ok := capacities[key]; return capacity, ok })
}

// ReconcileActiveMatching preserves series selected by dynamic symbol patterns
// and removes every series no longer covered by an active subscription.
func (s *MemoryMarketStateStore) ReconcileActiveMatching(capacities map[domain.SeriesKey]int, matcher func(domain.SeriesKey) (int, bool)) error {
	for key, capacity := range capacities {
		if capacity < 1 {
			return fmt.Errorf("capacity for %+v must be positive", key)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeOnly = true
	s.active = make(map[domain.SeriesKey]int, len(capacities)+len(s.series))
	for key, capacity := range capacities {
		s.active[key] = capacity
		if current := s.series[key]; current != nil {
			current.mu.Lock()
			if current.capacity != capacity {
				current.resize(capacity)
			}
			current.mu.Unlock()
		}
	}
	for key := range s.series {
		if capacity, ok := matcher(key); ok && capacity > 0 {
			s.active[key] = capacity
			if current := s.series[key]; current != nil {
				current.mu.Lock()
				if current.capacity != capacity {
					current.resize(capacity)
				}
				current.mu.Unlock()
			}
			continue
		}
		if _, ok := s.active[key]; !ok {
			delete(s.series, key)
		}
	}
	return nil
}

func (s *MemoryMarketStateStore) SetCapacity(key domain.SeriesKey, capacity int) error {
	if capacity < 1 {
		return fmt.Errorf("market state capacity must be positive")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active[key] = capacity
	if current := s.series[key]; current != nil {
		current.mu.Lock()
		current.resize(capacity)
		current.mu.Unlock()
	}
	return nil
}

func (s *seriesState) resize(capacity int) {
	keep := s.size
	if keep > capacity {
		keep = capacity
	}
	next := make([]domain.Snapshot, capacity)
	start := s.size - keep
	for i := 0; i < keep; i++ {
		next[i] = s.at(start + i)
	}
	s.values = next
	s.capacity = capacity
	s.size = keep
	s.head = keep % capacity
}

func (s *MemoryMarketStateStore) get(key domain.SeriesKey) (*seriesState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	capacity := s.capacity
	if requested, ok := s.active[key]; ok {
		capacity = requested
	} else if s.activeOnly {
		return nil, false
	}
	series := s.series[key]
	if series == nil {
		series = &seriesState{values: make([]domain.Snapshot, capacity), capacity: capacity}
		s.series[key] = series
	}
	return series, true
}

func (s *MemoryMarketStateStore) Apply(ctx context.Context, snapshot domain.Snapshot) (domain.ApplyResult, error) {
	key := seriesKey(snapshot.Candle)
	result := domain.ApplyResult{Key: key}
	if !snapshot.Candle.Closed {
		result.Status = domain.ApplyUnclosed
		return result, nil
	}
	if snapshot.Candle.OpenTime.IsZero() || snapshot.Candle.CloseTime.IsZero() || !snapshot.Candle.CloseTime.After(snapshot.Candle.OpenTime) {
		return result, fmt.Errorf("closed snapshot requires valid open and close times")
	}
	series, ok := s.get(key)
	if !ok {
		result.Status = domain.ApplyInactive
		return result, nil
	}
	series.mu.Lock()
	defer series.mu.Unlock()
	if series.size > 0 {
		last := series.at(series.size - 1)
		for i := 0; i < series.size; i++ {
			if series.at(i).Candle.OpenTime.Equal(snapshot.Candle.OpenTime) {
				result.Status = domain.ApplyDuplicate
				result.Dirty = !series.dirtyAt.IsZero()
				return result, nil
			}
		}
		if snapshot.Candle.OpenTime.Before(last.Candle.OpenTime) {
			result.Status = domain.ApplyLate
			result.Dirty = !series.dirtyAt.IsZero()
			return result, nil
		}
		expected, valid := s.gapPolicy.NextOpen(key, last.Candle.OpenTime)
		if valid {
			_, intervalKnown := timeframeDuration(key.Timeframe)
			if !intervalKnown {
				return result, fmt.Errorf("cannot validate recovered range for timeframe %q", key.Timeframe)
			}
			if snapshot.Candle.OpenTime.Before(expected) {
				result.Status = domain.ApplyLate
				result.Dirty = !series.dirtyAt.IsZero()
				return result, nil
			}
			if snapshot.Candle.OpenTime.After(expected) {
				result.Status = domain.ApplyGap
				result.MissingFrom = expected
				cursor := expected
				for cursor.Before(snapshot.Candle.OpenTime) {
					if err := ctx.Err(); err != nil {
						return result, err
					}
					next, ok := s.gapPolicy.NextOpen(key, cursor)
					if !ok || !next.After(cursor) {
						return result, fmt.Errorf("gap policy did not advance timeframe %q", key.Timeframe)
					}
					if !next.Before(snapshot.Candle.OpenTime) {
						result.MissingTo = cursor
						break
					}
					cursor = next
				}
				if series.dirtyAt.IsZero() {
					series.dirtyAt = expected
				}
				if s.loader != nil {
					missing, err := s.loader.LoadRange(ctx, key, result.MissingFrom, result.MissingTo)
					if err == nil && completeRange(missing, key, result.MissingFrom, result.MissingTo, s.gapPolicy) {
						for _, item := range missing {
							series.append(item)
						}
						if series.dirtyAt.Equal(result.MissingFrom) {
							series.dirtyAt = time.Time{}
						}
						result.Recovered = true
					} else if err != nil {
						logrus.WithError(err).WithFields(logrus.Fields{"market": key.Market, "symbol": key.Symbol, "timeframe": key.Timeframe}).Warn("market state gap recovery failed; series remains dirty")
					}
				}
				series.append(snapshot)
				result.Dirty = !series.dirtyAt.IsZero()
				return result, nil
			}
		}
	}
	result.Status = domain.ApplyNew
	series.append(snapshot)
	result.Dirty = !series.dirtyAt.IsZero()
	return result, nil
}

func (s *MemoryMarketStateStore) Window(ctx context.Context, key domain.SeriesKey, count int) ([]domain.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	series := s.find(key)
	if series == nil {
		return []domain.Snapshot{}, nil
	}
	series.mu.RLock()
	defer series.mu.RUnlock()
	n := series.size
	if count > 0 && count < n {
		n = count
	}
	out := make([]domain.Snapshot, n)
	for i := 0; i < n; i++ {
		out[i] = cloneSnapshot(series.at(series.size - n + i))
	}
	return out, nil
}

func (s *MemoryMarketStateStore) LatestClosedAtOrBefore(ctx context.Context, key domain.SeriesKey, at time.Time) (domain.Snapshot, bool, error) {
	if err := ctx.Err(); err != nil {
		return domain.Snapshot{}, false, err
	}
	series := s.find(key)
	if series == nil {
		return domain.Snapshot{}, false, nil
	}
	series.mu.RLock()
	defer series.mu.RUnlock()
	for i := series.size - 1; i >= 0; i-- {
		snapshot := series.at(i)
		if snapshot.Candle.Closed && !snapshot.Candle.CloseTime.After(at) {
			return cloneSnapshot(snapshot), true, nil
		}
	}
	return domain.Snapshot{}, false, nil
}

func (s *MemoryMarketStateStore) Ready(key domain.SeriesKey) bool {
	series := s.find(key)
	if series == nil {
		return false
	}
	series.mu.RLock()
	defer series.mu.RUnlock()
	return series.size >= series.capacity && series.dirtyAt.IsZero()
}

func (s *MemoryMarketStateStore) Warmup(ctx context.Context, key domain.SeriesKey, count int) error {
	if count < 1 {
		return fmt.Errorf("warm-up count must be positive")
	}
	if s.loader == nil {
		return fmt.Errorf("market state snapshot loader is not configured")
	}
	items, err := s.loader.LoadRecent(ctx, key, count)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Candle.OpenTime.Before(items[j].Candle.OpenTime) })
	for _, item := range items {
		if !item.Candle.Closed {
			return fmt.Errorf("warm-up loader returned an unclosed candle")
		}
		result, err := s.Apply(ctx, item)
		if err != nil {
			return err
		}
		if result.Status == domain.ApplyInactive {
			return fmt.Errorf("cannot warm up inactive series %+v", key)
		}
	}
	return nil
}

func (s *MemoryMarketStateStore) find(key domain.SeriesKey) *seriesState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.series[key]
}
func seriesKey(c domain.Candle) domain.SeriesKey {
	return domain.SeriesKey{Market: c.Market, Symbol: c.Symbol, Timeframe: c.Timeframe}
}
func cloneSnapshot(snapshot domain.Snapshot) domain.Snapshot {
	if snapshot.Indicators != nil {
		copyValues := make(map[string]decimal.Decimal, len(snapshot.Indicators))
		for k, v := range snapshot.Indicators {
			copyValues[k] = v
		}
		snapshot.Indicators = copyValues
	}
	return snapshot
}

func timeframeDuration(tf string) (time.Duration, bool) {
	if len(tf) < 2 {
		return 0, false
	}
	suffix := tf[len(tf)-1]
	value, err := strconv.Atoi(tf[:len(tf)-1])
	if err != nil || value < 1 {
		return 0, false
	}
	switch suffix {
	case 'm':
		return time.Duration(value) * time.Minute, true
	case 'h':
		return time.Duration(value) * time.Hour, true
	case 'd':
		return time.Duration(value) * 24 * time.Hour, true
	}
	return 0, false
}
func completeRange(items []domain.Snapshot, key domain.SeriesKey, from, to time.Time, policy GapPolicy) bool {
	if !from.After(to) {
		if len(items) == 0 {
			return false
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Candle.OpenTime.Before(items[j].Candle.OpenTime) })
		expected := from
		for _, item := range items {
			if seriesKey(item.Candle) != key || !item.Candle.Closed || !item.Candle.OpenTime.Equal(expected) || !item.Candle.CloseTime.After(item.Candle.OpenTime) {
				return false
			}
			var ok bool
			expected, ok = policy.NextOpen(key, item.Candle.OpenTime)
			if !ok || !expected.After(item.Candle.OpenTime) {
				return false
			}
		}
		return expected.After(to)
	}
	return true
}

var _ domain.MarketStateStore = (*MemoryMarketStateStore)(nil)
var _ domain.SnapshotReader = (*MemoryMarketStateStore)(nil)
