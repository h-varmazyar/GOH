package loader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/h-varmazyar/gatec/internal/repositories"
)

func EnsureIndicators(ctx context.Context, repo *repositories.IndicatorDefinitionRepository, sf *StrategyFile) error {
	if sf == nil || repo == nil {
		return nil
	}
	// for _, ind := range sf.Indicators {
	// params, err := normalizeJSON(ind.Configs)
	// if err != nil {
	// 	return err
	// }

	// fp := buildFingerprint(ind.Type, ind.MarketID, ind.ProviderID, ind.TimeframeID, params)
	// m := models.IndicatorDefinition{
	// 	MarketID:    ind.MarketID,
	// 	ProviderID:  ind.ProviderID,
	// 	TimeframeID: ind.TimeframeID,
	// 	Type:        ind.Type,
	// 	Params:      params,
	// 	Fingerprint: fp,
	// }
	// if err := repo.Upsert(ctx, &m); err != nil {
	// 	return err
	// }
	// ind.ID = m.ID
	// }
	return nil
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
