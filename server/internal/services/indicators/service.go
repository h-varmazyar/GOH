// Package indicators manages the indicator definitions of TSE markets: which
// indicators of pkg/indicators are calculated, with which params, in which
// timeframes and for which markets.
//
// A definition belongs to the TSE provider and to one timeframe that candles
// are built in. Its scope is every market of the provider (market ID 0, no
// market IDs), a group of markets (market ID 0 and the sorted market IDs) or
// one market (its market ID). The fingerprint of a definition is the one the
// indicator calculator uses, extended with the markets of a group, so the
// same indicator cannot be added twice.
package indicators

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	candleService "github.com/h-varmazyar/gatec/internal/services/candles"
	"github.com/h-varmazyar/gatec/pkg/paginator"
	"gorm.io/gorm"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrConflict     = errors.New("already exists")
)

// Scope is the markets a definition is calculated for.
type Scope string

const (
	ScopeAll    Scope = repositories.IndicatorScopeAll
	ScopeGroup  Scope = repositories.IndicatorScopeGroup
	ScopeMarket Scope = repositories.IndicatorScopeMarket
)

type Service struct {
	definitionRepo *repositories.IndicatorDefinitionRepository
	tseMarketRepo  *repositories.TSEMarketRepository
	candles        *candleService.Service
}

func NewService(
	definitionRepo *repositories.IndicatorDefinitionRepository,
	tseMarketRepo *repositories.TSEMarketRepository,
	candles *candleService.Service,
) *Service {
	return &Service{definitionRepo: definitionRepo, tseMarketRepo: tseMarketRepo, candles: candles}
}

// MarketRef is a market of a definition.
type MarketRef struct {
	ID    uint   `json:"id"`
	Label string `json:"label"`
}

// Item is a definition with the names the panel shows.
type Item struct {
	ID          uint
	Type        string
	Definition  *Definition
	Params      map[string]any
	TimeframeID uint
	// Timeframe is empty when the timeframe is not one candles are built in.
	Timeframe string
	Scope     Scope
	// Markets are the market of a MARKET definition or the markets of a
	// GROUP definition; empty for ALL.
	Markets   []MarketRef
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Timeframes returns the TSE timeframes indicators can be calculated in.
func (s *Service) Timeframes(ctx context.Context) ([]models.Timeframe, error) {
	provider, err := s.candles.TSEProvider(ctx)
	if err != nil {
		return nil, s.translate(err)
	}
	timeframes, err := s.candles.EnsureTimeframes(ctx, provider.ID)
	return timeframes, s.translate(err)
}

// MarketFilter selects the markets an indicator can be defined on.
type MarketFilter struct {
	Query             string
	IndustryGroupCode string
	IncludeDelisted   bool
}

// Markets returns a page of the TSE markets, one row per market.
func (s *Service) Markets(ctx context.Context, filter MarketFilter, pg paginator.Params) ([]repositories.TSEMarketSummary, int64, error) {
	provider, err := s.candles.TSEProvider(ctx)
	if err != nil {
		return nil, 0, s.translate(err)
	}
	return s.tseMarketRepo.MarketSummaries(ctx, repositories.TSEMarketSummaryFilter{
		ProviderID: provider.ID, Query: filter.Query, IndustryGroupCode: filter.IndustryGroupCode,
		IncludeDelisted: filter.IncludeDelisted,
	}, pg)
}

// Filter selects definitions in the panel; empty fields match everything.
type Filter struct {
	// Query matches the type, name and title of the indicator.
	Query       string
	Category    Category
	Type        string
	TimeframeID uint
	// MarketID selects the definitions of a market and of the groups that include it.
	MarketID uint
	Scope    Scope
	Enabled  *bool
}

func (s *Service) List(ctx context.Context, filter Filter, pg paginator.Params) ([]Item, int64, error) {
	if filter.Scope != "" && !validScope(filter.Scope) {
		return nil, 0, fmt.Errorf("%w: scope must be ALL, GROUP or MARKET", ErrInvalidInput)
	}
	provider, err := s.candles.TSEProvider(ctx)
	if err != nil {
		return nil, 0, s.translate(err)
	}
	repoFilter := repositories.IndicatorDefinitionFilter{
		ProviderID: provider.ID, TimeframeID: filter.TimeframeID, MarketID: filter.MarketID,
		Scope: string(filter.Scope), Enabled: filter.Enabled,
	}
	if filter.Query != "" || filter.Category != "" || filter.Type != "" {
		repoFilter.Types = matchingTypes(filter)
	}
	definitions, total, err := s.definitionRepo.ListFiltered(ctx, repoFilter, pg)
	if err != nil {
		return nil, 0, err
	}
	items, err := s.toItems(ctx, provider.ID, definitions)
	return items, total, err
}

func validScope(scope Scope) bool {
	return scope == ScopeAll || scope == ScopeGroup || scope == ScopeMarket
}

func matchingTypes(filter Filter) []string {
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	types := make([]string, 0)
	for _, definition := range catalog {
		if filter.Category != "" && definition.Category != filter.Category {
			continue
		}
		if filter.Type != "" && !strings.EqualFold(definition.Type, filter.Type) {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(definition.Type+" "+definition.Name+" "+definition.Title), query) {
			continue
		}
		types = append(types, definition.Type)
	}
	return types
}

func (s *Service) Get(ctx context.Context, id uint) (*Item, error) {
	provider, err := s.candles.TSEProvider(ctx)
	if err != nil {
		return nil, s.translate(err)
	}
	definition, err := s.get(ctx, provider.ID, id)
	if err != nil {
		return nil, err
	}
	items, err := s.toItems(ctx, provider.ID, []models.IndicatorDefinition{*definition})
	if err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (s *Service) get(ctx context.Context, providerID, id uint) (*models.IndicatorDefinition, error) {
	definition, err := s.definitionRepo.Return(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && definition.ProviderID != providerID) {
		return nil, fmt.Errorf("%w: indicator %d", ErrNotFound, id)
	}
	return definition, err
}

// MarketSelection is the markets of a definition.
type MarketSelection struct {
	Scope Scope
	// MarketIDs are empty for ALL, one market for MARKET and the markets of GROUP.
	MarketIDs []uint
}

// CreateRequest adds an indicator in one or more timeframes.
type CreateRequest struct {
	Type         string
	Params       map[string]any
	TimeframeIDs []uint
	Markets      MarketSelection
}

// Create adds a definition per timeframe. Nothing is added when one of them
// exists already.
func (s *Service) Create(ctx context.Context, req CreateRequest) ([]Item, error) {
	catalogEntry, ok := Lookup(req.Type)
	if !ok {
		return nil, fmt.Errorf("%w: unknown indicator type %q", ErrInvalidInput, req.Type)
	}
	params, err := catalogEntry.NormalizeParams(req.Params)
	if err != nil {
		return nil, err
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	provider, err := s.candles.TSEProvider(ctx)
	if err != nil {
		return nil, s.translate(err)
	}
	marketID, marketIDs, err := s.resolveMarkets(ctx, provider.ID, req.Markets)
	if err != nil {
		return nil, err
	}
	timeframes, err := s.selectTimeframes(ctx, req.TimeframeIDs)
	if err != nil {
		return nil, err
	}

	definitions := make([]*models.IndicatorDefinition, len(timeframes))
	fingerprints := make([]string, len(timeframes))
	for i, timeframe := range timeframes {
		fingerprints[i] = fingerprint(catalogEntry.Type, marketID, marketIDs, provider.ID, timeframe.ID, paramsJSON)
		definitions[i] = &models.IndicatorDefinition{
			MarketID: marketID, MarketIDs: marketIDs, ProviderID: provider.ID, TimeframeID: timeframe.ID,
			Type: catalogEntry.Type, Params: paramsJSON, Fingerprint: fingerprints[i], Enabled: true,
		}
	}
	existing, err := s.definitionRepo.ExistingFingerprints(ctx, fingerprints, 0)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		labels := make([]string, 0, len(existing))
		for i, timeframe := range timeframes {
			if slices.Contains(existing, fingerprints[i]) {
				labels = append(labels, timeframe.Label)
			}
		}
		return nil, fmt.Errorf("%w: %s with these params and markets is already added in %s", ErrConflict, catalogEntry.Name, strings.Join(labels, ", "))
	}
	if err := s.definitionRepo.CreateMany(ctx, definitions); err != nil {
		return nil, s.translate(err)
	}
	stored := make([]models.IndicatorDefinition, len(definitions))
	for i, definition := range definitions {
		stored[i] = *definition
	}
	return s.toItems(ctx, provider.ID, stored)
}

// resolveMarkets checks the markets of a selection and returns the market ID
// and market IDs to store. A group of one market is stored as that market.
func (s *Service) resolveMarkets(ctx context.Context, providerID uint, selection MarketSelection) (uint, []uint, error) {
	ids := slices.Clone(selection.MarketIDs)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if slices.Contains(ids, 0) {
		return 0, nil, fmt.Errorf("%w: market IDs must be positive", ErrInvalidInput)
	}
	switch selection.Scope {
	case ScopeAll:
		if len(ids) > 0 {
			return 0, nil, fmt.Errorf("%w: an indicator of every market takes no market IDs", ErrInvalidInput)
		}
		return 0, nil, nil
	case ScopeMarket:
		if len(ids) != 1 {
			return 0, nil, fmt.Errorf("%w: select one market", ErrInvalidInput)
		}
	case ScopeGroup:
		if len(ids) == 0 {
			return 0, nil, fmt.Errorf("%w: select the markets of the group", ErrInvalidInput)
		}
	default:
		return 0, nil, fmt.Errorf("%w: scope must be ALL, GROUP or MARKET", ErrInvalidInput)
	}
	labels, err := s.tseMarketRepo.ProviderMarketLabels(ctx, providerID, ids)
	if err != nil {
		return 0, nil, err
	}
	if missing := slices.DeleteFunc(slices.Clone(ids), func(id uint) bool { _, ok := labels[id]; return ok }); len(missing) > 0 {
		return 0, nil, fmt.Errorf("%w: markets %v are not TSE markets", ErrNotFound, missing)
	}
	if len(ids) == 1 {
		return ids[0], nil, nil
	}
	return 0, ids, nil
}

func (s *Service) selectTimeframes(ctx context.Context, ids []uint) ([]models.Timeframe, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: select at least one timeframe", ErrInvalidInput)
	}
	available, err := s.Timeframes(ctx)
	if err != nil {
		return nil, err
	}
	selected := make([]models.Timeframe, 0, len(ids))
	for _, timeframe := range available {
		if slices.Contains(ids, timeframe.ID) {
			selected = append(selected, timeframe)
		}
	}
	for _, id := range ids {
		if !slices.ContainsFunc(selected, func(t models.Timeframe) bool { return t.ID == id }) {
			return nil, fmt.Errorf("%w: timeframe %d is not a TSE candle timeframe", ErrInvalidInput, id)
		}
	}
	return selected, nil
}

// UpdateRequest replaces the params and markets of a definition; the type
// and timeframe of a definition do not change.
type UpdateRequest struct {
	Params  map[string]any
	Markets MarketSelection
}

func (s *Service) Update(ctx context.Context, id uint, req UpdateRequest) (*Item, error) {
	provider, err := s.candles.TSEProvider(ctx)
	if err != nil {
		return nil, s.translate(err)
	}
	definition, err := s.get(ctx, provider.ID, id)
	if err != nil {
		return nil, err
	}
	catalogEntry, ok := Lookup(definition.Type)
	if !ok {
		return nil, fmt.Errorf("%w: indicator type %q cannot be edited", ErrInvalidInput, definition.Type)
	}
	normalized, err := catalogEntry.NormalizeParams(req.Params)
	if err != nil {
		return nil, err
	}
	paramsJSON, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	marketID, marketIDs, err := s.resolveMarkets(ctx, provider.ID, req.Markets)
	if err != nil {
		return nil, err
	}
	updated := &models.IndicatorDefinition{
		ID: definition.ID, Params: paramsJSON, MarketID: marketID, MarketIDs: marketIDs,
		Fingerprint: fingerprint(definition.Type, marketID, marketIDs, definition.ProviderID, definition.TimeframeID, paramsJSON),
	}
	existing, err := s.definitionRepo.ExistingFingerprints(ctx, []string{updated.Fingerprint}, definition.ID)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return nil, fmt.Errorf("%w: %s with these params and markets is already added in this timeframe", ErrConflict, catalogEntry.Name)
	}
	if err := s.definitionRepo.UpdateDefinition(ctx, updated); err != nil {
		return nil, s.translate(err)
	}
	return s.Get(ctx, id)
}

func (s *Service) SetEnabled(ctx context.Context, id uint, enabled bool) (*Item, error) {
	provider, err := s.candles.TSEProvider(ctx)
	if err != nil {
		return nil, s.translate(err)
	}
	if _, err := s.get(ctx, provider.ID, id); err != nil {
		return nil, err
	}
	if err := s.definitionRepo.SetEnabled(ctx, id, enabled); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id uint) error {
	provider, err := s.candles.TSEProvider(ctx)
	if err != nil {
		return s.translate(err)
	}
	if _, err := s.get(ctx, provider.ID, id); err != nil {
		return err
	}
	return s.definitionRepo.Delete(ctx, id)
}

// fingerprint is the fingerprint the indicator calculator gives definitions;
// the definition of a group of markets adds its markets.
func fingerprint(indicatorType string, marketID uint, marketIDs []uint, providerID, timeframeID uint, params []byte) string {
	input := fmt.Sprintf("%s|%d|%d|%d|%s", indicatorType, marketID, providerID, timeframeID, params)
	if len(marketIDs) > 0 {
		ids := make([]string, len(marketIDs))
		for i, id := range marketIDs {
			ids[i] = strconv.FormatUint(uint64(id), 10)
		}
		input += "|markets:" + strings.Join(ids, ",")
	}
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}

func scopeOf(definition models.IndicatorDefinition) Scope {
	switch {
	case definition.MarketID != 0:
		return ScopeMarket
	case len(definition.MarketIDs) > 0:
		return ScopeGroup
	}
	return ScopeAll
}

func (s *Service) toItems(ctx context.Context, providerID uint, definitions []models.IndicatorDefinition) ([]Item, error) {
	items := make([]Item, len(definitions))
	if len(definitions) == 0 {
		return items, nil
	}
	timeframes, err := s.candles.Timeframes(ctx, providerID)
	if err != nil {
		return nil, err
	}
	timeframeLabels := make(map[uint]string, len(timeframes))
	for _, timeframe := range timeframes {
		timeframeLabels[timeframe.ID] = timeframe.Label
	}
	marketIDs := make([]uint, 0)
	for _, definition := range definitions {
		if definition.MarketID != 0 {
			marketIDs = append(marketIDs, definition.MarketID)
		}
		marketIDs = append(marketIDs, definition.MarketIDs...)
	}
	slices.Sort(marketIDs)
	marketIDs = slices.Compact(marketIDs)
	marketLabels := map[uint]string{}
	if len(marketIDs) > 0 {
		if marketLabels, err = s.tseMarketRepo.ProviderMarketLabels(ctx, providerID, marketIDs); err != nil {
			return nil, err
		}
	}
	for i, definition := range definitions {
		params := map[string]any{}
		if len(definition.Params) > 0 {
			if err := json.Unmarshal(definition.Params, &params); err != nil {
				return nil, fmt.Errorf("read params of indicator %d: %w", definition.ID, err)
			}
		}
		ids := definition.MarketIDs
		if definition.MarketID != 0 {
			ids = []uint{definition.MarketID}
		}
		markets := make([]MarketRef, len(ids))
		for j, id := range ids {
			markets[j] = MarketRef{ID: id, Label: marketLabels[id]}
		}
		catalogEntry, _ := Lookup(definition.Type)
		items[i] = Item{
			ID: definition.ID, Type: definition.Type, Definition: catalogEntry, Params: params,
			TimeframeID: definition.TimeframeID, Timeframe: timeframeLabels[definition.TimeframeID],
			Scope: scopeOf(definition), Markets: markets, Enabled: definition.Enabled,
			CreatedAt: definition.CreatedAt, UpdatedAt: definition.UpdatedAt,
		}
	}
	return items, nil
}

// translate maps the errors of the candle service and of unique indexes to
// the errors of this service.
func (s *Service) translate(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, candleService.ErrNotFound):
		return fmt.Errorf("%w: %s", ErrNotFound, strings.TrimPrefix(err.Error(), candleService.ErrNotFound.Error()+": "))
	case errors.Is(err, candleService.ErrInvalidInput):
		return fmt.Errorf("%w: %s", ErrInvalidInput, strings.TrimPrefix(err.Error(), candleService.ErrInvalidInput.Error()+": "))
	case errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "SQLSTATE 23505"):
		return fmt.Errorf("%w: this indicator is already added", ErrConflict)
	}
	return err
}
