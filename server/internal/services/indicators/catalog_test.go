package indicators

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/worker/workers"
)

// Every catalog entry, with its defaults, must be buildable by the
// indicator calculator, which reads the same param keys.
func TestCatalogDefaultsBuildIndicators(t *testing.T) {
	seen := map[string]bool{}
	for _, definition := range Catalog() {
		if seen[definition.Type] {
			t.Fatalf("type %s is listed twice", definition.Type)
		}
		seen[definition.Type] = true
		if !validCategory(definition.Category) {
			t.Fatalf("%s has unknown category %q", definition.Type, definition.Category)
		}
		params, err := definition.NormalizeParams(map[string]any{})
		if err != nil {
			t.Fatalf("%s defaults: %v", definition.Type, err)
		}
		raw, err := json.Marshal(params)
		if err != nil {
			t.Fatal(err)
		}
		indicator, err := workers.CreateIndicatorFromDefinition(models.IndicatorDefinition{
			ID: 1, MarketID: 2, ProviderID: 3, TimeframeID: 4, Type: definition.Type, Params: raw,
		})
		if err != nil {
			t.Fatalf("%s with %s: %v", definition.Type, raw, err)
		}
		if indicator.GetID() != 1 || indicator.GetMarketID() != 2 || indicator.GetTimeframeID() != 4 {
			t.Fatalf("%s: unexpected ids", definition.Type)
		}
	}
}

func validCategory(category Category) bool {
	for _, info := range Categories {
		if info.Key == category {
			return true
		}
	}
	return false
}

func TestNormalizeParams(t *testing.T) {
	rsi, _ := Lookup("rsi")
	got, err := rsi.NormalizeParams(map[string]any{"period": float64(21)})
	if err != nil || got["period"] != 21 {
		t.Fatalf("got %v, %v", got, err)
	}
	got, err = rsi.NormalizeParams(map[string]any{"period": "9"})
	if err != nil || got["period"] != 9 {
		t.Fatalf("string number: got %v, %v", got, err)
	}

	for name, params := range map[string]map[string]any{
		"fraction":    {"period": 14.5},
		"zero":        {"period": 0},
		"too large":   {"period": MaxPeriod + 1},
		"unknown key": {"length": 14},
		"not number":  {"period": true},
	} {
		if _, err := rsi.NormalizeParams(params); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: expected invalid input, got %v", name, err)
		}
	}

	spike, _ := Lookup("VOLUME_SPIKE")
	got, err = spike.NormalizeParams(map[string]any{"multiplier": 1.5})
	if err != nil || got["multiplier"] != 1.5 || got["period"] != 20 {
		t.Fatalf("volume spike: got %v, %v", got, err)
	}

	slope, _ := Lookup("EMA_SLOPE")
	got, err = slope.NormalizeParams(map[string]any{"source": "hlc3"})
	if err != nil || got["source"] != "HLC3" {
		t.Fatalf("source: got %v, %v", got, err)
	}
	if _, err := slope.NormalizeParams(map[string]any{"source": "MEDIAN"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("unknown source: got %v", err)
	}

	macd, _ := Lookup("MACD")
	if _, err := macd.NormalizeParams(map[string]any{"fast_period": 30, "slow_period": 26}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("MACD fast >= slow: got %v", err)
	}
}

func TestFingerprintMatchesCalculator(t *testing.T) {
	// The indicator calculator seeds definitions with sha256 of
	// "TYPE|market|provider|timeframe|params"; definitions of every market and
	// of one market keep that fingerprint.
	sum := sha256.Sum256([]byte(`RSI|7|1|2|{"period":14}`))
	if got := fingerprint("RSI", 7, nil, 1, 2, []byte(`{"period":14}`)); got != hex.EncodeToString(sum[:]) {
		t.Fatalf("market fingerprint %s differs from the calculator", got)
	}
	all := fingerprint("RSI", 0, nil, 1, 2, []byte(`{"period":14}`))
	group := fingerprint("RSI", 0, []uint{3, 5}, 1, 2, []byte(`{"period":14}`))
	other := fingerprint("RSI", 0, []uint{3, 6}, 1, 2, []byte(`{"period":14}`))
	if all == group || group == other || len(group) != 64 {
		t.Fatalf("fingerprints %s %s %s", all, group, other)
	}
}

func TestScopeOf(t *testing.T) {
	for _, tc := range []struct {
		definition models.IndicatorDefinition
		want       Scope
	}{
		{models.IndicatorDefinition{}, ScopeAll},
		{models.IndicatorDefinition{MarketIDs: []uint{}}, ScopeAll},
		{models.IndicatorDefinition{MarketID: 4}, ScopeMarket},
		{models.IndicatorDefinition{MarketIDs: []uint{4, 9}}, ScopeGroup},
	} {
		if got := scopeOf(tc.definition); got != tc.want {
			t.Errorf("%+v: got %s, want %s", tc.definition, got, tc.want)
		}
	}
}

func TestMatchingTypes(t *testing.T) {
	types := matchingTypes(Filter{Category: CategoryVolume})
	if len(types) == 0 {
		t.Fatal("no volume indicators")
	}
	for _, indicatorType := range types {
		definition, _ := Lookup(indicatorType)
		if definition.Category != CategoryVolume {
			t.Fatalf("%s is not a volume indicator", indicatorType)
		}
	}
	if got := matchingTypes(Filter{Query: "قدرت نسبی"}); len(got) != 1 || got[0] != "RSI" {
		t.Fatalf("persian query: %v", got)
	}
	if got := matchingTypes(Filter{Query: "macd"}); len(got) != 1 || got[0] != "MACD" {
		t.Fatalf("query: %v", got)
	}
	if got := matchingTypes(Filter{Query: "nothing-matches"}); got == nil || len(got) != 0 {
		t.Fatalf("no match must be an empty, non-nil list: %v", got)
	}
}
