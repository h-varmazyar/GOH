package strategies

// import (
// 	"github.com/h-varmazyar/gatec/pkg/indicators"
// 	"github.com/shopspring/decimal"
// )

// type DefaultStrategy struct {
// 	ema50ID      uint
// 	ema200ID     uint
// 	adxID        uint
// 	stochasticID uint
// 	rsiID        uint
// }

// func (d DefaultStrategy) ExtractFeatures(ctx MarketContext) Features {

// 	trend := d.computeTrend(ctx)
// 	momentum := d.computeMomentum(ctx)
// 	volatility := d.computeVolatility(ctx)
// 	volume := d.computeVolume(ctx)

// 	return Features{
// 		TrendScore:      trend,
// 		MomentumScore:   momentum,
// 		VolatilityScore: volatility,
// 		VolumeScore:     volume,
// 	}
// }

// func (d DefaultStrategy) computeTrend(ctx MarketContext) float64 {

// 	ema50 := ctx.Indicators[d.ema50ID]
// 	ema200 := ctx.Indicators[d.ema200ID]
// 	adx := ctx.Indicators[d.adxID]

// 	score := 0.0

// 	if (ema50.Value).(*indicators.EMAValue).EMA.GreaterThan((ema200.Value).(*indicators.EMAValue).EMA) {
// 		score += 0.5
// 	} else {
// 		score -= 0.5
// 	}

// 	if ctx.Close.GreaterThan((ema50.Value).(*indicators.EMAValue).EMA) {
// 		score += 0.2
// 	}

// 	if (adx.Value).(*indicators.ADXValue).ADX.GreaterThan(decimal.NewFromInt(25)) {
// 		score += 0.3
// 	}

// 	return clamp(score, -1, 1)
// }

// func (d DefaultStrategy) computeMomentum(ctx MarketContext) float64 {
// 	stochastic := ctx.Indicators[d.stochasticID]
// 	rsi := ctx.Indicators[d.rsiID]

// 	score := 0.0

// 	// oversold
// 	if k < 20 && dv < 20 {
// 		score += (20 - k) / 20 * 0.4
// 	}

// 	// bullish cross
// 	if k > dv {
// 		score += 0.2
// 	} else {
// 		score -= 0.2
// 	}

// 	// RSI
// 	if rsi < 30 {
// 		score += 0.3
// 	} else if rsi > 70 {
// 		score -= 0.3
// 	}

// 	return clamp(score, -1, 1)
// }

// func (d DefaultStrategy) computeVolatility(ctx MarketContext) float64 {

// 	atr := i.Get("atr")

// 	if atr == 0 {
// 		return 0
// 	}

// 	// normalize (example)
// 	return clamp((atr-ctx.State.ATRMean)/ctx.State.ATRStd, -1, 1)
// }

// func (d DefaultStrategy) computeVolume(ctx MarketContext) float64 {

// 	vol := ctx.Event.Candle.Volume
// 	avg := i.Get("volume_sma_20")

// 	if avg == 0 {
// 		return 0
// 	}

// 	ratio := vol / avg

// 	return clamp((ratio-1)/2, -1, 1)
// }

// func (d DefaultStrategy) DetectRegime(ctx MarketContext, f Features) Regime {

// 	i := ctx.Indicators

// 	adx := i.Get("adx")
// 	ema50 := i.Get("ema_50")
// 	ema200 := i.Get("ema_200")

// 	if adx < 15 {
// 		return RegimeDead
// 	}

// 	if adx > 25 {
// 		if ema50 > ema200 {
// 			return RegimeTrendingBull
// 		}
// 		return RegimeTrendingBear
// 	}

// 	if f.VolatilityScore > 0.7 {
// 		return RegimeVolatile
// 	}

// 	return RegimeRanging
// }

// func (d DefaultStrategy) GenerateSignal(
// 	ctx MarketContext,
// 	f Features,
// 	r Regime,
// ) Signal {

// 	weights := d.getWeights(r)

// 	score :=
// 		f.TrendScore*weights.Trend +
// 			f.MomentumScore*weights.Momentum +
// 			f.VolatilityScore*weights.Volatility +
// 			f.VolumeScore*weights.Volume

// 	return Signal{
// 		Score: clamp(score, -1, 1),
// 	}
// }

// func (d DefaultStrategy) getWeights(r Regime) Weights {

// 	switch r {

// 	case RegimeTrendingBull:
// 		return Weights{
// 			Trend: 0.4, Momentum: 0.3,
// 			Volatility: 0.1, Volume: 0.2,
// 		}

// 	case RegimeRanging:
// 		return Weights{
// 			Trend: 0.1, Momentum: 0.5,
// 			Volatility: 0.2, Volume: 0.2,
// 		}
// 	}

// 	return Weights{}
// }

// func (d DefaultStrategy) EvaluateRisk(ctx MarketContext, s Signal) RiskResult {

// 	if math.Abs(s.Score) < 0.5 {
// 		return RiskResult{AllowTrade: false}
// 	}

// 	if ctx.State.OpenPosition {
// 		return RiskResult{AllowTrade: false}
// 	}

// 	return RiskResult{AllowTrade: true}
// }
