package loader

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type StrategyFile struct {
	Version  int          `yaml:"version"`
	Strategy StrategyMeta `yaml:"strategy"`
	Market   MarketConfig `yaml:"market"`
	// Filters        FiltersConfig     `yaml:"filters"`
	// Regime         RegimeConfig      `yaml:"regime"`
	Entry EntryConfig `yaml:"entry"`
	// Exit           ExitConfig        `yaml:"exit"`
	// Risk           RiskConfig        `yaml:"risk"`
	// Execution      ExecutionConfig   `yaml:"execution"`
	// Scoring        ScoringConfig     `yaml:"scoring"`
	// State          StateConfig       `yaml:"state"`
	// MultiTimeframe MultiTFConfig     `yaml:"multi_timeframe"`
	// Telemetry      TelemetryConfig   `yaml:"telemetry"`
	Indicators map[string]IndicatorConfig `yaml:"indicators"`
}

type StrategyMeta struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Provider    string   `yaml:"provider"`
	Enabled     bool     `yaml:"enabled"`
	Tags        []string `yaml:"tags"`
}
type MarketConfig struct {
	Symbols         []string `yaml:"symbols"`
	ExcludedSymbols []string `yaml:"excluded_symbols"`
	Timeframes      []string `yaml:"timeframes"`
	Sessions        []string `yaml:"sessions"`
}
type RuleGroup struct{ All, Any []string }
type RegimeNode struct {
	Conditions RuleGroup `yaml:"conditions"`
}
type RegimeConfig struct{ Trending, Ranging, Volatile RegimeNode }
type EntrySide struct {
	All   []string `yaml:"all"`
	Score struct {
		Minimum float64            `yaml:"minimum"`
		Weights map[string]float64 `yaml:"weights"`
	} `yaml:"score"`
}
type EntryConfig struct{ Long, Short EntrySide }
type ExitSide struct {
	Any []string `yaml:"any"`
}
type ExitConfig struct{ Long, Short ExitSide }
type FiltersConfig struct {
	Market struct {
		MinDailyVolumeUSDT float64 `yaml:"min_daily_volume_usdt"`
	} `yaml:"market"`
}
type RiskConfig struct {
	Position struct {
		MaxOpenPositions int `yaml:"max_open_positions"`
	} `yaml:"position"`
}
type ExecutionConfig struct {
	OrderType string `yaml:"order_type"`
}
type ScoringConfig struct {
	Confidence struct {
		Minimum float64 `yaml:"minimum"`
	} `yaml:"confidence"`
}
type StateConfig struct {
	SignalCooldownSeconds int `yaml:"signal_cooldown_seconds"`
}
type MultiTFConfig struct {
	Enabled      bool `yaml:"enabled"`
	Confirmation struct {
		HigherTimeframe string    `yaml:"higher_timeframe"`
		Conditions      RuleGroup `yaml:"conditions"`
	} `yaml:"confirmation"`
}
type TelemetryConfig struct {
	Enabled bool     `yaml:"enabled"`
	Metrics []string `yaml:"metrics"`
}

type IndicatorConfig struct {
	ID   uint   `yaml:"id"`
	Type string `yaml:"type"`
	// MarketID    uint           `yaml:"market_id"`
	// ProviderID  uint           `yaml:"provider_id"`
	// Timeframes string         `yaml:"timeframes"`
	Configs map[string]any `yaml:"configs"`
}

func Load(path string) (*StrategyFile, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	var s StrategyFile
	if e = yaml.Unmarshal(b, &s); e != nil {
		return nil, e
	}
	return &s, s.Validate()
}

func (s StrategyFile) Validate() error {
	if s.Version <= 0 {
		return fmt.Errorf("version must be >0")
	}
	if s.Strategy.Name == "" {
		return fmt.Errorf("strategy.name is required")
	}
	if len(s.Entry.Long.All) == 0 && len(s.Entry.Short.All) == 0 {
		return fmt.Errorf("entry rules are empty")
	}
	return nil
}
