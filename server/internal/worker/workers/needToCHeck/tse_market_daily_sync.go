package workers

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"strings"
// 	"time"

// 	"github.com/h-varmazyar/gatec/configs"
// 	"github.com/h-varmazyar/gatec/internal/adapters/stock"
// 	"github.com/h-varmazyar/gatec/internal/models"
// 	"github.com/h-varmazyar/gatec/internal/repositories"
// 	indicatorsPkg "github.com/h-varmazyar/gatec/pkg/indicators"
// 	"github.com/shopspring/decimal"
// 	"github.com/sirupsen/logrus"
// )

// const (
// 	tseDailyWorkerName     = "tse-market-daily-sync"
// 	tseIndicatorProviderID = uint(2)
// 	// tseIndicatorHistorySize = 100
// 	telegramTextLimit = 4096
// )

// type TSEMarketDailySync struct {
// 	adapter             *stock.Adapter
// 	marketRepo          *repositories.TSEMarketRepository
// 	dailyRepo           *repositories.TSEDailyClosingPriceRepository
// 	indicatorRepo       *repositories.IndicatorDefinitionRepository
// 	snapshotRepo        *repositories.TSEIndicatorSnapshotRepository
// 	scheduleHour        int
// 	scheduleMin         int
// 	requestInterval     time.Duration
// 	reporter            *telegramBotReporter
// 	location            *time.Location
// 	cancel              context.CancelFunc
// 	logger              *logrus.Entry
// 	candleBroadcastChan chan models.Candle
// }

// type tseDailySyncReport struct {
// 	Date        time.Time
// 	Total       int
// 	WithData    int
// 	WithoutData int
// 	Failures    []string
// }

// func NewTSEMarketDailySync(
// 	cfg *configs.TSEMarketDailySync,
// 	marketRepo *repositories.TSEMarketRepository,
// 	dailyRepo *repositories.TSEDailyClosingPriceRepository,
// 	indicatorRepo *repositories.IndicatorDefinitionRepository,
// 	snapshotRepo *repositories.TSEIndicatorSnapshotRepository,
// 	candleBroadcastChan chan models.Candle,
// ) *TSEMarketDailySync {
// 	hour, minute, interval := 19, 0, 250*time.Millisecond
// 	var token string
// 	var chatID int64 = 1689926
// 	if cfg != nil {
// 		if cfg.ScheduleHour >= 0 && cfg.ScheduleHour <= 23 {
// 			hour = cfg.ScheduleHour
// 		}
// 		if cfg.ScheduleMin >= 0 && cfg.ScheduleMin <= 59 {
// 			minute = cfg.ScheduleMin
// 		}
// 		if cfg.RequestInterval >= 0 {
// 			interval = cfg.RequestInterval
// 		}
// 		token, chatID = cfg.BotToken, cfg.ReportChatID
// 	}
// 	location, err := time.LoadLocation("Asia/Tehran")
// 	if err != nil {
// 		location = time.FixedZone("Asia/Tehran", 3*60*60+30*60)
// 	}
// 	return &TSEMarketDailySync{
// 		adapter:             stock.NewAdapter(),
// 		marketRepo:          marketRepo,
// 		dailyRepo:           dailyRepo,
// 		indicatorRepo:       indicatorRepo,
// 		snapshotRepo:        snapshotRepo,
// 		scheduleHour:        hour,
// 		scheduleMin:         minute,
// 		requestInterval:     interval,
// 		reporter:            newTelegramBotReporter(token, chatID),
// 		location:            location,
// 		logger:              logrus.WithField("worker", tseDailyWorkerName),
// 		candleBroadcastChan: candleBroadcastChan,
// 	}
// }

// func (w *TSEMarketDailySync) Name() string { return tseDailyWorkerName }

// func (w *TSEMarketDailySync) Stop() error {
// 	if w.cancel != nil {
// 		w.cancel()
// 	}
// 	return nil
// }
// func (w *TSEMarketDailySync) Start(ctx context.Context) error {
// 	ctx, w.cancel = context.WithCancel(ctx)
// 	w.logger.Infof("starting daily sync at %02d:%02d Asia/Tehran", w.scheduleHour, w.scheduleMin)
// 	for {
// 		now := time.Now().In(w.location)
// 		next := w.nextRunTime(now)

// 		timer := time.NewTimer(time.Second * 5)
// 		w.logger.Infof("next TSE daily sync scheduled at %s", next.Format(time.RFC3339))
// 		select {
// 		case <-ctx.Done():
// 			timer.Stop()
// 			return nil
// 		case <-timer.C:
// 			report, err := w.syncAll(ctx, time.Now().In(w.location))
// 			if err != nil {
// 				w.logger.WithError(err).Error("daily TSE market sync failed")
// 				continue
// 			}
// 			if err := w.reporter.Send(ctx, report.message()); err != nil {
// 				w.logger.WithError(err).Error("failed to send daily TSE report")
// 			}
// 		}
// 	}
// }

// func (w *TSEMarketDailySync) nextRunTime(now time.Time) time.Time {
// 	next := time.Date(now.Year(), now.Month(), now.Day(), w.scheduleHour, w.scheduleMin, 0, 0, w.location)
// 	if !now.Before(next) {
// 		next = next.AddDate(0, 0, 1)
// 	}
// 	return next
// }

// func (w *TSEMarketDailySync) syncAll(ctx context.Context, today time.Time) (*tseDailySyncReport, error) {
// 	condition := map[string]any{
// 		"status = ?": "tradeable",
// 		// "state_updated_at > ?": time.Now().Add(time.Hour * -96),
// 	}
// 	markets, err := w.marketRepo.List(ctx, condition)
// 	if err != nil {
// 		return nil, fmt.Errorf("list TSE markets: %w", err)
// 	}
// 	report := &tseDailySyncReport{Date: today, Total: len(markets)}
// 	w.logger.WithField("markets", len(markets)).Info("starting daily TSE market sync")
// 	for index, market := range markets {
// 		if ctx.Err() != nil {
// 			return nil, ctx.Err()
// 		}
// 		if market.InsCode == "" {
// 			report.Failures = append(report.Failures, market.Market.Label+": ins_code خالی است")
// 		} else {
// 			hasTodayData, syncErr := w.syncMarket(ctx, market, today)
// 			switch {
// 			case syncErr != nil:
// 				report.Failures = append(report.Failures, fmt.Sprintf("%s (%s): %v", market.Market.Label, market.InsCode, syncErr))
// 				w.logger.WithFields(logrus.Fields{"market_id": market.MarketID, "ins_code": market.InsCode, "market": market.Market.Label}).WithError(syncErr).Error("failed to sync current TSE market")
// 			case hasTodayData:
// 				report.WithData++
// 			default:
// 				report.WithoutData++
// 			}
// 		}
// 		if index < len(markets)-1 && !w.wait(ctx) {
// 			return nil, ctx.Err()
// 		}
// 	}
// 	w.logger.WithFields(logrus.Fields{"markets": report.Total, "with_data": report.WithData, "without_data": report.WithoutData, "failed": len(report.Failures)}).Info("daily TSE market sync completed")
// 	return report, nil
// }

// func (w *TSEMarketDailySync) syncMarket(ctx context.Context, market *models.TSEMarket, today time.Time) (bool, error) {
// 	info, err := w.adapter.FetchInstrumentInfo(ctx, market.InsCode)
// 	if err != nil {
// 		return false, fmt.Errorf("fetch instrument info: %w", err)
// 	}
// 	marketUpdateColumns := make(map[string]any)
// 	marketUpdateColumns["nav"] = info.Nav
// 	marketUpdateColumns["esp"] = info.ESP

// 	price, err := w.adapter.FetchCurrentClosingPrice(ctx, info.InsCode)
// 	if err != nil {
// 		return false, fmt.Errorf("fetch closing price: %w", err)
// 	}

// 	if !price.FinalLastDate.IsZero() {
// 		marketUpdateColumns["last_price_date"] = &price.FinalLastDate
// 	}
// 	marketUpdateColumns["instrument_state_code"] = price.InstrumentStateCode

// 	if err = w.marketRepo.Update(ctx, market.MarketID, marketUpdateColumns); err != nil {
// 		return false, fmt.Errorf("failed to update tse market %w", err)
// 	}

// 	if !sameDate(price.Date, today) || !sameDate(price.FinalLastDate, today) {
// 		return false, nil
// 	}

// 	participants, err := w.adapter.FetchCurrentClientType(ctx, info.InsCode)
// 	if err != nil {
// 		return false, fmt.Errorf("fetch client type: %w", err)
// 	}
// 	lastChange := price.Last - price.Yesterday
// 	closeChange := price.Close - price.Yesterday
// 	lastPercent, closePercent := 0.0, 0.0
// 	if price.Yesterday != 0 {
// 		lastPercent = 100 * lastChange / price.Yesterday
// 		closePercent = 100 * closeChange / price.Yesterday
// 	}

// 	row := &models.TSEDailyClosingPrice{
// 		Candle: models.Candle{
// 			MarketID: market.Market.ID,
// 			Time:     dateOnly(today).Unix(),
// 			Open:     decimal.NewFromFloat(price.First),
// 			High:     decimal.NewFromFloat(price.High),
// 			Low:      decimal.NewFromFloat(price.Low),
// 			Close:    decimal.NewFromFloat(price.Close),
// 			Volume:   decimal.NewFromInt(price.Volume),
// 		},
// 		// InstrumentID:            market.MarketID,
// 		// Date:                    dateOnly(today),
// 		TradeCount: price.TradeCount,
// 		// Volume:                  price.Volume,
// 		Value:     price.Value,
// 		Yesterday: price.Yesterday,
// 		// First:                   price.First,
// 		Last:              price.Last,
// 		LastChange:        lastChange,
// 		LastChangePercent: lastPercent,
// 		// Close:                   price.Close,
// 		CloseChange:        closeChange,
// 		CloseChangePercent: closePercent,
// 		// Low:                     price.Low,
// 		// High:                    price.High,
// 		InstrumentStateCode:     price.InstrumentStateCode,
// 		BuyIndividualVolume:     participants.BuyIndividualVolume,
// 		BuyIndividualCount:      participants.BuyIndividualCount,
// 		BuyInstitutionalValue:   participants.BuyIndividualCount,
// 		BuyInstitutionalVolume:  participants.BuyInstitutionalVolume,
// 		BuyInstitutionalCount:   participants.BuyInstitutionalCount,
// 		SellIndividualVolume:    participants.SellIndividualVolume,
// 		SellIndividualCount:     participants.SellIndividualCount,
// 		SellInstitutionalVolume: participants.SellInstitutionalVolume,
// 		SellInstitutionalCount:  participants.SellInstitutionalCount,
// 	}
// 	if err := w.dailyRepo.UpsertPrices(ctx, []*models.TSEDailyClosingPrice{row}); err != nil {
// 		return false, fmt.Errorf("upsert current price: %w", err)
// 	}
// 	if err := w.dailyRepo.UpsertParticipants(ctx, []*models.TSEDailyClosingPrice{row}); err != nil {
// 		return false, fmt.Errorf("upsert current client type: %w", err)
// 	}

// 	w.broadcastCandle(ctx, row, today)
// 	// if err := w.calculateIndicatorSnapshot(ctx, market.ID, row.Date); err != nil {
// 	// 	return false, fmt.Errorf("calculate indicator snapshot: %w", err)
// 	// }
// 	return true, nil
// }

// func (w *TSEMarketDailySync) broadcastCandle(ctx context.Context, row *models.TSEDailyClosingPrice, candleTime time.Time) {
// 	// newCandle := models.Candle{
// 	// 	Time:     candleTime.Unix(),
// 	// 	Open:     decimal.NewFromFloat(row.First),
// 	// 	High:     decimal.NewFromFloat(row.High),
// 	// 	Low:      decimal.NewFromFloat(row.Low),
// 	// 	Close:    decimal.NewFromFloat(row.Close),
// 	// 	Volume:   decimal.NewFromInt(row.Volume),
// 	// 	MarketID: uint(row.InstrumentID),
// 	// }

// 	// w.mu.Lock()
// 	// candles := w.candleStorage[candle.Key]

// 	// if candleTime.Unix() > candles[len(candles)-1].Time {
// 	// 	ids := strings.Split(candle.Key, ":")
// 	// 	marketID, _ := strconv.ParseUint(ids[1], 10, 64)
// 	// 	tfID, _ := strconv.ParseUint(ids[2], 10, 64)

// 	// 	newCandle.MarketID = uint(marketID)
// 	// 	newCandle.TimeframeID = uint(tfID)

// 	// 	w.candleRepo.Create(ctx, &candles[len(candles)-1])
// 	// 	tmpCandles := append(candles[1:], newCandle)
// 	// 	candles = tmpCandles
// 	// } else if candleTime.Unix() == candles[len(candles)-1].Time {
// 	// 	newCandle.MarketID = candles[len(candles)-1].MarketID
// 	// 	newCandle.TimeframeID = candles[len(candles)-1].TimeframeID
// 	// 	candles[len(candles)-1] = newCandle
// 	// }
// 	w.candleBroadcastChan <- row.Candle
// }

// type tseIndicatorSnapshotResult struct {
// 	DefinitionID uint               `json:"definition_id"`
// 	Type         string             `json:"type"`
// 	TimeframeID  uint               `json:"timeframe_id"`
// 	Params       json.RawMessage    `json:"params"`
// 	Values       map[string]float64 `json:"values"`
// }

// // func (w *TSEMarketDailySync) calculateIndicatorSnapshot(ctx context.Context, marketID uint64, date time.Time) error {
// // 	definitions, err := w.indicatorRepo.ListForProviderAndMarket(ctx, tseIndicatorProviderID, marketID)
// // 	if err != nil {
// // 		return fmt.Errorf("list indicator definitions: %w", err)
// // 	}
// // 	if len(definitions) == 0 {
// // 		return nil
// // 	}

// // 	rows, err := w.dailyRepo.LastByMarket(ctx, marketID, 0)
// // 	if err != nil {
// // 		return fmt.Errorf("list latest market data: %w", err)
// // 	}
// // 	if len(rows) == 0 {
// // 		return fmt.Errorf("market has no daily data")
// // 	}

// // 	candles := make([]indicatorsPkg.Candle, 0)
// // 	for _, row := range rows {
// // 		if row.First == 0 || row.High == 0 || row.Low == 0 || row.Close == 0 {
// // 			continue
// // 		}
// // 		candles = append(candles, indicatorsPkg.Candle{
// // 			Time:   row.Date.Unix(),
// // 			Open:   decimal.NewFromFloat(row.First),
// // 			High:   decimal.NewFromFloat(row.High),
// // 			Low:    decimal.NewFromFloat(row.Low),
// // 			Close:  decimal.NewFromFloat(row.Close),
// // 			Volume: decimal.NewFromInt(row.Volume),
// // 		})
// // 	}

// // 	results := make([]tseIndicatorSnapshotResult, 0, len(definitions))
// // 	for _, definition := range definitions {
// // 		indicator, err := CreateIndicatorFromDefinition(definition)
// // 		if err != nil {
// // 			return fmt.Errorf("create definition %d (%s): %w", definition.ID, definition.Type, err)
// // 		}
// // 		if len(candles) < indicator.GetRequiredCandleCount() {
// // 			return fmt.Errorf("definition %d (%s) requires %d data points, got %d", definition.ID, definition.Type, indicator.GetRequiredCandleCount(), len(candles))
// // 		}

// // 		values, err := calculateLatestIndicatorValue(ctx, indicator, candles)
// // 		if err != nil {
// // 			return fmt.Errorf("definition %d (%s): %w", definition.ID, definition.Type, err)
// // 		}
// // 		results = append(results, tseIndicatorSnapshotResult{
// // 			DefinitionID: definition.ID,
// // 			Type:         strings.ToUpper(definition.Type),
// // 			TimeframeID:  definition.TimeframeID,
// // 			Params:       json.RawMessage(definition.Params),
// // 			Values:       values,
// // 		})
// // 	}

// // 	encoded, err := json.Marshal(results)
// // 	if err != nil {
// // 		return fmt.Errorf("encode snapshot: %w", err)
// // 	}
// // 	return w.snapshotRepo.Upsert(ctx, &models.TSEIndicatorSnapshot{
// // 		MarketID: marketID, ProviderID: tseIndicatorProviderID, Date: dateOnly(date), Results: encoded,
// // 	})
// // }

// func calculateLatestIndicatorValue(ctx context.Context, indicator indicatorsPkg.Indicator, candles []indicatorsPkg.Candle) (fields map[string]float64, err error) {
// 	defer func() {
// 		if recovered := recover(); recovered != nil {
// 			err = fmt.Errorf("indicator calculation panicked: %v", recovered)
// 		}
// 	}()
// 	values := indicator.Calculate(ctx, candles)
// 	if len(values) == 0 || values[len(values)-1].Value == nil {
// 		return nil, fmt.Errorf("indicator returned no value for latest day")
// 	}
// 	fields = make(map[string]float64)
// 	for name, value := range values[len(values)-1].Value.Fields() {
// 		number, _ := value.Float64()
// 		fields[name] = number
// 	}
// 	return fields, nil
// }

// func (w *TSEMarketDailySync) wait(ctx context.Context) bool {
// 	if w.requestInterval == 0 {
// 		return true
// 	}
// 	timer := time.NewTimer(w.requestInterval)
// 	defer timer.Stop()
// 	select {
// 	case <-ctx.Done():
// 		return false
// 	case <-timer.C:
// 		return true
// 	}
// }

// func sameDate(left, right time.Time) bool {
// 	ly, lm, ld := left.Date()
// 	ry, rm, rd := right.Date()
// 	return ly == ry && lm == rm && ld == rd
// }

// func dateOnly(value time.Time) time.Time {
// 	year, month, day := value.Date()
// 	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
// }

// func (r *tseDailySyncReport) message() string {
// 	var result strings.Builder
// 	fmt.Fprintf(&result, "گزارش بروزرسانی روزانه بورس تهران\nتاریخ: %s\nکل مارکت‌ها: %d\nدارای داده امروز: %d\nبدون داده امروز: %d\nخطاها: %d", r.Date.Format("2006-01-02"), r.Total, r.WithData, r.WithoutData, len(r.Failures))
// 	if len(r.Failures) > 0 {
// 		result.WriteString("\n\nمارکت‌های خطادار:\n")
// 		for _, failure := range r.Failures {
// 			result.WriteString("• ")
// 			result.WriteString(failure)
// 			result.WriteByte('\n')
// 		}
// 	}
// 	return strings.TrimSpace(result.String())
// }

// type telegramBotReporter struct {
// 	token      string
// 	chatID     int64
// 	httpClient *http.Client
// }

// func newTelegramBotReporter(token string, chatID int64) *telegramBotReporter {
// 	return &telegramBotReporter{token: strings.TrimSpace(token), chatID: chatID, httpClient: &http.Client{Timeout: 30 * time.Second}}
// }

// func (r *telegramBotReporter) Send(ctx context.Context, text string) error {
// 	if r.token == "" {
// 		return fmt.Errorf("telegram bot token is empty")
// 	}
// 	if r.chatID == 0 {
// 		return fmt.Errorf("telegram report chat ID is empty")
// 	}
// 	for _, part := range splitTelegramText(text) {
// 		if err := r.sendPart(ctx, part); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// func (r *telegramBotReporter) sendPart(ctx context.Context, text string) error {
// 	body, err := json.Marshal(map[string]any{"chat_id": r.chatID, "text": text})
// 	if err != nil {
// 		return fmt.Errorf("encode telegram report: %w", err)
// 	}
// 	endpoint := "https://api.telegram.org/bot" + r.token + "/sendMessage"
// 	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
// 	if err != nil {
// 		return fmt.Errorf("create telegram report request: %w", err)
// 	}
// 	req.Header.Set("Content-Type", "application/json")
// 	resp, err := r.httpClient.Do(req)
// 	if err != nil {
// 		return fmt.Errorf("send telegram report: %w", err)
// 	}
// 	defer resp.Body.Close()
// 	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
// 		return fmt.Errorf("telegram returned status %d", resp.StatusCode)
// 	}
// 	return nil
// }
// func splitTelegramText(text string) []string {
// 	runes := []rune(strings.TrimSpace(text))
// 	parts := make([]string, 0, len(runes)/telegramTextLimit+1)
// 	for len(runes) > 0 {
// 		end := len(runes)
// 		if end > telegramTextLimit {
// 			end = telegramTextLimit
// 			for end > 0 && runes[end-1] != '\n' {
// 				end--
// 			}
// 			if end == 0 {
// 				end = telegramTextLimit
// 			}
// 		}
// 		part := strings.TrimSpace(string(runes[:end]))
// 		if part != "" {
// 			parts = append(parts, part)
// 		}
// 		runes = runes[end:]
// 	}
// 	return parts
// }
