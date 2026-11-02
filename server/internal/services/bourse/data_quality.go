package bourse

import (
	"context"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
)

// DailyMarketDataSource reads the listed-market universe and its daily OHLCV
// plus investor-flow rows. Date bounds are inclusive calendar dates.
type DailyMarketDataSource interface {
	ListMarkets(context.Context) ([]*models.TSEMarket, error)
	ListDailyRows(context.Context, time.Time, time.Time) ([]*models.TSEDailyClosingPrice, error)
}

type DailyBar struct {
	Date                    time.Time
	MarketID                uint
	TSEMarketID             uint
	InsCode                 string
	Symbol                  string
	MarketType              string
	SymbolType              string
	InstrumentStateCode     string
	Open, High, Low, Close  float64
	ReferencePrice          float64
	OfficialClose           float64
	Volume                  float64
	TradeCount              int64
	Value                   int64
	BuyIndividualCount      int64
	BuyIndividualVolume     int64
	BuyIndividualValue      int64
	BuyInstitutionalCount   int64
	BuyInstitutionalVolume  int64
	BuyInstitutionalValue   int64
	SellIndividualCount     int64
	SellIndividualVolume    int64
	SellIndividualValue     int64
	SellInstitutionalCount  int64
	SellInstitutionalVolume int64
	SellInstitutionalValue  int64
	ExcludedFromPriceWindow bool
	AdjustmentDiscontinuity bool
}

type QualityIssue struct {
	InsCode string
	Symbol  string
	Date    time.Time
	Reasons []string
}

type QualityReport struct {
	Date                         time.Time
	LookbackStart                time.Time
	UniverseCount                int
	RowsInRange                  int
	ObservedTradingDays          int
	RowsOnDate                   int
	ParticipantCovered           int
	ParticipantCoveragePercent   float64
	NoTradeCount                 int
	InvalidCount                 int
	MissingDayCount              int
	AdjustmentDiscontinuityCount int
	UnclassifiedInstrumentCount  int
	ExcludedInstrumentCount      int
	Issues                       []QualityIssue
}

// LoadQualityReport reads the full eligible universe and daily rows, then
// builds a deterministic quality summary for the requested date.
func LoadQualityReport(ctx context.Context, source DailyMarketDataSource, from, date time.Time, location *time.Location) (QualityReport, error) {
	if source == nil {
		return QualityReport{}, fmt.Errorf("daily market data source is required")
	}
	if location == nil {
		location = time.UTC
	}
	from = calendarDate(from, location)
	date = calendarDate(date, location)
	if from.After(date) {
		return QualityReport{}, fmt.Errorf("lookback start %s is after report date %s", from.Format("2006-01-02"), date.Format("2006-01-02"))
	}
	markets, err := source.ListMarkets(ctx)
	if err != nil {
		return QualityReport{}, fmt.Errorf("list TSE market universe: %w", err)
	}
	rows, err := source.ListDailyRows(ctx, from, date)
	if err != nil {
		return QualityReport{}, fmt.Errorf("load TSE daily rows: %w", err)
	}
	return AnalyzeQuality(markets, rows, from, date, location), nil
}

// LoadDailyBars returns normalized TSE OHLCV and investor-flow data for the
// inclusive date range. Bars whose requested lookback crosses a price-basis
// discontinuity are marked ExcludedFromPriceWindow.
func LoadDailyBars(ctx context.Context, source DailyMarketDataSource, from, to time.Time, location *time.Location) ([]DailyBar, error) {
	if source == nil {
		return nil, fmt.Errorf("daily market data source is required")
	}
	if location == nil {
		location = time.UTC
	}
	from = calendarDate(from, location)
	to = calendarDate(to, location)
	if from.After(to) {
		return nil, fmt.Errorf("range start %s is after end %s", from.Format("2006-01-02"), to.Format("2006-01-02"))
	}
	markets, err := source.ListMarkets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list TSE market universe: %w", err)
	}
	rows, err := source.ListDailyRows(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("load TSE daily rows: %w", err)
	}
	marketByID := make(map[uint]*models.TSEMarket, len(markets))
	for _, market := range markets {
		if market != nil && eligibleMarket(market) {
			marketByID[market.ID] = market
		}
	}
	barsByMarket := make(map[uint][]*DailyBar)
	for _, row := range rows {
		if row == nil || row.CandleID == 0 {
			continue
		}
		market := marketByID[row.TSEMarketID]
		if market == nil {
			continue
		}
		day := calendarDate(time.Unix(row.Candle.Time, 0), location)
		barsByMarket[row.TSEMarketID] = append(barsByMarket[row.TSEMarketID], mapDailyBar(row, market, day))
	}
	result := make([]DailyBar, 0, len(rows))
	for _, bars := range barsByMarket {
		markPriceDiscontinuities(bars, &QualityReport{})
		for _, bar := range bars {
			result = append(result, *bar)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Date.Equal(result[j].Date) {
			return result[i].TSEMarketID < result[j].TSEMarketID
		}
		return result[i].Date.Before(result[j].Date)
	})
	return result, nil
}

// AnalyzeQuality checks daily-market rows returned by the source. The TSE
// adapter does not provide adjusted history or a corporate-action factor.
// Price basis changes therefore exclude the symbol's requested price window
// from downstream indicator calculations.
func AnalyzeQuality(markets []*models.TSEMarket, rows []*models.TSEDailyClosingPrice, from, date time.Time, location *time.Location) QualityReport {
	if location == nil {
		location = time.UTC
	}
	from = calendarDate(from, location)
	date = calendarDate(date, location)

	marketByID := make(map[uint]*models.TSEMarket, len(markets))
	report := QualityReport{Date: date, LookbackStart: from}
	for _, market := range markets {
		if market == nil {
			continue
		}
		switch instrumentType(market) {
		case "stock":
			if !eligibleMarket(market) {
				report.ExcludedInstrumentCount++
				continue
			}
		case "excluded":
			report.ExcludedInstrumentCount++
			continue
		default:
			report.UnclassifiedInstrumentCount++
			continue
		}
		marketByID[market.ID] = market
		report.UniverseCount++
	}

	barsByMarket := make(map[uint][]*DailyBar)
	tradingDays := make(map[string]time.Time)
	for _, row := range rows {
		if row == nil || row.CandleID == 0 {
			continue
		}
		market := marketByID[row.TSEMarketID]
		if market == nil {
			continue
		}
		day := calendarDate(time.Unix(row.Candle.Time, 0), location)
		bar := mapDailyBar(row, market, day)
		barsByMarket[row.TSEMarketID] = append(barsByMarket[row.TSEMarketID], bar)
		tradingDays[day.Format("2006-01-02")] = day
		report.RowsInRange++
	}
	report.ObservedTradingDays = len(tradingDays)
	for _, bars := range barsByMarket {
		markPriceDiscontinuities(bars, &report)
	}

	orderedDays := make([]time.Time, 0, len(tradingDays))
	for _, day := range tradingDays {
		orderedDays = append(orderedDays, day)
	}
	for _, bars := range barsByMarket {
		if len(bars) == 0 {
			continue
		}
		seen := make(map[string]struct{}, len(bars))
		for _, bar := range bars {
			seen[bar.Date.Format("2006-01-02")] = struct{}{}
		}
		for _, day := range orderedDays {
			if day.Before(bars[0].Date) || day.After(bars[len(bars)-1].Date) {
				continue
			}
			if _, ok := seen[day.Format("2006-01-02")]; !ok {
				report.MissingDayCount++
				appendIssue(&report, bars[0].InsCode, bars[0].Symbol, day, "missing_market_day")
			}
		}
	}

	for _, market := range marketByID {
		bars := barsByMarket[market.ID]
		var current *DailyBar
		for _, bar := range bars {
			if bar.Date.Equal(date) {
				current = bar
				break
			}
		}
		if current == nil {
			appendIssue(&report, market.InsCode, market.Label, date, "missing_report_day")
			continue
		}
		report.RowsOnDate++
		if participantDataPresent(current) {
			report.ParticipantCovered++
		} else if current.TradeCount > 0 || current.Volume > 0 {
			appendIssue(&report, current.InsCode, current.Symbol, date, "participant_data_missing")
		}
		if current.TradeCount <= 0 || current.Volume <= 0 {
			report.NoTradeCount++
			appendIssue(&report, current.InsCode, current.Symbol, date, "no_trade_or_zero_volume")
		}
		if invalidBar(current) {
			report.InvalidCount++
			appendIssue(&report, current.InsCode, current.Symbol, date, "invalid_ohlcv_or_participant_value")
		}
		if (current.TradeCount <= 0 || current.Volume <= 0) && strings.TrimSpace(current.InstrumentStateCode) != "" {
			appendIssue(&report, current.InsCode, current.Symbol, date, "instrument_state:"+current.InstrumentStateCode)
		}
	}
	if report.UniverseCount > 0 {
		report.ParticipantCoveragePercent = float64(report.ParticipantCovered) * 100 / float64(report.UniverseCount)
	}
	sort.Slice(report.Issues, func(i, j int) bool {
		if report.Issues[i].Date.Equal(report.Issues[j].Date) {
			return report.Issues[i].InsCode < report.Issues[j].InsCode
		}
		return report.Issues[i].Date.Before(report.Issues[j].Date)
	})
	return report
}

func (r QualityReport) WriteReport(w io.Writer) error {
	if w == nil {
		return fmt.Errorf("quality report writer is required")
	}
	if _, err := fmt.Fprintf(w, "کیفیت داده بورس | %s\n", r.Date.Format("2006-01-02")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "نمادهای واجد بررسی: %d | رکوردهای بازه: %d | روزهای معاملاتی مشاهده‌شده: %d\n", r.UniverseCount, r.RowsInRange, r.ObservedTradingDays); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "رکورد روز گزارش: %d | پوشش داده حقیقی/حقوقی: %d/%d (%.2f%%)\n", r.RowsOnDate, r.ParticipantCovered, r.UniverseCount, r.ParticipantCoveragePercent); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "بدون معامله/حجم صفر: %d | OHLCV نامعتبر: %d | روزهای جاافتاده: %d | گپ مرجع قیمت: %d | نماد نوع‌نامشخص: %d | غیرسهام عادی خارج‌شده: %d\n", r.NoTradeCount, r.InvalidCount, r.MissingDayCount, r.AdjustmentDiscontinuityCount, r.UnclassifiedInstrumentCount, r.ExcludedInstrumentCount); err != nil {
		return err
	}
	const issueSampleLimit = 25
	for i, issue := range r.Issues {
		if i == issueSampleLimit {
			if _, err := fmt.Fprintf(w, "... %d مورد دیگر در گزارش جزئی موجود است\n", len(r.Issues)-issueSampleLimit); err != nil {
				return err
			}
			break
		}
		if _, err := fmt.Fprintf(w, "- %s %s %s: %s\n", issue.Date.Format("2006-01-02"), issue.InsCode, issue.Symbol, strings.Join(issue.Reasons, ",")); err != nil {
			return err
		}
	}
	return nil
}

func mapDailyBar(row *models.TSEDailyClosingPrice, market *models.TSEMarket, date time.Time) *DailyBar {
	candle := row.Candle
	return &DailyBar{
		Date: date, MarketID: row.MarketID, TSEMarketID: row.TSEMarketID,
		InsCode: market.InsCode, Symbol: market.Label, MarketType: string(market.MarketType),
		SymbolType: string(market.Kind), InstrumentStateCode: row.InstrumentStateCode,
		Open: candle.Open.InexactFloat64(), High: candle.High.InexactFloat64(), Low: candle.Low.InexactFloat64(),
		Close: candle.Close.InexactFloat64(), Volume: candle.Volume.InexactFloat64(),
		ReferencePrice: float64(row.Yesterday), OfficialClose: row.Closing,
		TradeCount: row.TradeCount, Value: row.Value,
		BuyIndividualCount: row.BuyIndividualCount, BuyIndividualVolume: row.BuyIndividualVolume, BuyIndividualValue: row.BuyIndividualValue,
		BuyInstitutionalCount: row.BuyInstitutionalCount, BuyInstitutionalVolume: row.BuyInstitutionalVolume, BuyInstitutionalValue: row.BuyInstitutionalValue,
		SellIndividualCount: row.SellIndividualCount, SellIndividualVolume: row.SellIndividualVolume, SellIndividualValue: row.SellIndividualValue,
		SellInstitutionalCount: row.SellInstitutionalCount, SellInstitutionalVolume: row.SellInstitutionalVolume, SellInstitutionalValue: row.SellInstitutionalValue,
	}
}

func markPriceDiscontinuities(bars []*DailyBar, report *QualityReport) {
	if len(bars) < 2 {
		return
	}
	for i := 1; i < len(bars); i++ {
		current := bars[i]
		previous := bars[i-1]
		if current.ReferencePrice <= 0 || previous.OfficialClose <= 0 ||
			math.Abs(current.ReferencePrice-previous.OfficialClose) > 0.001 {
			current.AdjustmentDiscontinuity = true
			for _, bar := range bars {
				bar.ExcludedFromPriceWindow = true
			}
			report.AdjustmentDiscontinuityCount++
			appendIssue(report, current.InsCode, current.Symbol, current.Date, "reference_price_basis_change_excludes_lookback")
		}
	}
}

func eligibleMarket(market *models.TSEMarket) bool {
	if market.Market != nil && (market.Market.Status == consts.MarketStatusDeactivateByExchange || market.Market.Status == consts.MarketStatusDeactivateBySystem) {
		return false
	}
	return instrumentType(market) == "stock"
}

func instrumentType(market *models.TSEMarket) string {
	switch market.Kind {
	case consts.TSEInstrumentKindStock:
		return "stock"
	case consts.TSEInstrumentKindFund, consts.TSEInstrumentKindRight:
		return "excluded"
	}
	return "unknown"
}

func participantDataPresent(bar *DailyBar) bool {
	return bar.BuyIndividualCount > 0 || bar.BuyInstitutionalCount > 0 || bar.SellIndividualCount > 0 || bar.SellInstitutionalCount > 0 ||
		bar.BuyIndividualVolume > 0 || bar.BuyInstitutionalVolume > 0 || bar.SellIndividualVolume > 0 || bar.SellInstitutionalVolume > 0 ||
		bar.BuyIndividualValue > 0 || bar.BuyInstitutionalValue > 0 || bar.SellIndividualValue > 0 || bar.SellInstitutionalValue > 0
}

func invalidBar(bar *DailyBar) bool {
	if !finite(bar.Open) || !finite(bar.High) || !finite(bar.Low) || !finite(bar.Close) || !finite(bar.Volume) ||
		bar.Open <= 0 || bar.High <= 0 || bar.Low <= 0 || bar.Close <= 0 || bar.Volume < 0 || bar.Value < 0 || bar.TradeCount < 0 {
		return true
	}
	if bar.High < math.Max(bar.Open, math.Max(bar.Low, bar.Close)) || bar.Low > math.Min(bar.Open, math.Min(bar.High, bar.Close)) {
		return true
	}
	return bar.BuyIndividualCount < 0 || bar.BuyIndividualVolume < 0 || bar.BuyIndividualValue < 0 ||
		bar.BuyInstitutionalCount < 0 || bar.BuyInstitutionalVolume < 0 || bar.BuyInstitutionalValue < 0 ||
		bar.SellIndividualCount < 0 || bar.SellIndividualVolume < 0 || bar.SellIndividualValue < 0 ||
		bar.SellInstitutionalCount < 0 || bar.SellInstitutionalVolume < 0 || bar.SellInstitutionalValue < 0
}

func appendIssue(report *QualityReport, insCode, symbol string, date time.Time, reason string) {
	for i := range report.Issues {
		if report.Issues[i].InsCode == insCode && report.Issues[i].Date.Equal(date) {
			report.Issues[i].Reasons = append(report.Issues[i].Reasons, reason)
			return
		}
	}
	report.Issues = append(report.Issues, QualityIssue{InsCode: insCode, Symbol: symbol, Date: date, Reasons: []string{reason}})
}

func calendarDate(value time.Time, location *time.Location) time.Time {
	local := value.In(location)
	year, month, day := local.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, location)
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
