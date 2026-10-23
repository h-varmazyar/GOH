package indicators

import "errors"

var (
	ErrMissingIndicatorCfg     = errors.New("Indicator configs required")
	ErrInvalidPeriod           = errors.New("Invalid period")
	ErrInvalidSlopePeriod      = errors.New("Invalid EMA slope period")
	ErrInvalidSmoothingPeriod  = errors.New("Invalid smoothing period")
	ErrInvalidMultiplier       = errors.New("Invalid multiplier")
	ErrInvalidSource           = errors.New("Invalid indicator source")
	ErrInvalidValueStorageSize = errors.New("StoredValueCount has invalid value for this indicator")
)
