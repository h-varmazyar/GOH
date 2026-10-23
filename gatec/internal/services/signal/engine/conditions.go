package engine

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/h-varmazyar/gatec/internal/services/signal/definition"
	"github.com/h-varmazyar/gatec/internal/services/signal/domain"
	"github.com/shopspring/decimal"
)

// Condition is a pluggable declarative condition operator.
type Condition interface {
	Evaluate(*Evaluation, any) (bool, error)
}

type ConditionFunc func(*Evaluation, any) (bool, error)

func (f ConditionFunc) Evaluate(e *Evaluation, value any) (bool, error) { return f(e, value) }

// ConditionRegistry maps schema operator names to condition implementations.
type ConditionRegistry struct{ conditions map[string]Condition }

func NewConditionRegistry() *ConditionRegistry {
	r := &ConditionRegistry{conditions: make(map[string]Condition)}
	r.Register("gt", compareCondition(func(c int) bool { return c > 0 }))
	r.Register("gte", compareCondition(func(c int) bool { return c >= 0 }))
	r.Register("lt", compareCondition(func(c int) bool { return c < 0 }))
	r.Register("lte", compareCondition(func(c int) bool { return c <= 0 }))
	r.Register("eq", compareCondition(func(c int) bool { return c == 0 }))
	r.Register("between", rangeCondition(false))
	r.Register("outside", rangeCondition(true))
	r.Register("crosses_above", crossCondition(true))
	r.Register("crosses_below", crossCondition(false))
	r.Register("rising", trendCondition(true))
	r.Register("falling", trendCondition(false))
	r.Register("within_last", ConditionFunc(withinLastCondition))
	r.Register("all", logicalCondition(true))
	r.Register("any", logicalCondition(false))
	r.Register("not", ConditionFunc(notCondition))
	return r
}

func (r *ConditionRegistry) Register(name string, condition Condition) error {
	name = strings.TrimSpace(name)
	if name == "" || condition == nil {
		return fmt.Errorf("condition name and implementation are required")
	}
	if r.conditions == nil {
		r.conditions = make(map[string]Condition)
	}
	if _, exists := r.conditions[name]; exists {
		return fmt.Errorf("condition %q already registered", name)
	}
	r.conditions[name] = condition
	return nil
}

func (r *ConditionRegistry) Evaluate(e *Evaluation, raw map[string]any) (bool, error) {
	if len(raw) != 1 {
		return false, fmt.Errorf("condition must contain exactly one operator")
	}
	for name, args := range raw {
		condition, ok := r.conditions[name]
		if !ok {
			return false, fmt.Errorf("unknown condition %q", name)
		}
		return condition.Evaluate(e, args)
	}
	return false, nil
}

type Evaluation struct {
	Context   domain.EvalContext
	Strategy  definition.Strategy
	Registry  *ConditionRegistry
	Reason    domain.Reason
	NotReady  bool
	Missing   []string
	currentAt int
}

var offsetSuffix = regexp.MustCompile(`\[(-[0-9]+)\]$`)
var timeframeReference = regexp.MustCompile(`^[1-9][0-9]*(m|h|d)$`)

type operand struct {
	Ref string
	Mul decimal.Decimal
	Add decimal.Decimal
}

func parseOperand(raw any) (operand, error) {
	switch value := raw.(type) {
	case string:
		return operand{Ref: value, Mul: decimal.NewFromInt(1), Add: decimal.Zero}, nil
	case int:
		return operand{Mul: decimal.NewFromInt(1), Add: decimal.NewFromInt(int64(value))}, nil
	case int64:
		return operand{Mul: decimal.NewFromInt(1), Add: decimal.NewFromInt(value)}, nil
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return operand{}, fmt.Errorf("operand must be finite")
		}
		return operand{Mul: decimal.NewFromInt(1), Add: decimal.NewFromFloat(value)}, nil
	case map[string]any:
		ref, ok := value["ref"].(string)
		if !ok || strings.TrimSpace(ref) == "" {
			return operand{}, fmt.Errorf("scaled operand requires ref")
		}
		mul, err := decimalValue(value["mul"], decimal.NewFromInt(1))
		if err != nil {
			return operand{}, fmt.Errorf("mul: %w", err)
		}
		add, err := decimalValue(value["add"], decimal.Zero)
		if err != nil {
			return operand{}, fmt.Errorf("add: %w", err)
		}
		return operand{Ref: ref, Mul: mul, Add: add}, nil
	default:
		return operand{}, fmt.Errorf("unsupported operand type %T", raw)
	}
}

func decimalValue(raw any, fallback decimal.Decimal) (decimal.Decimal, error) {
	if raw == nil {
		return fallback, nil
	}
	switch value := raw.(type) {
	case float64:
		return decimal.NewFromFloat(value), nil
	case int:
		return decimal.NewFromInt(int64(value)), nil
	case int64:
		return decimal.NewFromInt(value), nil
	case string:
		return decimal.NewFromString(value)
	default:
		return decimal.Zero, fmt.Errorf("expected number, got %T", raw)
	}
}

func (e *Evaluation) value(raw any, historyOffset int) (decimal.Decimal, string, bool, error) {
	op, err := parseOperand(raw)
	if err != nil {
		return decimal.Zero, "", false, err
	}
	if op.Ref == "" {
		return op.Add, "", true, nil
	}
	value, ok, err := e.resolve(op.Ref, historyOffset+e.currentAt)
	if err != nil || !ok {
		return decimal.Zero, op.Ref, false, err
	}
	return value.Mul(op.Mul).Add(op.Add), op.Ref, true, nil
}

func (e *Evaluation) resolve(reference string, historyOffset int) (decimal.Decimal, bool, error) {
	offset := 0
	if match := offsetSuffix.FindStringSubmatch(reference); len(match) == 2 {
		n, _ := strconv.Atoi(match[1])
		offset = -n
		reference = reference[:len(reference)-len(match[0])]
	}
	parts := strings.Split(reference, ".")
	if len(parts) == 0 {
		return decimal.Zero, false, nil
	}
	timeframe := e.Strategy.Timeframes.Trigger
	symbol := e.Context.Current.Candle.Symbol
	partIndex := 0
	if len(parts) > 1 {
		if isTimeframe(parts[0]) {
			timeframe, partIndex = parts[0], 1
		} else if len(parts) > 2 && isTimeframe(parts[1]) {
			alias := parts[0]
			for _, reference := range e.Strategy.References {
				if reference.Alias == alias {
					symbol = reference.Symbol
					break
				}
			}
			if symbol == e.Context.Current.Candle.Symbol {
				return decimal.Zero, false, nil
			}
			timeframe, partIndex = parts[1], 2
		} else if isCandleField(parts[0]) {
			partIndex = 0
		} else if len(parts) == 2 && isOutput(parts[1]) {
			// An output-qualified indicator, e.g. macd.histogram.
			partIndex = 0
		} else {
			for _, reference := range e.Strategy.References {
				if reference.Alias == parts[0] {
					symbol, partIndex = reference.Symbol, 1
					break
				}
			}
		}
	}
	series := e.Context.Current.Candle
	var snapshots []domain.Snapshot
	if timeframe == e.Strategy.Timeframes.Trigger && symbol == series.Symbol {
		snapshots = e.Context.History
		if len(snapshots) == 0 {
			snapshots = []domain.Snapshot{e.Context.Current}
		}
	} else {
		if e.Context.Contexts == nil {
			return decimal.Zero, false, nil
		}
		key := domain.SeriesKey{Market: series.Market, Symbol: symbol, Timeframe: timeframe}
		count := offset + historyOffset + 1
		if count < 1 {
			count = 1
		}
		loaded, err := e.Context.Contexts.Window(e.Context.Context, key, count)
		if err != nil {
			return decimal.Zero, false, err
		}
		for _, item := range loaded {
			if item.Candle.Closed && !item.Candle.CloseTime.After(series.CloseTime) {
				snapshots = append(snapshots, item)
			}
		}
	}
	index := len(snapshots) - 1 - offset - historyOffset
	if index < 0 || index >= len(snapshots) {
		return decimal.Zero, false, nil
	}
	snapshot := snapshots[index]
	if !snapshot.Candle.Closed || snapshot.Candle.CloseTime.After(series.CloseTime) {
		return decimal.Zero, false, nil
	}
	name := strings.Join(parts[partIndex:], ".")
	if isCandleField(name) {
		switch name {
		case "open":
			return snapshot.Candle.Open, true, nil
		case "high":
			return snapshot.Candle.High, true, nil
		case "low":
			return snapshot.Candle.Low, true, nil
		case "close":
			return snapshot.Candle.Close, true, nil
		case "volume":
			return snapshot.Candle.Volume, true, nil
		}
	}
	keys := []string{name}
	if name == "macd.hist" {
		keys = append(keys, "macd.histogram")
	}
	if name == "hist" {
		keys = append(keys, "histogram")
	}
	if len(parts)-partIndex == 2 {
		base, output := parts[len(parts)-2], parts[len(parts)-1]
		keys = append(keys, base+"."+output)
		if output == "hist" {
			keys = append(keys, base+".histogram")
		}
	}
	for _, key := range keys {
		if value, ok := snapshot.Indicators[key]; ok {
			return value, true, nil
		}
	}
	return decimal.Zero, false, nil
}

func isTimeframe(value string) bool { return timeframeReference.MatchString(value) }
func isCandleField(value string) bool {
	return value == "open" || value == "high" || value == "low" || value == "close" || value == "volume"
}
func isOutput(value string) bool {
	return value == "hist" || value == "histogram" || value == "signal" || value == "upper" || value == "upper_band" || value == "lower" || value == "lower_band" || value == "middle_band" || value == "indexK" || value == "indexD" || value == "value" || value == "distance" || value == "distance_percent" || value == "slope" || value == "slope_percent"
}

func compareCondition(compare func(int) bool) Condition {
	return ConditionFunc(func(e *Evaluation, raw any) (bool, error) {
		args, err := argsOf(raw, 2)
		if err != nil {
			return false, err
		}
		left, leftName, lok, err := e.value(args[0], 0)
		if err != nil {
			return false, err
		}
		right, rightName, rok, err := e.value(args[1], 0)
		if err != nil {
			return false, err
		}
		if !lok || !rok {
			e.missing(leftName)
			e.missing(rightName)
			return false, nil
		}
		matched := compare(left.Cmp(right))
		e.record(fmt.Sprintf("%v %v %v", args[0], args[1], matched), leftName, left, rightName, right)
		return matched, nil
	})
}

func rangeCondition(outside bool) Condition {
	return ConditionFunc(func(e *Evaluation, raw any) (bool, error) {
		args, err := argsOf(raw, 3)
		if err != nil {
			return false, err
		}
		value, name, vok, err := e.value(args[0], 0)
		if err != nil {
			return false, err
		}
		low, lowName, lok, err := e.value(args[1], 0)
		if err != nil {
			return false, err
		}
		high, highName, hok, err := e.value(args[2], 0)
		if err != nil {
			return false, err
		}
		if !vok || !lok || !hok {
			e.missing(name)
			e.missing(lowName)
			e.missing(highName)
			return false, nil
		}
		within := value.GreaterThanOrEqual(low) && value.LessThanOrEqual(high)
		matched := within
		if outside {
			matched = !within
		}
		e.record("range condition", name, value, lowName, low, highName, high)
		return matched, nil
	})
}

func crossCondition(above bool) Condition {
	return ConditionFunc(func(e *Evaluation, raw any) (bool, error) {
		args, err := argsOf(raw, 2)
		if err != nil {
			return false, err
		}
		current, name, ok, err := e.value(args[0], 0)
		if err != nil {
			return false, err
		}
		previous, _, previousOK, err := e.value(args[0], 1)
		if err != nil {
			return false, err
		}
		threshold, thresholdName, thresholdOK, err := e.value(args[1], 0)
		if err != nil {
			return false, err
		}
		if !ok || !previousOK || !thresholdOK {
			e.missing(name)
			e.missing(thresholdName)
			return false, nil
		}
		matched := previous.LessThanOrEqual(threshold) && current.GreaterThan(threshold)
		if !above {
			matched = previous.GreaterThanOrEqual(threshold) && current.LessThan(threshold)
		}
		e.record("cross condition", name, current, thresholdName, threshold)
		return matched, nil
	})
}

func trendCondition(rising bool) Condition {
	return ConditionFunc(func(e *Evaluation, raw any) (bool, error) {
		args, err := argsOf(raw, 2)
		if err != nil {
			return false, err
		}
		barsValue, _, valid, err := e.value(args[1], 0)
		if err != nil {
			return false, err
		}
		if !valid || !barsValue.IsInteger() || barsValue.LessThan(decimal.NewFromInt(1)) {
			return false, fmt.Errorf("trend bars must be a positive integer")
		}
		bars := int(barsValue.IntPart())
		previous, _, ok, err := e.value(args[0], bars)
		if err != nil {
			return false, err
		}
		if !ok {
			e.missing(fmt.Sprint(args[0]))
			return false, nil
		}
		matched := true
		for offset := bars - 1; offset >= 0; offset-- {
			current, name, found, err := e.value(args[0], offset)
			if err != nil {
				return false, err
			}
			if !found {
				e.missing(name)
				return false, nil
			}
			if (rising && !current.GreaterThan(previous)) || (!rising && !current.LessThan(previous)) {
				matched = false
			}
			previous = current
		}
		return matched, nil
	})
}

func withinLastCondition(e *Evaluation, raw any) (bool, error) {
	value, ok := raw.(map[string]any)
	if !ok {
		return false, fmt.Errorf("within_last expects an object")
	}
	barsValue, ok := value["bars"]
	if !ok {
		return false, fmt.Errorf("within_last.bars is required")
	}
	bars, err := decimalValue(barsValue, decimal.Zero)
	if err != nil || !bars.IsInteger() || !bars.GreaterThan(decimal.Zero) {
		return false, fmt.Errorf("within_last.bars must be a positive integer")
	}
	conditionMap, ok := value["cond"].(map[string]any)
	if !ok {
		return false, fmt.Errorf("within_last.cond must be a condition")
	}
	for offset := 0; offset < int(bars.IntPart()); offset++ {
		child := *e
		child.currentAt = offset
		child.Missing = nil
		matched, err := e.Registry.Evaluate(&child, conditionMap)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
		if len(child.Missing) > 0 {
			e.Missing = append(e.Missing, child.Missing...)
		}
	}
	if len(e.Missing) > 0 {
		e.NotReady = true
	}
	return false, nil
}

func logicalCondition(all bool) Condition {
	return ConditionFunc(func(e *Evaluation, raw any) (bool, error) {
		args, ok := raw.([]any)
		if !ok {
			return false, fmt.Errorf("logical operator expects a list")
		}
		if len(args) == 0 {
			return all, nil
		}
		for _, arg := range args {
			child, ok := arg.(map[string]any)
			if !ok {
				return false, fmt.Errorf("nested condition must be an object")
			}
			matched, err := e.Registry.Evaluate(e, child)
			if err != nil {
				return false, err
			}
			if all && !matched {
				return false, nil
			}
			if !all && matched {
				return true, nil
			}
		}
		return all, nil
	})
}

func notCondition(e *Evaluation, raw any) (bool, error) {
	condition, ok := raw.(map[string]any)
	if !ok {
		return false, fmt.Errorf("not expects a condition object")
	}
	matched, err := e.Registry.Evaluate(e, condition)
	return !matched, err
}

func argsOf(raw any, count int) ([]any, error) {
	args, ok := raw.([]any)
	if !ok || len(args) != count {
		return nil, fmt.Errorf("expected %d operands", count)
	}
	return args, nil
}

func (e *Evaluation) missing(name string) {
	if name == "" {
		return
	}
	e.NotReady = true
	for _, existing := range e.Missing {
		if existing == name {
			return
		}
	}
	e.Missing = append(e.Missing, name)
}

func (e *Evaluation) record(description string, values ...any) {
	e.Reason.Matched = append(e.Reason.Matched, description)
	if e.Reason.Values == nil {
		e.Reason.Values = make(map[string]decimal.Decimal)
	}
	for i := 0; i+1 < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok || key == "" {
			continue
		}
		value, ok := values[i+1].(decimal.Decimal)
		if ok {
			e.Reason.Values[key] = value
		}
	}
}
