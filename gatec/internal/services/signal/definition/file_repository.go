package definition

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/h-varmazyar/gatec/internal/services/signal/domain"
	"github.com/sirupsen/logrus"
)

// FileRepository loads YAML definitions and polls for filesystem changes. Polling
// provides a debounce window without adding a new runtime dependency.
type FileRepository struct {
	Dir          string
	Validator    Validator
	PollInterval time.Duration
	Debounce     time.Duration
	mu           sync.Mutex
	current      map[string]domain.StrategyDefinition
	sources      map[string]string
}

// FileStrategyRepository is the strategy-domain name for FileRepository.
type FileStrategyRepository = FileRepository

func NewFileStrategyRepository(dir string, validator Validator, poll, debounce time.Duration) *FileStrategyRepository {
	return NewFileRepository(dir, validator, poll, debounce)
}

func NewFileRepository(dir string, validator Validator, poll, debounce time.Duration) *FileRepository {
	if poll <= 0 {
		poll = time.Second
	}
	if debounce <= 0 {
		debounce = 250 * time.Millisecond
	}
	return &FileRepository{Dir: dir, Validator: validator, PollInterval: poll, Debounce: debounce, current: make(map[string]domain.StrategyDefinition), sources: make(map[string]string)}
}

func (r *FileRepository) List(ctx context.Context) ([]domain.StrategyDefinition, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	next, sources, err := r.read(ctx, r.current, r.sources)
	if err != nil {
		return nil, err
	}
	r.current = next
	r.sources = sources
	return sortedDefinitions(next), nil
}

func (r *FileRepository) read(ctx context.Context, previous map[string]domain.StrategyDefinition, previousSources map[string]string) (map[string]domain.StrategyDefinition, map[string]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(r.Dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		extension := strings.ToLower(filepath.Ext(path))
		if extension == ".yaml" || extension == ".yml" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, err
	}
	sort.Strings(files)
	next := make(map[string]domain.StrategyDefinition, len(previous))
	for k, v := range previous {
		next[k] = v
	}
	present := make(map[string]struct{}, len(files))
	for _, path := range files {
		present[path] = struct{}{}
	}
	nextSources := make(map[string]string, len(previousSources))
	for p, id := range previousSources {
		nextSources[p] = id
	}
	for p, id := range previousSources {
		if _, ok := present[p]; !ok {
			delete(next, id)
			delete(nextSources, p)
		}
	}
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			logrus.WithError(err).WithField("path", path).Error("read strategy definition")
			continue
		}
		s, err := ParseYAML(raw)
		if err != nil {
			logrus.WithError(err).WithField("path", path).Error("invalid strategy definition")
			continue
		}
		if err = r.Validator.Validate(ctx, s); err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{"path": path, "strategy_id": s.ID}).Error("invalid strategy definition")
			continue
		}
		d, err := s.Domain()
		if err != nil {
			logrus.WithError(err).WithField("path", path).Error("encode strategy definition")
			continue
		}
		if oldID, exists := nextSources[path]; exists && oldID != s.ID {
			delete(next, oldID)
		}
		if _, exists := next[s.ID]; exists {
			if source, wasPrevious := nextSources[path]; !wasPrevious && source == "" {
				logrus.WithField("strategy_id", s.ID).Warn("duplicate strategy id in YAML files; later file ignored")
				continue
			}
		}
		next[s.ID] = d
		nextSources[path] = s.ID
	}
	// Keep an invalid changed file's last known-good value, but remove strategies
	// whose source files were deleted.
	return next, nextSources, nil
}

func (r *FileRepository) Watch(ctx context.Context) (<-chan domain.ChangeSet, error) {
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
				timer := time.NewTimer(r.Debounce)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
				now, err := r.List(ctx)
				if err != nil {
					logrus.WithError(err).Error("poll strategy files")
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

func sortedDefinitions(m map[string]domain.StrategyDefinition) []domain.StrategyDefinition {
	out := make([]domain.StrategyDefinition, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
