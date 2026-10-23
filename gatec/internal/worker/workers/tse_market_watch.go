package workers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/adapters/stock/tse"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/internal/worker/events"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

const tseMarketWatchWorkerName = "tse-market-watch"

type TSEMarketWatch struct {
	provider         *tse.Adapter
	marketRepo       *repositories.TSEMarketRepository
	eventBus         events.CandleBus
	startMinute      int
	endMinute        int
	pollInterval     time.Duration
	location         *time.Location
	cancel           context.CancelFunc
	mu               sync.Mutex
	heven            int64
	ref              int64
	lastPoll         time.Time
	candleStorage    map[string][]models.Candle
	marketsByInsCode map[string]*models.TSEMarket
	logger           *logrus.Entry
}

func NewTSEMarketWatch(cfg *configs.TSEMarketWatch, provider *tse.Adapter, marketRepo *repositories.TSEMarketRepository, eventBus events.CandleBus) *TSEMarketWatch {
	logrus.Info("creating tse market watch worker")
	startHour, startMin, endHour, endMin, interval := 8, 0, 16, 0, 30*time.Second
	if cfg != nil {
		startHour, startMin, endHour, endMin = cfg.StartHour, cfg.StartMin, cfg.EndHour, cfg.EndMin
		if cfg.PollInterval > 0 {
			interval = cfg.PollInterval
		}
	}
	location, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		location = time.FixedZone("Asia/Tehran", 3*60*60+30*60)
	}
	return &TSEMarketWatch{
		provider:         provider,
		marketRepo:       marketRepo,
		startMinute:      startHour*60 + startMin,
		endMinute:        endHour*60 + endMin,
		pollInterval:     interval,
		location:         location,
		candleStorage:    make(map[string][]models.Candle),
		marketsByInsCode: make(map[string]*models.TSEMarket),
		logger:           logrus.WithField("worker", tseMarketWatchWorkerName),
		eventBus:         eventBus,
	}
}

func (w *TSEMarketWatch) Name() string { return tseMarketWatchWorkerName }

func (w *TSEMarketWatch) Start(ctx context.Context) error {
	logrus.Info("starting tse market watch worker")
	markets, err := w.marketRepo.List(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to load TSE markets: %w", err)
	}
	for _, market := range markets {
		w.marketsByInsCode[market.InsCode] = market
	}

	ctx, w.cancel = context.WithCancel(ctx)
	go w.run(ctx)
	return nil
}

func (w *TSEMarketWatch) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}
	return nil
}

func (w *TSEMarketWatch) run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			now = now.In(w.location)
			fmt.Println(w.inWindow(now), w.lastPoll.IsZero(), now.Sub(w.lastPoll), w.pollInterval)
			if w.inWindow(now) && (w.lastPoll.IsZero() || now.Sub(w.lastPoll) >= w.pollInterval) {
				w.lastPoll = now
				w.poll(ctx)
			}
		}
	}
}

func (w *TSEMarketWatch) poll(ctx context.Context) {
	records, nextRef, err := w.provider.GetMarketWatch(ctx, w.heven, w.ref)
	if err != nil {
		w.logger.WithError(err).Error("failed to fetch TSE market watch")
		return
	}
	maxHeven := w.heven
	for _, record := range records {
		if value, parseErr := strconv.ParseInt(strings.TrimSpace(record.LastDateUpdate), 10, 64); parseErr == nil && value > maxHeven {
			maxHeven = value
		}
	}
	w.heven, w.ref = maxHeven, nextRef
	w.consumeRecords(ctx, records)
}

func (w *TSEMarketWatch) consumeRecords(ctx context.Context, records []tse.MarketWatchRecord) {
	for _, record := range records {
		candle, err := w.candleFromRecord(ctx, record)
		if err != nil {
			w.logger.WithError(err).WithField("ins_code", record.InsCode).Warn("invalid TSE market watch record")
			continue
		}
		w.mu.Lock()
		candles := w.candleStorage[record.InsCode]
		if len(candles) == 0 {
			candles = append(candles, candle)
		} else if candle.Time == candles[len(candles)-1].Time {
			candles[len(candles)-1] = candle
		} else if candle.Time > candles[len(candles)-1].Time {
			candles = append(candles, candle)
		} else {
			w.mu.Unlock()
			continue
		}
		w.candleStorage[record.InsCode] = candles
		w.mu.Unlock()
		if w.eventBus != nil {
			if err := w.eventBus.PublishCandle(ctx, candle); err != nil && ctx.Err() == nil {
				w.logger.WithError(err).Error("publish TSE watch candle")
			}
			continue
		}
	}
}

func (w *TSEMarketWatch) candleFromRecord(_ context.Context, record tse.MarketWatchRecord) (models.Candle, error) {
	var marketID uint
	if market, ok := w.marketsByInsCode[record.InsCode]; ok {
		marketID = uint(market.MarketID)
	}
	timestamp, err := strconv.ParseInt(strings.TrimSpace(record.LastDateUpdate), 10, 64)
	if err != nil {
		return models.Candle{}, fmt.Errorf("invalid last_date_update %q: %w", record.LastDateUpdate, err)
	}
	parse := func(value string) (decimal.Decimal, error) {
		return decimal.NewFromString(strings.ReplaceAll(strings.TrimSpace(value), ",", ""))
	}
	open, err := parse(record.First)
	if err != nil {
		return models.Candle{}, err
	}
	high, err := parse(record.MaxPrice)
	if err != nil {
		return models.Candle{}, err
	}
	low, err := parse(record.MinPrice)
	if err != nil {
		return models.Candle{}, err
	}
	close, err := parse(record.Close)
	if err != nil {
		return models.Candle{}, err
	}
	volume, err := parse(record.TradeVolume)
	if err != nil {
		return models.Candle{}, err
	}
	candle := models.Candle{
		MarketID: marketID,
		Time:     timestamp,
		Open:     open,
		High:     high,
		Low:      low,
		Close:    close,
		Volume:   volume,
	}
	return candle, nil
}

func (w *TSEMarketWatch) inWindow(now time.Time) bool {
	minute := now.Hour()*60 + now.Minute()
	if w.startMinute <= w.endMinute {
		return minute >= w.startMinute && minute < w.endMinute
	}
	return minute >= w.startMinute || minute < w.endMinute
}
