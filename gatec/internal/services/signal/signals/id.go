package signals

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/h-varmazyar/gatec/internal/services/signal/domain"
)

// DeterministicID makes at-least-once processing idempotent for one candle event.
func DeterministicID(strategyID, market, symbol, timeframe string, open time.Time, kind domain.SignalType) string {
	value := strings.Join([]string{strategyID, market, symbol, timeframe, open.UTC().Format(time.RFC3339Nano), string(kind)}, "\x00")
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}
