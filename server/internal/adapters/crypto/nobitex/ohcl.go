package nobitex

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/h-varmazyar/gatec/internal/adapters/crypto"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

type historicalCandles struct {
	Times []int64           `json:"t"`
	Open  []decimal.Decimal `json:"o"`
	High  []decimal.Decimal `json:"h"`
	Low   []decimal.Decimal `json:"l"`
	Close []decimal.Decimal `json:"c"`
}

type ohlc struct {
	Time   int64           `json:"t"`
	Open   decimal.Decimal `json:"o"`
	High   decimal.Decimal `json:"h"`
	Low    decimal.Decimal `json:"l"`
	Close  decimal.Decimal `json:"c"`
	Volume decimal.Decimal `json:"v"`
}

func (n *Adapter) GetCandles(key, market string, timeframe time.Duration) error {
	tf, ok := timeframeLabel[timeframe]
	if !ok {
		tf = timeframeLabel[time.Hour]
	}
	channel := fmt.Sprintf("public:candle-%v-%v", strings.ToUpper(strings.Replace(market, "-", "", 1)), tf)
	n.RegisterChannel(key, channel)

	err := n.wssPool.RegisterChannel(key, channel, n.candlePollingChan, true)
	if err != nil {
		logrus.WithError(err).Warnf("initial register failed for channel %s, it will be retried by watchdog", channel)
		return err
	}
	n.RegisterChannel(key, channel)

	return nil
}

func (n *Adapter) GetCandleHistory(market string, timeframe time.Duration, to time.Time, count int) ([]crypto.Candle, error) {
	pages := count / 500
	if count%500 != 0 {
		pages++
	}

	fmt.Println("getting candle history:", market, timeframe, to, count)

	candles := make([]crypto.Candle, 0)

	for page := 1; page <= pages; page++ {
		url := fmt.Sprintf(
			"%v/market/udf/history?symbol=%v&resolution=%v&to=%v&countback=%v&page=%v",
			n.cfg.APIBaseURL,
			strings.Replace(market, "-", "", -1),
			timeframeLabel[timeframe],
			to.Unix(),
			count,
			page,
		)
		resp, err := n.httlClient.Get(url)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		hc := new(historicalCandles)
		err = json.Unmarshal(body, &hc)
		if err != nil {
			return nil, err
		}

		for i := 0; i < len(hc.Times); i++ {
			candle := crypto.Candle{
				Time:  hc.Times[i],
				Open:  hc.Open[i],
				High:  hc.High[i],
				Low:   hc.Low[i],
				Close: hc.Close[i],
			}
			candles = append(candles, candle)
		}

	}

	sort.Slice(candles, func(i, j int) bool {
		return candles[i].Time < candles[j].Time
	})

	fmt.Println("candles got", len(candles))

	return candles, nil
}

func (n *Adapter) proccessCandleResponse(candleChan chan crypto.Candle) {
	lastReceiveAtByKey := make(map[string]time.Time)
	ticker := time.NewTicker(45 * time.Second)
	defer ticker.Stop()

	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("candle processing crashed: %v", r)
			n.wssPool.Shutdown()
		}
	}()

	for {
		select {
		case <-ticker.C:
			n.reconnectStaleCandleChannels(lastReceiveAtByKey)
		case asyncResponse := <-n.candlePollingChan:
			lastReceiveAtByKey[asyncResponse.Key] = time.Now()
			ohlc := new(ohlc)
			if err := json.Unmarshal([]byte(asyncResponse.Data), ohlc); err != nil {
				logrus.Errorf("failed to unmarshal: %v\n", err)
				continue
			}

			candle := crypto.Candle{
				Time:   ohlc.Time,
				Open:   ohlc.Open,
				High:   ohlc.High,
				Low:    ohlc.Low,
				Close:  ohlc.Close,
				Volume: ohlc.Volume,
			}
			candle.Key = asyncResponse.Key
			// fmt.Printf("%v: %v\n", asyncResponse.Key, candle)

			candleChan <- candle

		// TODO: implement correct scenario for stopping workflow
		case <-n.stopChan:
			return
		}
	}
}

func (n *Adapter) reconnectStaleCandleChannels(lastReceiveAtByKey map[string]time.Time) {
	channels := n.RegisteredChannels()
	if len(channels) == 0 {
		return
	}

	now := time.Now()
	for key, channel := range channels {
		lastReceiveAt, ok := lastReceiveAtByKey[key]
		if !ok {
			lastReceiveAtByKey[key] = now
			continue
		}

		if now.Sub(lastReceiveAt) < 90*time.Second {
			continue
		}

		logrus.Warnf("no candle data received for channel %s since %s, reconnecting only this channel", channel, lastReceiveAt.Format(time.RFC3339))
		if err := n.wssPool.ReconnectChannel(key, channel, n.candlePollingChan, true); err != nil {
			logrus.WithError(err).Errorf("failed to reconnect stale channel %s", channel)
			continue
		}

		lastReceiveAtByKey[key] = now
	}
}
