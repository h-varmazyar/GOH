package nobitex

import (
	"net/http"
	"sync"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/adapters/crypto"
	connectionPool "github.com/h-varmazyar/gatec/pkg/connection_pool"
)

type Adapter struct {
	wssPool    *connectionPool.WebSocketPool
	httlClient *http.Client
	cfg        *configs.Nobitex
	stopChan   chan bool
	id         uint

	// response gather channels
	candlePollingChan chan connectionPool.AsyncResponse
	marketUpdateChan  chan connectionPool.AsyncResponse

	channelsLock sync.RWMutex
	channels     map[string]string
}

type baseResponse struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

var timeframeLabel = map[time.Duration]string{
	time.Minute:        "1",
	time.Minute * 5:    "5",
	time.Minute * 15:   "15",
	time.Minute * 30:   "30",
	time.Hour:          "60",
	time.Hour * 3:      "180",
	time.Hour * 4:      "240",
	time.Hour * 6:      "360",
	time.Hour * 12:     "720",
	time.Hour * 24:     "D",
	time.Hour * 24 * 2: "2D",
	time.Hour * 24 * 3: "3D",
}

var timeframeDuration = map[string]time.Duration{
	"1":   time.Minute,
	"5":   time.Minute * 5,
	"15":  time.Minute * 15,
	"30":  time.Minute * 30,
	"60":  time.Hour,
	"180": time.Hour * 3,
	"240": time.Hour * 4,
	"360": time.Hour * 6,
	"720": time.Hour * 12,
	"D":   time.Hour * 24,
	"2D":  time.Hour * 24 * 2,
	"3D":  time.Hour * 24 * 3,
}

func NewAdapter(cfg *configs.Nobitex, candleChan chan crypto.Candle, providerID uint) (*Adapter, error) {
	wssPool, err := connectionPool.NewWebSocketPool(cfg.Socket)
	if err != nil {
		return nil, err
	}

	n := &Adapter{
		wssPool:           wssPool,
		httlClient:        http.DefaultClient,
		cfg:               cfg,
		id:                providerID,
		candlePollingChan: make(chan connectionPool.AsyncResponse, 1000),
		marketUpdateChan:  make(chan connectionPool.AsyncResponse, 1000),
		stopChan:          make(chan bool),
		channels:          make(map[string]string),
	}

	go n.proccessCandleResponse(candleChan)

	return n, nil
}

func (n *Adapter) GetID() uint {
	return n.id
}

func (n *Adapter) RegisterChannel(key, channel string) {
	n.channelsLock.Lock()
	defer n.channelsLock.Unlock()
	n.channels[key] = channel
}

func (n *Adapter) RegisteredChannels() map[string]string {
	n.channelsLock.RLock()
	defer n.channelsLock.RUnlock()

	clone := make(map[string]string, len(n.channels))
	for k, v := range n.channels {
		clone[k] = v
	}

	return clone
}

func (n *Adapter) Shutdown() {
	select {
	case n.stopChan <- true:
	default:
	}
	n.wssPool.Shutdown()
}
