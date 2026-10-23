package definition

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/h-varmazyar/gatec/internal/services/signal/domain"
	"gopkg.in/yaml.v3"
)

// Strategy is the shared YAML/JSON representation persisted by file and DB repositories.
type Strategy struct {
	SchemaVersion int           `json:"schema_version" yaml:"schema_version"`
	ID            string        `json:"id" yaml:"id"`
	Name          string        `json:"name" yaml:"name"`
	Version       int           `json:"version" yaml:"version"`
	Enabled       bool          `json:"enabled" yaml:"enabled"`
	Mode          string        `json:"mode" yaml:"mode"`
	Type          string        `json:"type" yaml:"type"`
	Market        string        `json:"market" yaml:"market"`
	Direction     string        `json:"direction,omitempty" yaml:"direction,omitempty"`
	Universe      Universe      `json:"universe,omitempty" yaml:"universe,omitempty"`
	Timeframes    Timeframes    `json:"timeframes" yaml:"timeframes"`
	References    []Reference   `json:"references,omitempty" yaml:"references,omitempty"`
	Entry         *ConditionSet `json:"entry,omitempty" yaml:"entry,omitempty"`
	Exit          *ConditionSet `json:"exit,omitempty" yaml:"exit,omitempty"`
	Risk          *Risk         `json:"risk,omitempty" yaml:"risk,omitempty"`
	Limits        Limits        `json:"limits,omitempty" yaml:"limits,omitempty"`
}

type Universe struct {
	Symbols []string `json:"symbols,omitempty" yaml:"symbols,omitempty"`
	Exclude []string `json:"exclude,omitempty" yaml:"exclude,omitempty"`
}
type Timeframes struct {
	Trigger string   `json:"trigger" yaml:"trigger"`
	Context []string `json:"context,omitempty" yaml:"context,omitempty"`
}
type Reference struct {
	Alias  string `json:"alias" yaml:"alias"`
	Symbol string `json:"symbol" yaml:"symbol"`
}
type ConditionSet struct {
	All []map[string]any `json:"all,omitempty" yaml:"all,omitempty"`
	Any []map[string]any `json:"any,omitempty" yaml:"any,omitempty"`
}
type Risk struct {
	StopLoss   *RiskRule `json:"stop_loss,omitempty" yaml:"stop_loss,omitempty"`
	TakeProfit *RiskRule `json:"take_profit,omitempty" yaml:"take_profit,omitempty"`
	TimeStop   *TimeStop `json:"time_stop,omitempty" yaml:"time_stop,omitempty"`
}
type RiskRule struct {
	Type   string  `json:"type" yaml:"type"`
	Value  float64 `json:"value" yaml:"value"`
	MinPct float64 `json:"min_pct,omitempty" yaml:"min_pct,omitempty"`
	MaxPct float64 `json:"max_pct,omitempty" yaml:"max_pct,omitempty"`
}
type TimeStop struct {
	MaxBars int `json:"max_bars" yaml:"max_bars"`
}
type Limits struct {
	CooldownBars     int `json:"cooldown_bars,omitempty" yaml:"cooldown_bars,omitempty"`
	MaxOpenPositions int `json:"max_open_positions,omitempty" yaml:"max_open_positions,omitempty"`
	MaxSignalsPerDay int `json:"max_signals_per_day,omitempty" yaml:"max_signals_per_day,omitempty"`
}

func ParseYAML(data []byte) (Strategy, error) {
	var value Strategy
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("decode strategy YAML: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return value, fmt.Errorf("decode strategy YAML: multiple documents are not supported")
		}
		return value, fmt.Errorf("decode strategy YAML: %w", err)
	}
	return value, nil
}

func ParseJSON(data []byte) (Strategy, error) {
	var value Strategy
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("decode strategy JSON: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return value, fmt.Errorf("decode strategy JSON: trailing content")
	}
	return value, nil
}

func (s Strategy) Domain() (domain.StrategyDefinition, error) {
	body, err := json.Marshal(s)
	if err != nil {
		return domain.StrategyDefinition{}, err
	}
	return domain.StrategyDefinition{SchemaVersion: s.SchemaVersion, ID: s.ID, Version: s.Version, Enabled: s.Enabled, Mode: domain.Mode(s.Mode), Type: s.Type, Market: s.Market, Config: body}, nil
}

func (s Strategy) IsEntry() bool {
	return s.Entry != nil && (len(s.Entry.All) > 0 || len(s.Entry.Any) > 0)
}
func (s Strategy) AllTimeframes() []string {
	out := []string{s.Timeframes.Trigger}
	return append(out, s.Timeframes.Context...)
}

type reference struct {
	value string
	path  string
}

func (s Strategy) indicatorReferences() []reference {
	refs := make([]reference, 0)
	var walk func(any, string)
	walk = func(v any, path string) {
		switch x := v.(type) {
		case string:
			refs = append(refs, reference{value: x, path: path})
		case map[string]any:
			for k, val := range x {
				walk(val, path+"."+k)
			}
		case []any:
			for i, v := range x {
				walk(v, fmt.Sprintf("%s[%d]", path, i))
			}
		}
	}
	for name, set := range map[string]*ConditionSet{"entry": s.Entry, "exit": s.Exit} {
		if set == nil {
			continue
		}
		for branch, items := range map[string][]map[string]any{"all": set.All, "any": set.Any} {
			for i, item := range items {
				walk(item, fmt.Sprintf("%s.%s[%d]", name, branch, i))
			}
		}
	}
	return refs
}

func indicatorName(reference, defaultTF string) (string, string, bool) {
	if _, err := strconv.ParseFloat(reference, 64); err == nil {
		return "", "", false
	}
	parts := strings.Split(reference, ".")
	for _, part := range parts {
		switch part {
		case "open", "high", "low", "close", "volume":
			return "", "", false
		}
	}
	if len(parts) == 1 {
		return indicatorBase(parts[0]), defaultTF, true
	}
	// An optional timeframe and/or reference alias precede the indicator name.
	tf := defaultTF
	key := parts[len(parts)-1]
	for _, part := range parts {
		if timeframePattern.MatchString(part) {
			tf = part
		}
	}
	// output-qualified indicators use the penultimate segment as their type.
	if len(parts) >= 2 && isIndicatorOutput(parts[len(parts)-1]) {
		key = parts[len(parts)-2]
	}
	return indicatorBase(key), tf, true
}

func indicatorBase(key string) string {
	if i := strings.IndexByte(key, '_'); i > 0 {
		if _, err := strconv.Atoi(key[i+1:]); err == nil {
			return key[:i]
		}
	}
	return key
}
func isIndicatorOutput(s string) bool {
	switch s {
	case "hist", "signal", "macd", "upper", "upper_band", "lower", "lower_band", "middle", "middle_band", "indexK", "indexD", "slope", "slope_percent", "distance", "distance_percent", "value", "ratio", "is_spike":
		return true
	}
	return false
}
