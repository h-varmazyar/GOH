package definition

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/h-varmazyar/gatec/internal/services/signal/domain"
)

type MarketCatalog interface {
	HasMarket(context.Context, string) (bool, error)
	HasTimeframe(context.Context, string, string) (bool, error)
}

type Validator struct {
	Markets    MarketCatalog
	Indicators domain.IndicatorCatalog
}

var timeframePattern = regexp.MustCompile(`^[1-9][0-9]*(m|h|d)$`)
var operators = map[string]int{"gt": 2, "gte": 2, "lt": 2, "lte": 2, "eq": 2, "between": 3, "outside": 3, "crosses_above": 2, "crosses_below": 2, "rising": 2, "falling": 2, "within_last": 1, "all": -1, "any": -1, "not": 1}

func (v Validator) Validate(ctx context.Context, s Strategy) error {
	if s.SchemaVersion != 1 {
		return fmt.Errorf("schema_version: unsupported version %d", s.SchemaVersion)
	}
	if strings.TrimSpace(s.ID) == "" {
		return fmt.Errorf("id: required")
	}
	if s.Version < 1 {
		return fmt.Errorf("version: must be positive")
	}
	if s.Mode != "shadow" && s.Mode != "live" {
		return fmt.Errorf("mode: must be shadow or live")
	}
	if strings.TrimSpace(s.Type) == "" {
		return fmt.Errorf("type: required")
	}
	if strings.TrimSpace(s.Market) == "" {
		return fmt.Errorf("market: required")
	}
	if v.Markets != nil {
		ok, err := v.Markets.HasMarket(ctx, s.Market)
		if err != nil {
			return fmt.Errorf("market: catalog lookup: %w", err)
		}
		if !ok {
			return fmt.Errorf("market: unknown market %q", s.Market)
		}
	}
	if s.Direction != "" && s.Direction != "long" && s.Direction != "short" {
		return fmt.Errorf("direction: must be long or short")
	}
	if !timeframePattern.MatchString(s.Timeframes.Trigger) {
		return fmt.Errorf("timeframes.trigger: invalid timeframe %q", s.Timeframes.Trigger)
	}
	seen := map[string]struct{}{s.Timeframes.Trigger: {}}
	for i, tf := range s.Timeframes.Context {
		if !timeframePattern.MatchString(tf) {
			return fmt.Errorf("timeframes.context[%d]: invalid timeframe %q", i, tf)
		}
		if _, ok := seen[tf]; ok {
			return fmt.Errorf("timeframes.context[%d]: duplicate timeframe %q", i, tf)
		}
		seen[tf] = struct{}{}
	}
	for tf := range seen {
		if v.Markets != nil {
			ok, err := v.Markets.HasTimeframe(ctx, s.Market, tf)
			if err != nil {
				return fmt.Errorf("timeframes: catalog lookup for %s: %w", tf, err)
			}
			if !ok {
				return fmt.Errorf("timeframes: %q is unavailable for market %q", tf, s.Market)
			}
		}
	}
	if s.IsEntry() && (s.Risk == nil || s.Risk.StopLoss == nil) {
		return fmt.Errorf("risk.stop_loss: required for entry strategy")
	}
	if s.IsEntry() && (s.Risk == nil || s.Risk.TakeProfit == nil) {
		return fmt.Errorf("risk.take_profit: required for entry strategy")
	}
	if s.Risk != nil {
		if s.Risk.StopLoss != nil {
			if err := validateRiskRule("risk.stop_loss", s.Risk.StopLoss); err != nil {
				return err
			}
		}
		if s.Risk.TakeProfit != nil {
			if err := validateRiskRule("risk.take_profit", s.Risk.TakeProfit); err != nil {
				return err
			}
		}
		if s.Risk.TimeStop != nil && s.Risk.TimeStop.MaxBars < 1 {
			return fmt.Errorf("risk.time_stop.max_bars: must be positive")
		}
	}
	if s.Limits.CooldownBars < 0 || s.Limits.MaxOpenPositions < 0 || s.Limits.MaxSignalsPerDay < 0 {
		return fmt.Errorf("limits: values cannot be negative")
	}
	for i, r := range s.References {
		if strings.TrimSpace(r.Alias) == "" {
			return fmt.Errorf("references[%d].alias: required", i)
		}
		if strings.TrimSpace(r.Symbol) == "" {
			return fmt.Errorf("references[%d].symbol: required", i)
		}
	}
	for name, set := range map[string]*ConditionSet{"entry": s.Entry, "exit": s.Exit} {
		if set == nil {
			continue
		}
		for _, branch := range []struct {
			name  string
			items []map[string]any
		}{{"all", set.All}, {"any", set.Any}} {
			for i, item := range branch.items {
				if err := validateCondition(item, fmt.Sprintf("%s.%s[%d]", name, branch.name, i)); err != nil {
					return err
				}
			}
		}
	}
	if v.Indicators != nil {
		for _, ref := range s.indicatorReferences() {
			key, tf, ok := indicatorName(ref.value, s.Timeframes.Trigger)
			if !ok {
				continue
			}
			exists, err := v.Indicators.Has(ctx, s.Market, tf, key)
			if err != nil {
				return fmt.Errorf("%s: catalog lookup for indicator %q: %w", ref.path, ref.value, err)
			}
			if !exists {
				return fmt.Errorf("%s: indicator %q is not configured for market %q timeframe %q", ref.path, ref.value, s.Market, tf)
			}
		}
	}
	return nil
}

func validateCondition(item map[string]any, path string) error {
	if len(item) != 1 {
		return fmt.Errorf("%s: condition must contain exactly one operator", path)
	}
	for op, arg := range item {
		arity, known := operators[op]
		if !known {
			return fmt.Errorf("%s: unknown operator %q", path, op)
		}
		switch x := arg.(type) {
		case []any:
			if arity >= 0 && len(x) != arity {
				return fmt.Errorf("%s.%s: expected %d operands", path, op, arity)
			}
			for i, val := range x {
				if child, ok := val.(map[string]any); ok {
					if op == "all" || op == "any" {
						if err := validateCondition(child, fmt.Sprintf("%s.%s[%d]", path, op, i)); err != nil {
							return err
						}
					} else if err := validateScaledOperand(child, fmt.Sprintf("%s.%s[%d]", path, op, i)); err != nil {
						return err
					}
				} else if op == "all" || op == "any" {
					return fmt.Errorf("%s.%s[%d]: expected condition object", path, op, i)
				}
			}
		case map[string]any:
			if op == "not" {
				return validateCondition(x, path+".not")
			}
			if op != "within_last" {
				return fmt.Errorf("%s.%s: expected operand list", path, op)
			}
			for key := range x {
				if key != "bars" && key != "cond" {
					return fmt.Errorf("%s.%s.%s: unknown field", path, op, key)
				}
			}
			bars, ok := x["bars"]
			if !ok {
				return fmt.Errorf("%s.%s.bars: required", path, op)
			}
			if !positiveInteger(bars) {
				return fmt.Errorf("%s.%s.bars: must be a positive integer", path, op)
			}
			cond, ok := x["cond"]
			if !ok {
				return fmt.Errorf("%s.%s.cond: required", path, op)
			}
			child, ok := cond.(map[string]any)
			if !ok {
				return fmt.Errorf("%s.%s.cond: expected condition object", path, op)
			}
			if err := validateCondition(child, path+"."+op+".cond"); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s.%s: invalid operands", path, op)
		}
	}
	return nil
}

func validateScaledOperand(value map[string]any, path string) error {
	for key := range value {
		if key != "ref" && key != "mul" && key != "add" {
			return fmt.Errorf("%s.%s: unknown scaled operand field", path, key)
		}
	}
	ref, ok := value["ref"].(string)
	if !ok || strings.TrimSpace(ref) == "" {
		return fmt.Errorf("%s.ref: required", path)
	}
	for _, field := range []string{"mul", "add"} {
		if value[field] != nil {
			switch value[field].(type) {
			case int, int64, float64, string:
			default:
				return fmt.Errorf("%s.%s: expected numeric value", path, field)
			}
		}
	}
	return nil
}

func validateRiskRule(path string, rule *RiskRule) error {
	if rule.Type != "percent" && rule.Type != "atr_multiple" {
		return fmt.Errorf("%s.type: must be percent or atr_multiple", path)
	}
	if rule.Value <= 0 {
		return fmt.Errorf("%s.value: must be positive", path)
	}
	if rule.MinPct < 0 || rule.MaxPct < 0 || (rule.MinPct > 0 && rule.MaxPct > 0 && rule.MinPct > rule.MaxPct) {
		return fmt.Errorf("%s: invalid min_pct/max_pct", path)
	}
	return nil
}

func positiveInteger(value any) bool {
	switch n := value.(type) {
	case int:
		return n > 0
	case int64:
		return n > 0
	case float64:
		return n > 0 && n == float64(int64(n))
	case uint64:
		return n > 0
	}
	return false
}

type FactoryRegistry struct {
	factories map[string]domain.StrategyFactory
}

func NewFactoryRegistry() *FactoryRegistry {
	return &FactoryRegistry{factories: make(map[string]domain.StrategyFactory)}
}
func (r *FactoryRegistry) Register(factory domain.StrategyFactory) error {
	if factory == nil || strings.TrimSpace(factory.Type()) == "" {
		return fmt.Errorf("strategy factory and type are required")
	}
	if _, ok := r.factories[factory.Type()]; ok {
		return fmt.Errorf("strategy factory %q already registered", factory.Type())
	}
	r.factories[factory.Type()] = factory
	return nil
}
func (r *FactoryRegistry) Get(kind string) (domain.StrategyFactory, bool) {
	f, ok := r.factories[kind]
	return f, ok
}
