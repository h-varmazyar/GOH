package strategies

type Features struct {
	TrendScore      float64
	MomentumScore   float64
	VolatilityScore float64
	VolumeScore     float64
}

type Regime string

const RegimeUnknown Regime = "unknown"

type Signal struct {
	Score     float64
	Direction string
}

type RiskResult struct{ AllowTrade bool }

type SymbolState struct {
	OpenPosition bool
	LastClose    float64
}

type IndicatorMemory struct{}
