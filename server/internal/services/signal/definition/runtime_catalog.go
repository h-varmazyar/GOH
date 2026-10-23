package definition

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
)

// RuntimeCatalog resolves strategy market aliases to configured providers and
// reads available markets, timeframes, and indicators from worker repositories.
// Indicator availability is never maintained as a separate hard-coded list.
type RuntimeCatalog struct {
	Markets     *repositories.MarketRepository
	Indicators  *repositories.IndicatorDefinitionRepository
	ProviderIDs map[string]uint
}

func (c RuntimeCatalog) HasMarket(ctx context.Context, market string) (bool, error) {
	id, ok := c.ProviderIDs[market]
	if !ok {
		return false, nil
	}
	markets, err := c.Markets.List(ctx, map[string]any{"provider_id = ?": id})
	return err == nil && len(markets) > 0, err
}
func (c RuntimeCatalog) HasTimeframe(ctx context.Context, market, timeframe string) (bool, error) {
	id, ok := c.ProviderIDs[market]
	if !ok {
		return false, nil
	}
	markets, err := c.Markets.List(ctx, map[string]any{"provider_id = ?": id})
	if err != nil {
		return false, err
	}
	for _, m := range markets {
		for _, tf := range m.TimeFrames {
			if tf.Label == timeframe {
				return true, nil
			}
		}
	}
	return false, nil
}
func (c RuntimeCatalog) Has(ctx context.Context, market, timeframe, key string) (bool, error) {
	_, err := c.find(ctx, market, timeframe, key)
	if err == errNotConfigured {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
func (c RuntimeCatalog) Warmup(ctx context.Context, market, timeframe, key string) (int, error) {
	row, err := c.find(ctx, market, timeframe, key)
	if err != nil {
		return 0, err
	}
	params := map[string]any{}
	_ = json.Unmarshal(row.Params, &params)
	period := firstNumber(params, "length", "period", "window")
	if period < 1 {
		period = 1
	}
	switch strings.ToLower(row.Type) {
	case "rsi":
		return period + 1, nil
	case "atr_pct", "atr_percent", "mfi", "roc":
		return period + 1, nil
	case "donchian":
		return period, nil
	case "candle_features":
		return 1, nil
	case "ema_slope":
		return firstNumber(params, "ema_period", "emaPeriod", "length", "period") + firstNumber(params, "slope_period", "slopePeriod"), nil
	case "rvi":
		return period + firstNumber(params, "smoothing_period", "smoothingPeriod"), nil
	case "macd":
		return firstNumber(params, "slow_period", "slowPeriod") + firstNumber(params, "signal_period", "signalPeriod"), nil
	case "stochastic":
		return period + firstNumber(params, "smooth_k", "smoothK") + firstNumber(params, "smooth_d", "smoothD"), nil
	case "volume_spike":
		return period + 1, nil
	default:
		return period, nil
	}
}
func (c RuntimeCatalog) find(ctx context.Context, market, timeframe, key string) (*models.IndicatorDefinition, error) {
	providerID, ok := c.ProviderIDs[market]
	if !ok {
		return nil, errNotConfigured
	}
	markets, err := c.Markets.List(ctx, map[string]any{"provider_id = ?": providerID})
	if err != nil {
		return nil, err
	}
	marketIDs := map[uint]map[uint]struct{}{}
	for _, m := range markets {
		for _, tf := range m.TimeFrames {
			if tf.Label == timeframe {
				if marketIDs[m.ID] == nil {
					marketIDs[m.ID] = map[uint]struct{}{}
				}
				marketIDs[m.ID][tf.ID] = struct{}{}
			}
		}
	}
	defs, err := c.Indicators.List(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].ID < defs[j].ID })
	for i := range defs {
		d := &defs[i]
		if !configuredIndicatorMatches(*d, key) {
			continue
		}
		if _, ok := marketIDs[uint(d.MarketID)][d.TimeframeID]; ok {
			return d, nil
		}
		if d.MarketID == 0 {
			for _, tfs := range marketIDs {
				if _, ok := tfs[d.TimeframeID]; ok {
					return d, nil
				}
			}
		}
	}
	return nil, errNotConfigured
}

func configuredIndicatorMatches(definition models.IndicatorDefinition, key string) bool {
	if !configuredTypeMatches(definition.Type, key) {
		return false
	}
	_, _, ok := indicatorName(key, "1m")
	if !ok {
		return true
	}
	base := strings.TrimSuffix(key, lastDigits(key))
	digits := lastDigits(key)
	if digits == "" {
		return true
	}
	base = strings.TrimSuffix(base, "_")
	requested, err := strconv.Atoi(digits)
	if err != nil {
		return true
	}
	params := map[string]any{}
	_ = json.Unmarshal(definition.Params, &params)
	configured := firstNumber(params, "length", "period", "window", "ema_period", "emaPeriod")
	return configured == requested
}
func configuredTypeMatches(configured, key string) bool {
	a := canonicalIndicatorType(configured)
	b := canonicalIndicatorType(key)
	if a == b {
		return true
	}
	if n, err := strconv.Atoi(lastDigits(key)); err == nil && n > 0 {
		return strings.HasPrefix(a, b[:len(b)-len(lastDigits(key))])
	}
	return false
}

func canonicalIndicatorType(value string) string {
	value = strings.ToLower(strings.ReplaceAll(value, "_", ""))
	switch value {
	case "bollingerbands":
		return "bollinger"
	case "atrpercent":
		return "atrpct"
	case "stochastics":
		return "stochastic"
	case "averagedvolume":
		return "averagevolume"
	case "volumespike":
		return "volumespike"
	case "emadistance":
		return "emadistance"
	case "emaslope":
		return "emaslope"
	}
	return value
}
func lastDigits(s string) string {
	i := len(s)
	for i > 0 && s[i-1] >= '0' && s[i-1] <= '9' {
		i--
	}
	return s[i:]
}
func number(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), n > 0
	case int:
		return n, n > 0
	case json.Number:
		i, e := n.Int64()
		return int(i), e == nil && i > 0
	}
	return 0, false
}

func firstNumber(params map[string]any, names ...string) int {
	for _, name := range names {
		if n, ok := number(params[name]); ok {
			return n
		}
	}
	return 0
}

var errNotConfigured = &catalogError{"indicator is not configured"}

type catalogError struct{ message string }

func (e *catalogError) Error() string { return e.message }
