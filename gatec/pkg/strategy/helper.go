package strategies

import "math"

func clamp(v, min, max float64) float64 {
	return math.Max(min, math.Min(max, v))
}
