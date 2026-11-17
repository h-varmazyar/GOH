package indicators

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	indicatorsPkg "github.com/h-varmazyar/gatec/pkg/indicators"
)

// Category groups the indicators in the panel.
type Category string

const (
	CategoryTrend       Category = "TREND"
	CategoryMomentum    Category = "MOMENTUM"
	CategoryVolatility  Category = "VOLATILITY"
	CategoryVolume      Category = "VOLUME"
	CategoryPriceAction Category = "PRICE_ACTION"
)

// CategoryInfo is a category with its label.
type CategoryInfo struct {
	Key   Category `json:"key"`
	Label string   `json:"label"`
}

var Categories = []CategoryInfo{
	{CategoryTrend, "روند"},
	{CategoryMomentum, "مومنتوم"},
	{CategoryVolatility, "نوسان"},
	{CategoryVolume, "حجم"},
	{CategoryPriceAction, "پرایس اکشن"},
}

// ParamKind is the kind of value a param takes.
type ParamKind string

const (
	ParamInt     ParamKind = "INT"
	ParamDecimal ParamKind = "DECIMAL"
	// ParamSource is one of the candle price sources of pkg/indicators.
	ParamSource ParamKind = "SOURCE"
)

// ParamOption is a choice of a SOURCE param.
type ParamOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Param describes one param of an indicator. Its key is the key the
// indicator calculator reads from the stored params.
type Param struct {
	Key         string        `json:"key"`
	Label       string        `json:"label"`
	Description string        `json:"description,omitempty"`
	Kind        ParamKind     `json:"kind"`
	Default     any           `json:"default"`
	Min         *float64      `json:"min,omitempty"`
	Max         *float64      `json:"max,omitempty"`
	Options     []ParamOption `json:"options,omitempty"`
}

// Definition describes an indicator of pkg/indicators that can be added.
type Definition struct {
	// Type is the type stored in indicator definitions.
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Category    Category `json:"category"`
	Params      []Param  `json:"params"`
	// Outputs are the value fields the indicator calculates.
	Outputs []string `json:"outputs"`
	// validate checks the params together, after each param is valid.
	validate func(params map[string]any) error
}

// MaxPeriod bounds the windows of indicators; the calculator warms up from
// the latest 500 candles.
const MaxPeriod = 400

func bound(v float64) *float64 { return &v }

var sourceOptions = []ParamOption{
	{indicatorsPkg.IndicatorSourceClose, "قیمت پایانی (Close)"},
	{indicatorsPkg.IndicatorSourceOpen, "قیمت باز شدن (Open)"},
	{indicatorsPkg.IndicatorSourceHigh, "بیشترین قیمت (High)"},
	{indicatorsPkg.IndicatorSourceLow, "کمترین قیمت (Low)"},
	{indicatorsPkg.IndicatorSourceHL2, "میانگین بیشترین و کمترین (HL2)"},
	{indicatorsPkg.IndicatorSourceHLC3, "قیمت معمول (HLC3)"},
	{indicatorsPkg.IndicatorSourceOHLC4, "میانگین چهار قیمت (OHLC4)"},
}

func periodParam(def int, label string) Param {
	return Param{Key: "period", Label: label, Kind: ParamInt, Default: def, Min: bound(1), Max: bound(MaxPeriod)}
}

func intParam(key, label string, def int, minValue, maxValue float64) Param {
	return Param{Key: key, Label: label, Kind: ParamInt, Default: def, Min: bound(minValue), Max: bound(maxValue)}
}

var sourceParam = Param{
	Key: "source", Label: "منبع قیمت", Kind: ParamSource,
	Default: indicatorsPkg.IndicatorSourceClose, Options: sourceOptions,
}

// catalog lists the indicators of pkg/indicators that the indicator
// calculator can build, in the order the panel shows them.
var catalog = []Definition{
	{
		Type: indicatorsPkg.IndicatorTypeSMA, Name: "SMA", Title: "میانگین متحرک ساده",
		Description: "میانگین قیمت پایانی در دوره.",
		Category:    CategoryTrend, Params: []Param{periodParam(20, "دوره")}, Outputs: []string{"sma"},
	},
	{
		Type: indicatorsPkg.IndicatorTypeEMA, Name: "EMA", Title: "میانگین متحرک نمایی",
		Description: "میانگین قیمت پایانی با وزن بیشتر برای کندل‌های اخیر.",
		Category:    CategoryTrend, Params: []Param{periodParam(20, "دوره")}, Outputs: []string{"ema"},
	},
	{
		Type: indicatorsPkg.IndicatorTypeEMASlope, Name: "EMA Slope", Title: "شیب میانگین نمایی",
		Description: "تغییر EMA نسبت به چند کندل قبل، به قیمت و به درصد.",
		Category:    CategoryTrend,
		Params: []Param{
			intParam("ema_period", "دوره EMA", 20, 1, MaxPeriod),
			intParam("slope_period", "فاصله شیب (کندل)", 5, 1, MaxPeriod),
			sourceParam,
		},
		Outputs: []string{"ema", "slope", "slope_percent"},
	},
	{
		Type: indicatorsPkg.IndicatorTypeEMADistance, Name: "EMA Distance", Title: "فاصله از میانگین نمایی",
		Description: "فاصله قیمت از EMA، به قیمت و به درصد.",
		Category:    CategoryTrend,
		Params:      []Param{intParam("ema_period", "دوره EMA", 20, 1, MaxPeriod), sourceParam},
		Outputs:     []string{"ema", "distance", "distance_percent"},
	},
	{
		Type: "MACD", Name: "MACD", Title: "مکدی",
		Description: "اختلاف دو EMA سریع و کند، با خط سیگنال و هیستوگرام.",
		Category:    CategoryTrend,
		Params: []Param{
			intParam("fast_period", "دوره سریع", 12, 1, MaxPeriod),
			intParam("slow_period", "دوره کند", 26, 2, MaxPeriod),
			intParam("signal_period", "دوره سیگنال", 9, 1, MaxPeriod),
		},
		Outputs: []string{"macd", "signal", "histogram"},
		validate: func(params map[string]any) error {
			if params["fast_period"].(int) >= params["slow_period"].(int) {
				return fmt.Errorf("%w: the fast period of MACD must be less than its slow period", ErrInvalidInput)
			}
			return nil
		},
	},
	{
		Type: "ADX", Name: "ADX", Title: "قدرت روند (ساده‌شده)",
		Description: "در پیاده‌سازی فعلی: دامنه هر کندل (بیشترین منهای کمترین) به درصد از قیمت پایانی.",
		Category:    CategoryTrend, Params: []Param{periodParam(14, "دوره")}, Outputs: []string{"adx"},
	},
	{
		Type: indicatorsPkg.IndicatorTypeDonchian, Name: "Donchian", Title: "کانال دانچیان",
		Description: "بیشترین سقف، کمترین کف و میانه آن‌ها در دوره.",
		Category:    CategoryTrend, Params: []Param{periodParam(20, "دوره")}, Outputs: []string{"upper", "lower", "mid"},
	},
	{
		Type: indicatorsPkg.IndicatorTypeRSI, Name: "RSI", Title: "شاخص قدرت نسبی",
		Description: "نسبت سودها به زیان‌های قیمت پایانی در دوره؛ بین ۰ تا ۱۰۰.",
		Category:    CategoryMomentum, Params: []Param{periodParam(14, "دوره")}, Outputs: []string{"rsi"},
	},
	{
		Type: indicatorsPkg.IndicatorTypeStochastic, Name: "Stochastic", Title: "استوکاستیک",
		Description: "جای قیمت پایانی در دامنه دوره، با خطوط K و D.",
		Category:    CategoryMomentum,
		Params: []Param{
			periodParam(14, "دوره"),
			intParam("smooth_k", "هموارسازی K", 3, 1, 50),
			intParam("smooth_d", "هموارسازی D", 3, 1, 50),
		},
		Outputs: []string{"indexK", "indexD"},
	},
	{
		Type: indicatorsPkg.IndicatorTypeROC, Name: "ROC", Title: "نرخ تغییر",
		Description: "درصد تغییر قیمت پایانی نسبت به چند کندل قبل.",
		Category:    CategoryMomentum, Params: []Param{periodParam(12, "دوره")}, Outputs: []string{"value"},
	},
	{
		Type: indicatorsPkg.IndicatorTypeRVI, Name: "RVI", Title: "شاخص نوسان نسبی",
		Description: "سهم انحراف معیار قیمت در کندل‌های صعودی؛ بین ۰ تا ۱۰۰.",
		Category:    CategoryMomentum,
		Params: []Param{
			periodParam(10, "دوره انحراف معیار"),
			intParam("smoothing_period", "دوره هموارسازی", 14, 1, MaxPeriod),
			sourceParam,
		},
		Outputs: []string{"rvi"},
	},
	{
		Type: indicatorsPkg.IndicatorTypeBollingerBands, Name: "Bollinger Bands", Title: "باندهای بولینگر",
		Description: "میانگین ساده با باندهایی به فاصله چند انحراف معیار، به همراه %B و پهنای باند.",
		Category:    CategoryVolatility,
		Params: []Param{
			periodParam(20, "دوره"),
			{Key: "deviation", Label: "ضریب انحراف معیار", Kind: ParamInt, Default: 2, Min: bound(1), Max: bound(5)},
		},
		Outputs: []string{"upper_band", "middle_band", "lower_band", "percent_b", "bandwidth"},
	},
	{
		Type: "ATR", Name: "ATR", Title: "میانگین دامنه واقعی",
		Description: "میانگین هموارشده دامنه واقعی کندل‌ها، به قیمت.",
		Category:    CategoryVolatility, Params: []Param{periodParam(14, "دوره")}, Outputs: []string{"atr"},
	},
	{
		Type: indicatorsPkg.IndicatorTypeATRPercent, Name: "ATR %", Title: "میانگین دامنه واقعی (درصدی)",
		Description: "ATR وایلدر به درصد از قیمت پایانی.",
		Category:    CategoryVolatility, Params: []Param{periodParam(14, "دوره")}, Outputs: []string{"value"},
	},
	{
		Type: indicatorsPkg.IndicatorTypeMFI, Name: "MFI", Title: "شاخص جریان پول",
		Description: "نسبت جریان پول مثبت به کل جریان پول در دوره؛ بین ۰ تا ۱۰۰.",
		Category:    CategoryVolume, Params: []Param{periodParam(14, "دوره")}, Outputs: []string{"value"},
	},
	{
		Type: indicatorsPkg.IndicatorTypeAverageVolume, Name: "Average Volume", Title: "میانگین حجم",
		Description: "میانگین حجم معاملات در دوره.",
		Category:    CategoryVolume, Params: []Param{periodParam(20, "دوره")}, Outputs: []string{"average_volume"},
	},
	{
		Type: indicatorsPkg.IndicatorTypeVolumeSpike, Name: "Volume Spike", Title: "جهش حجم",
		Description: "نسبت حجم کندل به میانگین حجم دوره؛ بیش از ضریب، جهش حجم است.",
		Category:    CategoryVolume,
		Params: []Param{
			periodParam(20, "دوره"),
			{Key: "multiplier", Label: "ضریب جهش", Kind: ParamDecimal, Default: 2.0, Min: bound(0.1), Max: bound(100)},
		},
		Outputs: []string{"volume", "average_volume", "ratio", "is_spike"},
	},
	{
		Type: "VWAP", Name: "VWAP", Title: "میانگین قیمت وزنی حجمی",
		Description: "میانگین قیمت معمول (HLC3) با وزن حجم، از ابتدای داده‌ها.",
		Category:    CategoryVolume, Params: []Param{}, Outputs: []string{"vwap"},
	},
	{
		Type: indicatorsPkg.IndicatorTypeCandleFeatures, Name: "Candle Features", Title: "ویژگی‌های کندل",
		Description: "درصد بدنه و سایه‌ها از دامنه کندل، دامنه به درصد قیمت و جای قیمت پایانی در دامنه.",
		Category:    CategoryPriceAction, Params: []Param{},
		Outputs: []string{"body_pct", "upper_wick_pct", "lower_wick_pct", "range_pct", "close_location"},
	},
}

// Catalog returns the indicators that can be added.
func Catalog() []Definition { return catalog }

// Lookup returns the catalog entry of a type.
func Lookup(indicatorType string) (*Definition, bool) {
	indicatorType = strings.ToUpper(strings.TrimSpace(indicatorType))
	for i := range catalog {
		if catalog[i].Type == indicatorType {
			return &catalog[i], true
		}
	}
	return nil, false
}

// NormalizeParams validates params against the definition and returns them
// with defaults for missing params: INT params as int, DECIMAL params as
// float64 and SOURCE params as an upper-case source.
func (d *Definition) NormalizeParams(params map[string]any) (map[string]any, error) {
	known := make(map[string]bool, len(d.Params))
	out := make(map[string]any, len(d.Params))
	for _, param := range d.Params {
		known[param.Key] = true
		raw, ok := params[param.Key]
		if !ok || raw == nil || raw == "" {
			out[param.Key] = param.Default
			continue
		}
		value, err := param.normalize(raw)
		if err != nil {
			return nil, err
		}
		out[param.Key] = value
	}
	for key := range params {
		if !known[key] {
			return nil, fmt.Errorf("%w: %s has no param %q", ErrInvalidInput, d.Name, key)
		}
	}
	if d.validate != nil {
		if err := d.validate(out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (p Param) normalize(raw any) (any, error) {
	if p.Kind == ParamSource {
		value, ok := raw.(string)
		source := indicatorsPkg.IndicatorSource(strings.ToUpper(strings.TrimSpace(value)))
		if !ok || !source.IsValidSource() {
			return nil, fmt.Errorf("%w: %s is not a price source", ErrInvalidInput, p.Key)
		}
		return string(source), nil
	}
	var number float64
	switch v := raw.(type) {
	case float64:
		number = v
	case int:
		number = float64(v)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return nil, fmt.Errorf("%w: %s must be a number", ErrInvalidInput, p.Key)
		}
		number = parsed
	default:
		return nil, fmt.Errorf("%w: %s must be a number", ErrInvalidInput, p.Key)
	}
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return nil, fmt.Errorf("%w: %s must be a number", ErrInvalidInput, p.Key)
	}
	if p.Min != nil && number < *p.Min {
		return nil, fmt.Errorf("%w: %s must be at least %v", ErrInvalidInput, p.Key, *p.Min)
	}
	if p.Max != nil && number > *p.Max {
		return nil, fmt.Errorf("%w: %s must be at most %v", ErrInvalidInput, p.Key, *p.Max)
	}
	if p.Kind == ParamInt {
		if number != math.Trunc(number) {
			return nil, fmt.Errorf("%w: %s must be a whole number", ErrInvalidInput, p.Key)
		}
		return int(number), nil
	}
	return number, nil
}
