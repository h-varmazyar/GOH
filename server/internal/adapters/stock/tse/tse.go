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

type Instrument struct {
	Name     string                  `json:"name"`
	Label    string                  `json:"label"`
	Type     string                  `json:"type"`
	ID       string                  `json:"id"`
	Group    string                  `json:"group"`
	SubGroup string                  `json:"sub_group"`
	Status   consts.InstrumentStatus `json:"status"`
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
