package tse

import (
	"net/http"
	"sync"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/adapters/crypto"
	"github.com/h-varmazyar/gatec/internal/consts"
	connectionPool "github.com/h-varmazyar/gatec/pkg/connection_pool"
	"github.com/shopspring/decimal"
)

type Candle struct {
	Key    string          `json:"key"`
	Time   int64           `json:"time"`
	Open   decimal.Decimal `json:"open"`
	High   decimal.Decimal `json:"high"`
	Low    decimal.Decimal `json:"low"`
	Close  decimal.Decimal `json:"close"`
	Volume decimal.Decimal `json:"volume"`
}

// Instrument is a stock, fund or right of the Tehran Stock Exchange or Iran Fara
// Bourse, including delisted ones.
type Instrument struct {
	// InsCode is the TSETMC internal code used by the market data endpoints.
	InsCode string `json:"ins_code"`
	// InstrumentID is the ISIN that starts with "IR".
	InstrumentID   string                   `json:"instrument_id"`
	Symbol         string                   `json:"symbol"`
	Name           string                   `json:"name"`
	MarketType     consts.TSEMarketType     `json:"market_type"`
	Kind           consts.TSEInstrumentKind `json:"kind"`
	InstrumentType int                      `json:"instrument_type"`
	Group          string                   `json:"group"`
	GroupName      string                   `json:"group_name"`
	SubGroup       string                   `json:"sub_group"`
	// Delisted is true when TSETMC no longer lists the instrument.
	Delisted bool `json:"delisted"`
	// InsertDate is the date the symbol was inserted on the trading board (تاریخ درج).
	InsertDate *time.Time `json:"insert_date,omitempty"`
	// ReleaseDate is the initial public offering date (تاریخ عرضه).
	ReleaseDate *time.Time `json:"release_date,omitempty"`
	// AcceptanceDate is the listing acceptance date (تاریخ پذیرش).
	AcceptanceDate *time.Time `json:"acceptance_date,omitempty"`
	// Info is the TSETMC instrument info when it was fetched while building the
	// instrument list; it is nil for instruments whose ID was already known.
	Info *InstrumentInfo `json:"-"`
}

type Adapter struct {
	httlClient *http.Client
	cfg        *configs.TSE
	stopChan   chan bool
	id         uint

	// response gather channels
	marketUpdateChan chan connectionPool.AsyncResponse

	channelsLock sync.RWMutex
	channels     map[string]string
}

func NewAdapter(cfg *configs.TSE, candleChan chan crypto.Candle, providerID uint) (*Adapter, error) {
	n := &Adapter{
		httlClient:       &http.Client{Timeout: 30 * time.Second},
		cfg:              cfg,
		id:               providerID,
		marketUpdateChan: make(chan connectionPool.AsyncResponse, 1000),
		stopChan:         make(chan bool),
		channels:         make(map[string]string),
	}

	return n, nil
}

func newAdapterWithClient(client *http.Client, cfg *configs.TSE, providerID uint) *Adapter {
	adapter, _ := NewAdapter(cfg, nil, providerID)
	if client != nil {
		adapter.httlClient = client
	}
	return adapter
}

func (n *Adapter) GetID() uint {
	return n.id
}

func (n *Adapter) Shutdown() {
	select {
	case n.stopChan <- true:
	default:
	}
}
