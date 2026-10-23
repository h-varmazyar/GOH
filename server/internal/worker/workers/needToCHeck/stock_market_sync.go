package workers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/adapters/stock"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type StockMarketSyncMode string

const (
	StockMarketSyncModeDaily      StockMarketSyncMode = "daily"
	StockMarketSyncModeHistorical StockMarketSyncMode = "historical"
)

// StockMarketSync fetches daily or historical stock data from TSETMC.
type StockMarketSync struct {
	adapter             *stock.Adapter
	assetRepo           *repositories.AssetRepository
	marketRepo          *repositories.MarketRepository
	stockDataRepo       *repositories.StockDailyDataRepository
	providerID          uint
	mode                StockMarketSyncMode
	scheduledHour       int
	scheduledMin        int
	historicalStartDate string
	historicalEndDate   string
	requestInterval     time.Duration
	cancel              context.CancelFunc
	logger              *logrus.Entry
}

func NewStockMarketSync(
	cfg *configs.StockMarketSync,
	assetRepo *repositories.AssetRepository,
	marketRepo *repositories.MarketRepository,
	stockDataRepo *repositories.StockDailyDataRepository,
) *StockMarketSync {
	mode := StockMarketSyncModeDaily
	hour, minute := 16, 0
	interval := time.Minute
	var providerID uint
	var startDate, endDate string

	if cfg != nil {
		providerID = cfg.ProviderID
		if cfg.Mode != "" {
			mode = StockMarketSyncMode(cfg.Mode)
		}
		if cfg.ScheduleHour >= 0 && cfg.ScheduleHour <= 23 {
			hour = cfg.ScheduleHour
		}
		if cfg.ScheduleMin >= 0 && cfg.ScheduleMin <= 59 {
			minute = cfg.ScheduleMin
		}
		if cfg.RequestInterval > 0 {
			interval = cfg.RequestInterval
		}
		startDate = cfg.HistoricalStartDate
		endDate = cfg.HistoricalEndDate
	}

	return &StockMarketSync{
		adapter:             stock.NewAdapter(),
		assetRepo:           assetRepo,
		marketRepo:          marketRepo,
		stockDataRepo:       stockDataRepo,
		providerID:          providerID,
		mode:                mode,
		scheduledHour:       hour,
		scheduledMin:        minute,
		historicalStartDate: startDate,
		historicalEndDate:   endDate,
		requestInterval:     interval,
		logger:              logrus.WithField("worker", "stock-market-sync"),
	}
}

func (w *StockMarketSync) Name() string { return "stock-market-sync" }

func (w *StockMarketSync) Start(ctx context.Context) error {
	ctx, w.cancel = context.WithCancel(ctx)

	switch w.mode {
	case StockMarketSyncModeDaily:
		return w.startDaily(ctx)
	case StockMarketSyncModeHistorical:
		return w.startHistorical(ctx)
	default:
		return fmt.Errorf("unsupported stock market sync mode %q", w.mode)
	}
}

func (w *StockMarketSync) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}
	return nil
}

func (w *StockMarketSync) startDaily(ctx context.Context) error {
	w.logger.Infof("starting daily sync at %02d:%02d", w.scheduledHour, w.scheduledMin)

	for {
		now := time.Now()
		next := w.nextRunTime(now)
		timer := time.NewTimer(next.Sub(now))
		w.logger.Infof("next stock sync scheduled at %s", next.Format("2006-01-02 15:04:05"))

		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
			if err := w.syncDate(ctx, w.todayShamsiDate()); err != nil {
				w.logger.WithError(err).Error("daily stock sync failed")
			}
		}
	}
}

func (w *StockMarketSync) startHistorical(ctx context.Context) error {
	start, err := parseJalaliDate(w.historicalStartDate)
	if err != nil {
		return fmt.Errorf("invalid historical start date: %w", err)
	}
	end, err := parseJalaliDate(w.historicalEndDate)
	if err != nil {
		return fmt.Errorf("invalid historical end date: %w", err)
	}
	if start.before(end) {
		return fmt.Errorf("historical start date %s must be equal to or after end date %s", start, end)
	}

	w.logger.Infof("starting historical sync from %s back to %s with interval %s", start, end, w.requestInterval)
	current := start
	for {
		if err := w.syncDate(ctx, current.String()); err != nil {
			w.logger.WithField("date", current.String()).WithError(err).Error("historical stock sync failed; continuing")
		}
		if current.equal(end) {
			w.logger.Info("historical stock sync completed")
			return nil
		}

		timer := time.NewTimer(w.requestInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
			current = current.previousDay()
		}
	}
}

func (w *StockMarketSync) nextRunTime(now time.Time) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), w.scheduledHour, w.scheduledMin, 0, 0, now.Location())
	if !now.Before(next) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func (w *StockMarketSync) syncDate(ctx context.Context, shamsiDate string) error {
	w.logger.Infof("fetching stock market data for date %s", shamsiDate)
	records, err := w.adapter.FetchMarketWatch(ctx, shamsiDate)
	if err != nil {
		return fmt.Errorf("fetching market watch: %w", err)
	}
	if len(records) == 0 {
		w.logger.WithField("date", shamsiDate).Warn("no stock records received")
		return nil
	}

	iriAsset, err := w.ensureAsset(ctx, "IRI", "ریال ایران", consts.AssetStock)
	if err != nil {
		return fmt.Errorf("ensuring IRI asset: %w", err)
	}

	fetchedAt := time.Now()
	stockDataRecords := make([]*models.StockDailyData, 0, len(records))
	for _, record := range records {
		sourceAsset, err := w.ensureAsset(ctx, record.Symbol, record.Name, consts.AssetStock)
		if err != nil {
			w.logger.WithField("symbol", record.Symbol).WithError(err).Error("failed to ensure source asset")
			continue
		}
		market, err := w.ensureMarket(ctx, sourceAsset, iriAsset, record.Symbol)
		if err != nil {
			w.logger.WithField("symbol", record.Symbol).WithError(err).Error("failed to ensure market")
			continue
		}

		stockDataRecords = append(stockDataRecords, &models.StockDailyData{
			MarketID:     market.ID,
			Date:         shamsiDate,
			Name:         record.Name,
			TradeCount:   record.TradeCount,
			Volume:       record.Volume,
			Value:        record.Value,
			Yesterday:    record.Yesterday,
			First:        record.First,
			Last:         record.Last,
			LastChange:   record.LastChange,
			LastPercent:  record.LastPercent,
			Close:        record.Close,
			CloseChange:  record.CloseChange,
			ClosePercent: record.ClosePercent,
			Low:          record.Low,
			High:         record.High,
			FetchedAt:    fetchedAt,
		})
	}

	if err := w.stockDataRepo.BulkUpsert(ctx, stockDataRecords); err != nil {
		return fmt.Errorf("saving stock data: %w", err)
	}
	w.logger.Infof("saved %d stock records for %s", len(stockDataRecords), shamsiDate)
	return nil
}

func (w *StockMarketSync) ensureAsset(ctx context.Context, label, name string, assetType consts.AssetType) (*models.Asset, error) {
	asset, err := w.assetRepo.ReturnByLabel(ctx, label)
	if err == nil {
		return asset, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("querying asset: %w", err)
	}

	asset = &models.Asset{Name: name, Label: label, Type: assetType}
	if err := w.assetRepo.Create(ctx, asset); err != nil {
		return nil, fmt.Errorf("creating asset: %w", err)
	}
	return asset, nil
}

func (w *StockMarketSync) ensureMarket(ctx context.Context, sourceAsset, destAsset *models.Asset, symbol string) (*models.Market, error) {
	conditions := map[string]any{
		"trading_asset_id = ?": sourceAsset.ID,
		"pricing_asset_id = ?": destAsset.ID,
	}
	if w.providerID > 0 {
		conditions["provider_id = ?"] = w.providerID
	}
	markets, err := w.marketRepo.List(ctx, conditions)
	if err != nil {
		return nil, fmt.Errorf("querying market: %w", err)
	}
	if len(markets) > 0 {
		return markets[0], nil
	}

	market := &models.Market{
		Name:           symbol,
		TradingAssetID: sourceAsset.ID,
		PricingAssetID: destAsset.ID,
		ProviderID:     w.providerID,
		Status:         consts.MarketStatusOpen,
	}
	if err := w.marketRepo.Create(ctx, market); err != nil {
		return nil, fmt.Errorf("creating market: %w", err)
	}
	return market, nil
}

func (w *StockMarketSync) todayShamsiDate() string {
	now := time.Now()
	jy, jm, jd := gregorianToShamsi(now.Year(), int(now.Month()), now.Day())
	return jalaliDate{year: jy, month: jm, day: jd}.String()
}

type jalaliDate struct{ year, month, day int }

func parseJalaliDate(value string) (jalaliDate, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 3 {
		return jalaliDate{}, fmt.Errorf("expected YYYY/MM/DD, got %q", value)
	}
	values := [3]int{}
	for i, part := range parts {
		parsed, err := strconv.Atoi(part)
		if err != nil {
			return jalaliDate{}, fmt.Errorf("invalid date %q", value)
		}
		values[i] = parsed
	}
	date := jalaliDate{year: values[0], month: values[1], day: values[2]}
	if date.year < 1 || date.month < 1 || date.month > 12 || date.day < 1 || date.day > date.daysInMonth() {
		return jalaliDate{}, fmt.Errorf("invalid Jalali date %q", value)
	}
	return date, nil
}

func (d jalaliDate) String() string              { return fmt.Sprintf("%04d/%02d/%02d", d.year, d.month, d.day) }
func (d jalaliDate) equal(other jalaliDate) bool { return d == other }
func (d jalaliDate) before(other jalaliDate) bool {
	if d.year != other.year {
		return d.year < other.year
	}
	if d.month != other.month {
		return d.month < other.month
	}
	return d.day < other.day
}
func (d jalaliDate) previousDay() jalaliDate {
	d.day--
	if d.day > 0 {
		return d
	}
	d.month--
	if d.month == 0 {
		d.year--
		d.month = 12
	}
	d.day = d.daysInMonth()
	return d
}
func (d jalaliDate) daysInMonth() int {
	if d.month <= 6 {
		return 31
	}
	if d.month <= 11 {
		return 30
	}
	if isJalaliLeapYear(d.year) {
		return 30
	}
	return 29
}
func isJalaliLeapYear(year int) bool {
	// The modern Jalali calendar follows this 33-year leap-year pattern.
	remainder := year % 33
	switch remainder {
	case 1, 5, 9, 13, 17, 22, 26, 30:
		return true
	default:
		return false
	}
}

func gregorianToShamsi(gy, gm, gd int) (int, int, int) {
	gDaysInMonth := []int{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}
	gy2 := gy + 1
	if gm > 2 {
		gy2 = gy
	}

	days := 355666 + 365*gy + (gy2+3)/4 - (gy2+99)/100 + (gy2+399)/400 + gd + gDaysInMonth[gm-1]
	jy := -1595 + 33*(days/12053)
	days %= 12053
	jy += 4 * (days / 1461)
	days %= 1461
	if days > 365 {
		jy += (days - 1) / 365
		days = (days - 1) % 365
	}
	if days < 186 {
		return jy, 1 + days/31, 1 + days%31
	}
	return jy, 7 + (days-186)/30, 1 + (days-186)%30
}
