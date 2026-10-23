package consts

import "time"

type Timeframe string

const (
	TF1m  Timeframe = "1m"
	TF3m  Timeframe = "3m"
	TF5m  Timeframe = "5m"
	TF15m Timeframe = "15m"
	TF1h  Timeframe = "1h"
	TF4h  Timeframe = "4h"
	TF1d  Timeframe = "1d"
)

func (tf Timeframe) Duration() time.Duration {
	switch tf {
	case TF1m:
		return time.Minute
	case TF3m:
		return 3 * time.Minute
	case TF5m:
		return 5 * time.Minute
	case TF15m:
		return 15 * time.Minute
	case TF1h:
		return time.Hour
	case TF4h:
		return 4 * time.Hour
	case TF1d:
		return 24 * time.Hour
	}
	return 0
}

func (t Timeframe) String() string {
	return string(t)
}
