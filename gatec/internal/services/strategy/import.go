package strategy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"

	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	strategyloader "github.com/h-varmazyar/gatec/pkg/strategy/loader"
)

func (s *Service) ImportNewStrategy(ctx context.Context, filePath string) (*models.Strategy, error) {
	strategyPath := "./assets/sample_strategy.yaml"
	loadedStrategy, err := strategyloader.Load(strategyPath)
	if err != nil {
		return nil, err
	}

	providerName, ok := consts.ToProvider(loadedStrategy.Strategy.Provider)
	if !ok {
		return nil, errors.New("invalid provider")
	}

	provider, err := s.providerRepo.ReturnByName(ctx, providerName)
	if err != nil {
		return nil, err
	}

	markets, err := s.prepareMarkets(ctx, provider, loadedStrategy.Market)
	if err != nil {
		return nil, err
	}

	for _, ind := range loadedStrategy.Indicators {
		params, err := normalizeJSON(ind.Configs)
		if err != nil {
			return nil, err
		}

		fp := buildFingerprint(ind.Type, 0, 0, 0, params)
		// fp := buildFingerprint(ind.Type, ind.MarketID, ind.ProviderID, ind.TimeframeID, params)
		m := models.IndicatorDefinition{
			MarketID:    0,
			ProviderID:  0,
			TimeframeID: 0,
			Type:        ind.Type,
			Params:      params,
			Fingerprint: fp,
		}
		if err := s.indicatorDefinitionRepo.Upsert(ctx, &m); err != nil {
			return nil, err
		}
		ind.ID = m.ID
	}

	strategy := &models.Strategy{
		Name:        loadedStrategy.Strategy.Name,
		Description: loadedStrategy.Strategy.Description,
		Provider:    provider,
		// Enabled:     loadedStrategy.Strategy.Enabled,
		Markets: markets,
	}

	return strategy, nil
}

func (s *Service) prepareMarkets(ctx context.Context, provider *models.Provider, strategyMarkets strategyloader.MarketConfig) ([]*models.Market, error) {
	markets := make([]*models.Market, 0)
	var err error
	if len(strategyMarkets.Symbols) == 1 && strategyMarkets.Symbols[0] == "*" {
		markets, err = s.marketRepo.List(ctx, map[string]any{
			"provider_id = ?": provider.ID,
		})
		if err != nil {
			return nil, err
		}
	} else {
		markets, err = s.marketRepo.List(ctx, map[string]any{
			"provider_id = ?": provider.ID,
			"name IN ?":       strategyMarkets.Symbols,
		})
		if err != nil {
			return nil, err
		}
	}

	if len(strategyMarkets.ExcludedSymbols) != 0 {
		for _, excluded := range strategyMarkets.ExcludedSymbols {
			for i, market := range markets {
				if market.Name == excluded {
					markets = append(markets[:i], markets[i+1:]...)
					break
				}
			}
		}
	}

	return markets, nil
}

func buildFingerprint(indType string, marketID, providerID, timeframeID uint, params []byte) string {
	h := sha256.New()
	_, _ = h.Write([]byte(indType))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(jsonNumber(marketID)))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(jsonNumber(providerID)))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(jsonNumber(timeframeID)))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write(params)
	return hex.EncodeToString(h.Sum(nil))
}

func jsonNumber(v uint) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func normalizeJSON(input map[string]any) ([]byte, error) {
	if input == nil {
		return []byte("{}"), nil
	}
	canonical := normalizeValue(input)
	return json.Marshal(canonical)
}

func normalizeValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		ordered := make(map[string]any, len(t))
		for _, k := range keys {
			ordered[k] = normalizeValue(t[k])
		}
		return ordered
	case []any:
		arr := make([]any, len(t))
		for i, x := range t {
			arr[i] = normalizeValue(x)
		}
		return arr
	default:
		return v
	}
}
