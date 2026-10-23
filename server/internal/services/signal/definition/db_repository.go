package definition

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/internal/services/signal/domain"
	"github.com/sirupsen/logrus"
)

type DBRepository struct {
	Rows         *repositories.StrategyDefinitionRepository
	Validator    Validator
	PollInterval time.Duration
	mu           sync.Mutex
	current      map[string]domain.StrategyDefinition
}

// DBStrategyRepository loads and polls the persisted strategy definitions.
type DBStrategyRepository = DBRepository

func NewDBStrategyRepository(rows *repositories.StrategyDefinitionRepository, validator Validator, interval time.Duration) *DBStrategyRepository {
	return NewDBRepository(rows, validator, interval)
}

func NewDBRepository(rows *repositories.StrategyDefinitionRepository, validator Validator, interval time.Duration) *DBRepository {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &DBRepository{Rows: rows, Validator: validator, PollInterval: interval, current: make(map[string]domain.StrategyDefinition)}
}
func (r *DBRepository) List(ctx context.Context) ([]domain.StrategyDefinition, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	next, err := r.load(ctx)
	if err != nil {
		return nil, err
	}
	r.current = next
	return sortedDefinitions(next), nil
}
func (r *DBRepository) load(ctx context.Context) (map[string]domain.StrategyDefinition, error) {
	rows, err := r.Rows.List(ctx)
	if err != nil {
		return nil, err
	}
	next := make(map[string]domain.StrategyDefinition, len(rows))
	invalid := make(map[string]bool)
	for _, row := range rows {
		if !row.Enabled {
			definition, parseErr := ParseJSON(row.Definition)
			if parseErr == nil && definition.ID == row.ID && definition.Market == row.Market {
				definition.Enabled = false
				if loaded, domainErr := definition.Domain(); domainErr == nil {
					next[row.ID] = loaded
					continue
				}
			}
			next[row.ID] = domain.StrategyDefinition{ID: row.ID, Version: row.Version, Enabled: false, Market: row.Market, Config: append([]byte(nil), row.Definition...)}
			continue
		}
		s, err := ParseJSON(row.Definition)
		if err == nil && s.ID != row.ID {
			err = fmt.Errorf("definition.id %q does not match row id %q", s.ID, row.ID)
		}
		if err == nil && s.Version != row.Version {
			err = fmt.Errorf("definition.version %d does not match row version %d", s.Version, row.Version)
		}
		if err == nil && s.Enabled != row.Enabled {
			err = fmt.Errorf("definition.enabled does not match row enabled")
		}
		if err == nil && s.Market != row.Market {
			err = fmt.Errorf("definition.market %q does not match row market %q", s.Market, row.Market)
		}
		if err == nil {
			err = r.Validator.Validate(ctx, s)
		}
		if err != nil {
			invalid[row.ID] = true
			logrus.WithError(err).WithField("strategy_id", row.ID).Error("invalid strategy definition in database")
			continue
		}
		d, err := s.Domain()
		if err != nil {
			invalid[row.ID] = true
			logrus.WithError(err).WithField("strategy_id", row.ID).Error("encode strategy definition")
			continue
		}
		next[row.ID] = d
	}
	// Invalid updates retain the last known-good version.
	for id, old := range r.current {
		if invalid[id] {
			next[id] = old
		}
	}
	// A new invalid DB row must still shadow a same-ID file definition. It is
	// represented as disabled until a valid database version is written.
	for _, row := range rows {
		if invalid[row.ID] {
			if _, hasLastGood := next[row.ID]; !hasLastGood {
				next[row.ID] = domain.StrategyDefinition{ID: row.ID, Version: row.Version, Enabled: false, Market: row.Market, Config: append([]byte(nil), row.Definition...)}
			}
		}
	}
	return next, nil
}
func (r *DBRepository) Watch(ctx context.Context) (<-chan domain.ChangeSet, error) {
	ch := make(chan domain.ChangeSet, 1)
	ticker := time.NewTicker(r.PollInterval)
	go func() {
		defer ticker.Stop()
		defer close(ch)
		before, _ := r.List(ctx)
		prev := definitionMap(before)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now, err := r.List(ctx)
				if err != nil {
					logrus.WithError(err).Error("poll strategy database")
					continue
				}
				next := definitionMap(now)
				changes := diffDefinitions(prev, next)
				if len(changes.Added)+len(changes.Updated)+len(changes.Removed) > 0 {
					select {
					case ch <- changes:
						prev = next
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return ch, nil
}

type CompositeRepository struct {
	File         domain.StrategyRepository
	Database     domain.StrategyRepository
	PollInterval time.Duration
	mu           sync.Mutex
	warned       map[string]struct{}
}

type CompositeStrategyRepository = CompositeRepository

func (r *CompositeRepository) List(ctx context.Context) ([]domain.StrategyDefinition, error) {
	files, err := r.File.List(ctx)
	if err != nil {
		return nil, err
	}
	db, err := r.Database.List(ctx)
	if err != nil {
		return nil, err
	}
	out := definitionMap(files)
	for _, d := range db {
		if _, duplicate := out[d.ID]; duplicate {
			r.mu.Lock()
			if r.warned == nil {
				r.warned = make(map[string]struct{})
			}
			_, alreadyWarned := r.warned[d.ID]
			if !alreadyWarned {
				r.warned[d.ID] = struct{}{}
			}
			r.mu.Unlock()
			if !alreadyWarned {
				logrus.WithField("strategy_id", d.ID).Warn("strategy exists in file and database; database definition takes precedence")
			}
		}
		out[d.ID] = d
	}
	return sortedDefinitions(out), nil
}
func (r *CompositeRepository) Watch(ctx context.Context) (<-chan domain.ChangeSet, error) {
	interval := r.PollInterval
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	ch := make(chan domain.ChangeSet, 1)
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		defer close(ch)
		before, err := r.List(ctx)
		if err != nil {
			logrus.WithError(err).Error("load strategy definitions")
			return
		}
		prev := definitionMap(before)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now, err := r.List(ctx)
				if err != nil {
					logrus.WithError(err).Error("reload strategy definitions")
					continue
				}
				next := definitionMap(now)
				changes := diffDefinitions(prev, next)
				if len(changes.Added)+len(changes.Updated)+len(changes.Removed) > 0 {
					select {
					case ch <- changes:
						prev = next
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return ch, nil
}

func definitionMap(ds []domain.StrategyDefinition) map[string]domain.StrategyDefinition {
	m := make(map[string]domain.StrategyDefinition, len(ds))
	for _, d := range ds {
		m[d.ID] = d
	}
	return m
}
func diffDefinitions(before, after map[string]domain.StrategyDefinition) domain.ChangeSet {
	c := domain.ChangeSet{}
	for id, n := range after {
		o, ok := before[id]
		if !ok {
			c.Added = append(c.Added, n)
		} else if o.Version != n.Version || o.Enabled != n.Enabled || o.Market != n.Market || o.Type != n.Type || !jsonEqual(o.Config, n.Config) {
			c.Updated = append(c.Updated, n)
		}
	}
	for id := range before {
		if _, ok := after[id]; !ok {
			c.Removed = append(c.Removed, id)
		}
	}
	sort.Slice(c.Added, func(i, j int) bool { return c.Added[i].ID < c.Added[j].ID })
	sort.Slice(c.Updated, func(i, j int) bool { return c.Updated[i].ID < c.Updated[j].ID })
	sort.Strings(c.Removed)
	return c
}
func jsonEqual(a, b []byte) bool { return string(a) == string(b) }
