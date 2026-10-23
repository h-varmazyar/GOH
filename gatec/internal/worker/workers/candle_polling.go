package workers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/adapters/crypto"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/internal/worker/events"
	"github.com/sirupsen/logrus"
)

type CandlePolling struct {
	marketRepo *repositories.MarketRepository
	candleRepo *repositories.CandleRepository
	cfg        *configs.CandlePolling
	providers  []crypto.Crypto
	candleChan chan crypto.Candle
	eventBus   events.CandleBus

	mu            *sync.Mutex
	candleStorage map[string][]models.Candle // todo: key must be combination of market, timeframe and provider
}

func NewCandlePolling(cfg *configs.CandlePolling, marketRepo *repositories.MarketRepository, candleRepo *repositories.CandleRepository, providers []crypto.Crypto, candleChan chan crypto.Candle, eventBus events.CandleBus) *CandlePolling {
	cp := &CandlePolling{
		marketRepo:    marketRepo,
		candleRepo:    candleRepo,
		providers:     providers,
		cfg:           cfg,
		mu:            new(sync.Mutex),
		candleChan:    candleChan,
		candleStorage: make(map[string][]models.Candle),
		eventBus:      eventBus,
	}
	return cp
}

func (w *CandlePolling) Name() string {
	return "candle-polling"
}

func (w *CandlePolling) Start(ctx context.Context) error {
	logrus.Info("starting candle polling")

	go w.consumeCandles(ctx)

	for _, provider := range w.providers {
		markets, err := w.marketRepo.List(ctx, map[string]any{
			"provider_id = ?": provider.GetID(),
			"status = ?":      consts.MarketStatusOpen,
		})
		if err != nil {
			logrus.WithError(err).Errorf("failed to get market list of %v", provider.GetID())
			continue
		}

		for _, market := range markets {
			for _, tf := range market.TimeFrames {
				if err = w.getRemoteCandles(ctx, provider, market, tf); err != nil {
					logrus.WithError(err).Errorf("failed to get candles for %v:%v:%v", provider.GetID(), market.Name, tf.Label)
				}
			}
		}
	}

	return nil
}

func (w *CandlePolling) Stop() error {
	return nil
}

func (w *CandlePolling) InsertMarket(ctx context.Context, provider crypto.Crypto, market *models.Market) {
	for _, tf := range market.TimeFrames {
		if err := w.getRemoteCandles(ctx, provider, market, tf); err != nil {
			logrus.WithError(err).Errorf("failed to insert market %v:%v:%v", provider.GetID(), market.Name, tf.Label)
		}
	}
}

func (w *CandlePolling) RemoveMarket(ctx context.Context, provider crypto.Crypto, market *models.Market) {
}

func (w *CandlePolling) getKey(providerID, marketID, timeframeID uint) string {
	return fmt.Sprintf("%v:%v:%v", providerID, marketID, timeframeID)
}

func (w *CandlePolling) getRemoteCandles(ctx context.Context, provider crypto.Crypto, market *models.Market, tf models.Timeframe) error {
	historicalCandles, err := provider.GetCandleHistory(market.Name, tf.Duration, time.Now(), 3000)
	if err != nil {
		logrus.WithError(err).Errorf("failed to get candles of %v", market.Name)
		return err
	}

	candles := make([]models.Candle, 0)

	for _, hc := range historicalCandles {
		c := models.Candle{
			Time:        hc.Time,
			Open:        hc.Open,
			High:        hc.High,
			Low:         hc.Low,
			Close:       hc.Close,
			Volume:      hc.Volume,
			MarketID:    market.ID,
			Market:      market,
			Timeframe:   &tf,
			TimeframeID: tf.ID,
		}
		candles = append(candles, c)
	}

	for _, candle := range candles {
		if err = w.candleRepo.Create(ctx, &candle); err != nil {
			if !strings.Contains(err.Error(), "SQLSTATE 23505") {
				continue
			}
			candles, err = w.candleRepo.List(ctx, market.ID, tf.ID, 0)
			if err != nil {
				return err
			}
		}
	}

	key := w.getKey(provider.GetID(), market.ID, tf.ID)
	w.candleStorage[key] = candles
	if err = provider.GetCandles(key, market.Name, tf.Duration); err != nil {
		return err
	}
	logrus.Infof("getting candles for %v", key)

	return nil
}

func (w *CandlePolling) consumeCandles(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case candle, ok := <-w.candleChan:
			if !ok {
				return
			}
			newCandle := models.Candle{
				Time:   candle.Time,
				Open:   candle.Open,
				High:   candle.High,
				Low:    candle.Low,
				Close:  candle.Close,
				Volume: candle.Volume,
			}
			logrus.Info(newCandle)
			w.mu.Lock()
			candles := w.candleStorage[candle.Key]
			if len(candles) == 0 {
				w.mu.Unlock()
				continue
			}

			if candle.Time > candles[len(candles)-1].Time {
				ids := strings.Split(candle.Key, ":")
				marketID, _ := strconv.ParseUint(ids[1], 10, 64)
				tfID, _ := strconv.ParseUint(ids[2], 10, 64)

				newCandle.MarketID = uint(marketID)
				newCandle.TimeframeID = uint(tfID)

				w.candleRepo.Create(ctx, &candles[len(candles)-1])
				tmpCandles := append(candles[1:], newCandle)
				candles = tmpCandles
			} else if candle.Time == candles[len(candles)-1].Time {
				newCandle.MarketID = candles[len(candles)-1].MarketID
				newCandle.TimeframeID = candles[len(candles)-1].TimeframeID
				candles[len(candles)-1] = newCandle
			}
			w.candleStorage[candle.Key] = candles
			w.mu.Unlock()
			if w.eventBus != nil {
				if err := w.eventBus.PublishCandle(ctx, newCandle); err != nil && ctx.Err() == nil {
					logrus.WithError(err).Error("publish candle event")
				}
				continue
			}
		}
	}
}
