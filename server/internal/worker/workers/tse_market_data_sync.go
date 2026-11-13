package workers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/adapters/stock/tse"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/internal/services/tradehistory"
	"github.com/h-varmazyar/gatec/internal/worker/events"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

type TSEMarketDataSync struct {
	adapter          *tse.Adapter
	marketRepo       *repositories.TSEMarketRepository
	dailyRepo        *repositories.TSEDailyClosingPriceRepository
	tradeHistory     *tradehistory.Service
	candleRepo       *repositories.CandleRepository
	requestInterval  time.Duration
	config           *configs.TSEMarketDataSync
	logger           *logrus.Entry
	eventBus         events.CandleBus
	receivedData     atomic.Int64
}

func NewTSEMarketDataSync(adapter *tse.Adapter, cfg *configs.TSEMarketDataSync, marketRepo *repositories.TSEMarketRepository, dailyRepo *repositories.TSEDailyClosingPriceRepository, tradeHistory *tradehistory.Service) *TSEMarketDataSync {
	return NewTSEMarketDataSyncWithBus(adapter, cfg, marketRepo, dailyRepo, nil, tradeHistory)
}

func NewTSEMarketDataSyncWithBus(adapter *tse.Adapter, cfg *configs.TSEMarketDataSync, marketRepo *repositories.TSEMarketRepository, dailyRepo *repositories.TSEDailyClosingPriceRepository, eventBus events.CandleBus, tradeHistory *tradehistory.Service) *TSEMarketDataSync {
	if cfg.Mode == "" {
		cfg.Mode = consts.TSEMarketDataToday
	}
	if cfg.RequestInterval < 0 {
		cfg.RequestInterval = 0
	}
	return &TSEMarketDataSync{
		adapter:          adapter,
		marketRepo:       marketRepo,
		dailyRepo:        dailyRepo,
		requestInterval:  cfg.RequestInterval,
		config:           cfg,
		logger:           logrus.WithField("worker", "tse-market-data-sync"),
		eventBus:         eventBus,
		tradeHistory:     tradeHistory,
	}
}

func (w *TSEMarketDataSync) Name() string { return "tse-market-data-sync" }

func (w *TSEMarketDataSync) Stop() error { return nil }

func (w *TSEMarketDataSync) Start(ctx context.Context) error {
	if w.config.Mode != consts.TSEMarketDataToday {
		return w.syncAllMarkets(ctx)
	}

	location, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		return fmt.Errorf("load Asia/Tehran timezone: %w", err)
	}
	for {
		nextRun := nextDailyRun(time.Now(), location, w.config.ScheduleHour, w.config.ScheduleMin)
		wait := time.Until(nextRun)
		w.logger.WithField("next_run", nextRun.Format(time.RFC3339)).Info("scheduled next TSE market data sync")
		if !w.waitDuration(ctx, wait) {
			return ctx.Err()
		}
		if err := w.syncAllMarkets(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			w.logger.WithError(err).Error("daily TSE market data sync failed")
		}
	}
}

func (w *TSEMarketDataSync) syncAllMarkets(ctx context.Context) error {
	logrus.Info("starting tse market data sync")
	markets, err := w.marketRepo.ListDataCollection(ctx)
	if err != nil {
		return fmt.Errorf("list TSE markets: %w", err)
	}
	switch w.config.CollectionMethod {
	case "ins_codes":
		wanted := make(map[string]bool, len(w.config.InsCodes))
		for _, code := range w.config.InsCodes {
			wanted[strings.TrimSpace(code)] = true
		}
		filtered := markets[:0]
		for _, market := range markets {
			if wanted[market.InsCode] {
				filtered = append(filtered, market)
			}
		}
		markets = filtered
	case "missed":
		from, to := w.collectionBounds()
		known, err := w.dailyRepo.MarketsWithPrices(ctx, from, to)
		if err != nil {
			return fmt.Errorf("list TSE markets with closing prices: %w", err)
		}
		synced := make(map[uint]bool, len(known))
		for _, id := range known {
			synced[id] = true
		}
		filtered := markets[:0]
		for _, market := range markets {
			if !synced[market.ID] {
				filtered = append(filtered, market)
			}
		}
		markets = filtered
	case "all":
		// continue
	default:
		return fmt.Errorf("please set collection method")
	}
	for index, market := range markets {
		if err := ctx.Err(); err != nil {
			return err
		}
		if market.InsCode == "" {
			continue
		}
		logrus.Infof("sync market %v of %v", index, len(markets))
		if err := w.syncMarket(ctx, market); err != nil {
			w.logger.WithError(err).WithField("ins_code", market.InsCode).Error("TSE market sync failed")
		}
		if index < len(markets)-1 && !w.wait(ctx) {
			return ctx.Err()
		}
	}
	logrus.Info("#---------- all market synced ----------#")
	if w.config.Mode == consts.TSEMarketDataAll {
		w.logger.WithField("received_data_count", w.receivedData.Load()).Info("TSE market data sync completed")
	}
	return nil
}

func (w *TSEMarketDataSync) collectionBounds() (time.Time, time.Time) {
	if w.config.Mode == consts.TSEMarketDataToday {
		location, err := time.LoadLocation("Asia/Tehran")
		if err != nil {
			location = time.UTC
		}
		now := time.Now().In(location)
		from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
		return from, from.AddDate(0, 0, 1)
	}
	from, to := w.bounds()
	if !to.IsZero() {
		to = to.AddDate(0, 0, 1)
	}
	return from, to
}

func nextDailyRun(now time.Time, location *time.Location, hour, minute int) time.Time {
	localNow := now.In(location)
	next := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), hour, minute, 0, 0, location)
	if !next.After(localNow) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func (w *TSEMarketDataSync) waitDuration(ctx context.Context, duration time.Duration) bool {
	if duration <= 0 {
		return true
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (w *TSEMarketDataSync) syncMarket(ctx context.Context, market *models.TSEMarket) error {
	if w.config.Mode == consts.TSEMarketDataToday {
		date, err := w.syncToday(ctx, market)
		if err != nil || date.IsZero() {
			return err
		}
		return w.syncTradeHistory(ctx, market, date)
	}

	prices, err := w.adapter.FetchClosingPriceHistory(ctx, market.InsCode)
	if err != nil {
		return fmt.Errorf("fetch closing price history: %w", err)
	}

	participants, err := w.adapter.FetchClientTypeHistory(ctx, market.InsCode)
	if err != nil {
		return fmt.Errorf("fetch client type history: %w", err)
	}

	return errors.Join(
		w.syncReleaseDate(ctx, market, prices),
		w.syncHistory(ctx, market, prices, participants),
	)
}

// syncReleaseDate takes the oldest trading day as the release date of markets
// whose listing sources do not provide one. The full history is used whatever
// the collection bounds are.
func (w *TSEMarketDataSync) syncReleaseDate(ctx context.Context, market *models.TSEMarket, prices []tse.ClosingPriceHistory) error {
	var oldest time.Time
	for _, item := range prices {
		if !item.Date.IsZero() && (oldest.IsZero() || item.Date.Before(oldest)) {
			oldest = item.Date
		}
	}
	if oldest.IsZero() {
		return nil
	}
	oldest = dateOnly(oldest)
	if market.IPODate != nil && (market.IPODateSource != models.TSEIPODateSourcePriceHistory || !oldest.Before(*market.IPODate)) {
		return nil
	}
	if err := w.marketRepo.Update(ctx, market.ID, map[string]any{
		"ipo_date":        oldest,
		"ipo_date_source": models.TSEIPODateSourcePriceHistory,
	}); err != nil {
		return fmt.Errorf("save release date: %w", err)
	}
	w.logger.WithFields(logrus.Fields{"ins_code": market.InsCode, "symbol": market.Label, "release_date": oldest.Format(time.DateOnly)}).
		Info("took the oldest trading day as the release date")
	market.IPODate = &oldest
	market.IPODateSource = models.TSEIPODateSourcePriceHistory
	return nil
}

// syncToday stores the current closing price of market and returns its date,
// which is the last trading day of the market; zero when it has none.
func (w *TSEMarketDataSync) syncToday(ctx context.Context, market *models.TSEMarket) (time.Time, error) {
	logrus.Info("sync today market")
	todayPrice, err := w.adapter.FetchCurrentClosingPrice(ctx, market.InsCode)
	if err != nil {
		return time.Time{}, fmt.Errorf("fetch current closing price: %w", err)
	}
	participant, err := w.adapter.FetchCurrentClientType(ctx, market.InsCode)
	if err != nil {
		return time.Time{}, fmt.Errorf("fetch current client type: %w", err)
	}
	if todayPrice.Date.IsZero() {
		return time.Time{}, nil
	}
	priceHistory := tse.ClosingPriceHistory{
		ClosingPrice: *todayPrice,
	}
	price := priceRow(priceHistory, market.MarketID, market.ID)
	prices := []models.TSEDailyClosingPrice{priceRowParticipant(price, *participant)}

	return dateOnly(todayPrice.Date), w.syncRows(ctx, prices)
}

// syncTradeHistory fetches the trades of the trading day just stored and
// records them in its daily closing price row. An older day whose trades were
// already fetched is not fetched again.
func (w *TSEMarketDataSync) syncTradeHistory(ctx context.Context, market *models.TSEMarket, date time.Time) error {
	if w.tradeHistory == nil {
		return fmt.Errorf("TSE trade history service is required in today mode")
	}
	day, err := w.tradeHistory.FindDay(ctx, market.ID, date)
	if err != nil {
		return fmt.Errorf("find trading day: %w", err)
	}
	if day == nil {
		return fmt.Errorf("daily closing price of %s was not stored", date.Format("2006-01-02"))
	}
	if day.HistoryStatus != nil && *day.HistoryStatus == consts.TSETradeDayFetched && date.Before(w.tradeHistory.Today()) {
		return nil
	}
	result, err := w.tradeHistory.CollectDay(ctx, market, *day, tradehistory.CollectOptions{
		MaxAttempts: tradeHistoryMaxAttempts,
		RetryDelay:  tradeHistoryRetryDelay,
	})
	if err != nil {
		return fmt.Errorf("collect trade history: %w", err)
	}
	if result.Err != nil {
		return fmt.Errorf("fetch trade history after %d attempts: %w", result.Attempts, result.Err)
	}
	w.logger.WithFields(logrus.Fields{
		"market_id": market.MarketID,
		"ins_code":  market.InsCode,
		"date":      date.Format("2006-01-02"),
		"status":    result.Status,
		"count":     result.TradeCount,
	}).Info("TSE trade history synced")
	return nil
}

func (w *TSEMarketDataSync) syncHistory(
	ctx context.Context,
	market *models.TSEMarket,
	prices []tse.ClosingPriceHistory,
	participants []tse.ClientType,
) error {
	from, to := w.bounds()
	pricesMap := make(map[int64]models.TSEDailyClosingPrice)
	for _, item := range prices {
		if !inBounds(item.Date, from, to) {
			continue
		}
		pricesMap[item.Date.Unix()] = priceRow(item, market.MarketID, market.ID)
	}
	priceRows := make([]models.TSEDailyClosingPrice, 0)
	for _, item := range participants {
		if inBounds(item.Date, from, to) {
			price, ok := pricesMap[item.Date.Unix()]
			if !ok {
				continue
			}
			priceRows = append(priceRows, priceRowParticipant(price, item))
		}
	}
	return w.syncRows(ctx, priceRows)
}

func (w *TSEMarketDataSync) syncRows(ctx context.Context, prices []models.TSEDailyClosingPrice) error {
	pricePointers := make([]*models.TSEDailyClosingPrice, len(prices))
	for i := range prices {
		pricePointers[i] = &prices[i]
	}
	if err := w.dailyRepo.UpsertPrices(ctx, pricePointers); err != nil {
		return fmt.Errorf("upsert candles: %w", err)
	}
	w.receivedData.Add(int64(len(prices)))
	if w.eventBus != nil {
		for _, row := range prices {
			if err := w.eventBus.PublishCandle(ctx, row.Candle); err != nil {
				return fmt.Errorf("publish TSE candle: %w", err)
			}
		}
	}
	logrus.Infof("sync rows done, %v added to %v", len(prices), w.receivedData.Load())
	return nil
}

func (w *TSEMarketDataSync) bounds() (time.Time, time.Time) {
	switch w.config.Mode {
	case consts.TSEMarketDataDate:
		return dateOnly(w.config.Date), dateOnly(w.config.Date)
	case consts.TSEMarketDataRange:
		return dateOnly(w.config.StartDate), dateOnly(w.config.EndDate)
	default:
		return time.Time{}, time.Time{}
	}
}

func inBounds(value, from, to time.Time) bool {
	value = dateOnly(value)
	return (from.IsZero() || !value.Before(from)) && (to.IsZero() || !value.After(to))
}

func (w *TSEMarketDataSync) wait(ctx context.Context) bool {
	if w.requestInterval == 0 {
		return true
	}
	timer := time.NewTimer(w.requestInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func priceRow(item tse.ClosingPriceHistory, marketID, tseMarketID uint) models.TSEDailyClosingPrice {
	lastChange := item.LastChange
	closeChange := item.Closing - item.Yesterday
	lastPercent, closePercent := 0.0, 0.0
	if item.Yesterday != 0 {
		lastPercent = 100 * lastChange / item.Yesterday
		closePercent = 100 * closeChange / item.Yesterday
	}
	return models.TSEDailyClosingPrice{
		Candle: models.Candle{
			MarketID:    marketID,
			TimeframeID: 1,
			Time:        item.Date.Unix(),
			Open:        decimal.NewFromFloat(item.First),
			High:        decimal.NewFromFloat(item.High),
			Low:         decimal.NewFromFloat(item.Low),
			Close:       decimal.NewFromFloat(item.Last),
			Volume:      decimal.NewFromInt(item.Volume),
		},
		MarketID:           marketID,
		TSEMarketID:        tseMarketID,
		TradeCount:         item.TradeCount,
		Value:              item.Value,
		Yesterday:          item.Yesterday,
		Last:               item.Last,
		LastChange:         lastChange,
		LastChangePercent:  lastPercent,
		CloseChange:        closeChange,
		CloseChangePercent: closePercent,
		Volume:             float64(item.Volume),
		First:              item.First,
		Closing:            item.Closing,
		Low:                item.Low,
		High:               item.High,
	}
}

func priceRowParticipant(price models.TSEDailyClosingPrice, participant tse.ClientType) models.TSEDailyClosingPrice {
	price.BuyIndividualVolume = participant.BuyIndividualVolume
	price.BuyIndividualValue = participant.BuyIndividualValue
	price.BuyIndividualCount = participant.BuyIndividualCount
	price.BuyInstitutionalVolume = participant.BuyInstitutionalVolume
	price.BuyInstitutionalValue = participant.BuyInstitutionalValue
	price.BuyInstitutionalCount = participant.BuyInstitutionalCount
	price.SellIndividualVolume = participant.SellIndividualVolume
	price.SellIndividualValue = participant.SellIndividualValue
	price.SellIndividualCount = participant.SellIndividualCount
	price.SellInstitutionalVolume = participant.SellInstitutionalVolume
	price.SellInstitutionalValue = participant.SellInstitutionalValue
	price.SellInstitutionalCount = participant.SellInstitutionalCount
	return price
}
