package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/h-varmazyar/gatec/internal/services/signal/definition"
	"github.com/h-varmazyar/gatec/internal/services/signal/domain"
)

const DeclarativeType = "declarative"

type DeclarativeFactory struct {
	Validator  *definition.Validator
	Conditions *ConditionRegistry
}

func (f *DeclarativeFactory) Type() string { return DeclarativeType }

func (f *DeclarativeFactory) Build(envelope domain.StrategyDefinition) (domain.Strategy, error) {
	var config definition.Strategy
	if err := json.Unmarshal(envelope.Config, &config); err != nil {
		return nil, fmt.Errorf("decode declarative strategy %q: %w", envelope.ID, err)
	}
	if config.ID != envelope.ID || config.Version != envelope.Version || config.Market != envelope.Market || config.Type != DeclarativeType {
		return nil, fmt.Errorf("declarative strategy envelope does not match its config")
	}
	if f.Validator != nil {
		if err := f.Validator.Validate(context.Background(), config); err != nil {
			return nil, err
		}
	}
	conditions := f.Conditions
	if conditions == nil {
		conditions = NewConditionRegistry()
	}
	return &DeclarativeStrategy{definition: config, conditions: conditions}, nil
}

type DeclarativeStrategy struct {
	definition definition.Strategy
	conditions *ConditionRegistry
}

func (s *DeclarativeStrategy) ID() string        { return s.definition.ID }
func (s *DeclarativeStrategy) Version() int      { return s.definition.Version }
func (s *DeclarativeStrategy) Mode() domain.Mode { return domain.Mode(s.definition.Mode) }

func (s *DeclarativeStrategy) Subscriptions() []domain.Subscription {
	refs := make([]domain.Reference, 0, len(s.definition.References))
	for _, ref := range s.definition.References {
		refs = append(refs, domain.Reference{Alias: ref.Alias, Market: s.definition.Market, Symbol: ref.Symbol})
	}
	return []domain.Subscription{{
		Market:            s.definition.Market,
		Symbols:           append([]string(nil), s.definition.Universe.Symbols...),
		TriggerTimeframe:  s.definition.Timeframes.Trigger,
		ContextTimeframes: append([]string(nil), s.definition.Timeframes.Context...),
		References:        refs,
	}}
}

func (s *DeclarativeStrategy) Requirements() []domain.IndicatorRequirement {
	seen := make(map[string]struct{})
	requirements := make([]domain.IndicatorRequirement, 0)
	for _, set := range []*definition.ConditionSet{s.definition.Entry, s.definition.Exit} {
		if set == nil {
			continue
		}
		for _, group := range [][]map[string]any{set.All, set.Any} {
			for _, condition := range group {
				collectRequirements(condition, s.definition.Market, s.definition.Timeframes.Trigger, seen, &requirements)
			}
		}
	}
	return requirements
}

func collectRequirements(value any, market, timeframe string, seen map[string]struct{}, out *[]domain.IndicatorRequirement) {
	switch item := value.(type) {
	case string:
		ref := strings.TrimSuffix(item, "[-1]")
		parts := strings.Split(ref, ".")
		key := parts[len(parts)-1]
		if isCandleField(key) || key == "hist" || key == "histogram" || key == "signal" || key == "upper_band" || key == "lower_band" || key == "middle_band" {
			return
		}
		tf := timeframe
		if len(parts) > 1 && isTimeframe(parts[0]) {
			tf, key = parts[0], parts[1]
		}
		if len(parts) > 2 && isTimeframe(parts[1]) {
			tf, key = parts[1], parts[2]
		}
		if i := strings.LastIndexByte(key, '_'); i > 0 {
			if _, err := strconv.Atoi(key[i+1:]); err == nil {
				key = key[:i]
			}
		}
		id := market + "\x00" + tf + "\x00" + key
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			*out = append(*out, domain.IndicatorRequirement{Market: market, Timeframe: tf, Key: key})
		}
	case map[string]any:
		if ref, ok := item["ref"].(string); ok {
			collectRequirements(ref, market, timeframe, seen, out)
			return
		}
		for _, child := range item {
			collectRequirements(child, market, timeframe, seen, out)
		}
	case []any:
		for _, child := range item {
			collectRequirements(child, market, timeframe, seen, out)
		}
	}
}

func (s *DeclarativeStrategy) Evaluate(input domain.EvalContext) (domain.Decision, error) {
	decision := domain.Decision{Kind: domain.DecisionNone, Direction: domain.Direction(strings.ToLower(s.definition.Direction))}
	if !s.definition.Enabled && input.Position == nil {
		return decision, nil
	}
	eval := &Evaluation{Context: input, Strategy: s.definition, Registry: s.conditions}
	var set *definition.ConditionSet
	if input.Position != nil {
		set = s.definition.Exit
	} else {
		set = s.definition.Entry
	}
	if set == nil {
		return decision, nil
	}
	matched, err := evaluateSet(eval, set)
	if err != nil {
		return decision, err
	}
	if eval.NotReady {
		decision.Reason = eval.Reason
		decision.Reason.Text = "not_ready: " + strings.Join(eval.Missing, ", ")
		return decision, nil
	}
	decision.Reason = eval.Reason
	if !matched {
		return decision, nil
	}
	if input.Position != nil {
		decision.Kind = domain.DecisionExit
		decision.Reason.Text = "declarative exit conditions matched"
	} else {
		decision.Kind = domain.DecisionEntry
		decision.Reason.Text = "declarative entry conditions matched"
	}
	return decision, nil
}

func evaluateSet(eval *Evaluation, set *definition.ConditionSet) (bool, error) {
	allMatched := true
	for _, raw := range set.All {
		matched, err := eval.Registry.Evaluate(eval, raw)
		if err != nil {
			return false, err
		}
		if !matched {
			allMatched = false
		}
	}
	anyMatched := len(set.Any) == 0
	for _, raw := range set.Any {
		matched, err := eval.Registry.Evaluate(eval, raw)
		if err != nil {
			return false, err
		}
		if matched {
			anyMatched = true
		}
	}
	return allMatched && anyMatched, nil
}
