package events

import (
	"encoding/json"

	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/services/signal/domain"
	"github.com/shopspring/decimal"
)

const IndicatorSnapshotSubject = "market.candle.indicators"

type CandleSnapshot struct {
	Candle     models.Candle              `json:"candle"`
	Indicators map[string]decimal.Decimal `json:"indicators,omitempty"`
}

func EncodeSnapshot(snapshot CandleSnapshot) ([]byte, error) { return json.Marshal(snapshot) }
func DecodeSnapshot(data []byte) (CandleSnapshot, error) {
	var snapshot CandleSnapshot
	err := json.Unmarshal(data, &snapshot)
	return snapshot, err
}

func EncodeSignal(signal domain.Signal) ([]byte, error) { return json.Marshal(signal) }
