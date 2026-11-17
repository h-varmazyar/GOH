// Package candles builds the candles of markets from their stored trades.
//
// Candles are kept in ClickHouse, next to the trades they are aggregated from:
// a build is one INSERT ... SELECT per market and range of days, so the trades
// never leave ClickHouse, and a candle built again, such as the weekly candle
// of a week still open, replaces the stored one. Candles belong to markets, not
// to TSE insCodes: the trades of every insCode of a market make its candles.
//
// The panel and the job-scheduler worker queue candle jobs with the service,
// and the candle-builder worker runs them with it. The markets and timeframes
// of a job are rows of the database that belong to the provider of the job.
// Only TSE trades are stored today; another provider needs a trade source of
// its own in BuildMarket.
package candles

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/h-varmazyar/gatec/configs"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/pkg/jalali"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrNoStorage    = errors.New("candle storage is not configured")
)

const dateLayout = "2006-01-02"

// SupportedTimeframe is a timeframe candles can be built in.
type SupportedTimeframe struct {
	Label    string
	Duration time.Duration
}

// SupportedTimeframes are the timeframes candles are built in. Intraday
// candles start at multiples of their duration from midnight in Tehran; weekly
// candles are TSE weeks, from Saturday to Friday; monthly candles are Jalali
// months, whose duration here only identifies the timeframe.
var SupportedTimeframes = []SupportedTimeframe{
	{Label: "5m", Duration: 5 * time.Minute},
	{Label: "15m", Duration: 15 * time.Minute},
	{Label: "1h", Duration: time.Hour},
	{Label: "4h", Duration: 4 * time.Hour},
	{Label: "1w", Duration: 7 * 24 * time.Hour},
	{Label: "1M", Duration: repositories.MonthlyDuration},
}

func supported(duration time.Duration) bool {
	return slices.ContainsFunc(SupportedTimeframes, func(t SupportedTimeframe) bool { return t.Duration == duration })
}

type Service struct {
	cfg           *configs.CandleBuilder
	jobRepo       *repositories.CandleJobRepository
	providerRepo  *repositories.ProviderRepository
	timeframeRepo *repositories.TimeframeRepository
	store         *repositories.MarketCandleRepository
	location      *time.Location
	logger        *logrus.Entry
}

// NewService creates the service; store may be nil where candles are only
// queued, such as in the job-scheduler worker.
func NewService(
	cfg *configs.CandleBuilder,
	jobRepo *repositories.CandleJobRepository,
	providerRepo *repositories.ProviderRepository,
	timeframeRepo *repositories.TimeframeRepository,
	store *repositories.MarketCandleRepository,
) *Service {
	if cfg == nil {
		cfg = &configs.CandleBuilder{}
	}
	location, err := time.LoadLocation(repositories.TehranTimezone)
	if err != nil {
		logrus.WithError(err).Warn("load Asia/Tehran timezone; using a fixed +03:30 offset")
		location = time.FixedZone(repositories.TehranTimezone, 3*3600+1800)
	}
	return &Service{
		cfg:           cfg,
		jobRepo:       jobRepo,
		providerRepo:  providerRepo,
		timeframeRepo: timeframeRepo,
		store:         store,
		location:      location,
		logger:        logrus.WithField("service", "candles"),
	}
}

// Today is the current date in Tehran, as UTC midnight like the stored days.
func (s *Service) Today() time.Time { return dateOnly(time.Now().In(s.location)) }

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// ParseDate parses a YYYY-MM-DD date.
func ParseDate(value string) (time.Time, error) {
	date, err := time.ParseInLocation(dateLayout, value, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: date %q must use YYYY-MM-DD", ErrInvalidInput, value)
	}
	return date, nil
}

// weekStart is the Saturday that starts the TSE week of date.
func weekStart(date time.Time) time.Time {
	return date.AddDate(0, 0, -((int(date.Weekday()) + 1) % 7))
}

// --- Providers and timeframes ---

// Provider returns a provider whose markets candles can be built for.
func (s *Service) Provider(ctx context.Context, id uint) (*models.Provider, error) {
	providers, err := s.providerRepo.Active(ctx)
	if err != nil {
		return nil, err
	}
	for i := range providers {
		if providers[i].ID != id {
			continue
		}
		if providers[i].Name != consts.ProviderTSE {
			return nil, fmt.Errorf("%w: candles are built only from TSE trades, not %s", ErrInvalidInput, providers[i].Name)
		}
		return &providers[i], nil
	}
	return nil, fmt.Errorf("%w: active provider %d", ErrNotFound, id)
}

// TSEProvider returns the active TSE provider.
func (s *Service) TSEProvider(ctx context.Context) (*models.Provider, error) {
	provider, err := s.providerRepo.ReturnByName(ctx, consts.ProviderTSE)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: TSE provider", ErrNotFound)
	}
	if err != nil {
		return nil, err
	}
	if provider.Status != consts.ProviderStatusEnable {
		return nil, fmt.Errorf("%w: the TSE provider is disabled", ErrInvalidInput)
	}
	return provider, nil
}

// Timeframes returns the timeframes of a provider that candles are built in.
func (s *Service) Timeframes(ctx context.Context, providerID uint) ([]models.Timeframe, error) {
	all, err := s.timeframeRepo.ListByProvider(ctx, providerID)
	if err != nil {
		return nil, err
	}
	result := make([]models.Timeframe, 0, len(all))
	for _, timeframe := range all {
		if supported(timeframe.Duration) {
			result = append(result, timeframe)
		}
	}
	return result, nil
}

// EnsureTimeframes creates the supported timeframes a provider does not have
// yet and returns every supported timeframe of the provider.
func (s *Service) EnsureTimeframes(ctx context.Context, providerID uint) ([]models.Timeframe, error) {
	existing, err := s.Timeframes(ctx, providerID)
	if err != nil {
		return nil, err
	}
	for _, timeframe := range SupportedTimeframes {
		if slices.ContainsFunc(existing, func(t models.Timeframe) bool { return t.Duration == timeframe.Duration }) {
			continue
		}
		created := &models.Timeframe{Label: timeframe.Label, Duration: timeframe.Duration, ProviderID: providerID}
		if err := s.timeframeRepo.Create(ctx, created); err != nil {
			return nil, fmt.Errorf("create timeframe %s: %w", timeframe.Label, err)
		}
		s.logger.WithFields(logrus.Fields{"provider_id": providerID, "timeframe": timeframe.Label, "id": created.ID}).
			Info("created candle timeframe")
	}
	return s.Timeframes(ctx, providerID)
}

// TimeframesByLabel returns the timeframes of a provider with the labels, as
// given in configuration; no labels selects every supported timeframe.
func (s *Service) TimeframesByLabel(ctx context.Context, providerID uint, labels []string) ([]uint, error) {
	timeframes, err := s.EnsureTimeframes(ctx, providerID)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(timeframes))
	for _, label := range labels {
		index := slices.IndexFunc(SupportedTimeframes, func(t SupportedTimeframe) bool { return t.Label == label })
		if index < 0 {
			return nil, fmt.Errorf("%w: timeframe %q is not supported", ErrInvalidInput, label)
		}
		for _, timeframe := range timeframes {
			if timeframe.Duration == SupportedTimeframes[index].Duration {
				ids = append(ids, timeframe.ID)
				break
			}
		}
	}
	if len(labels) == 0 {
		for _, timeframe := range timeframes {
			ids = append(ids, timeframe.ID)
		}
	}
	return ids, nil
}

// --- Candles ---

// Candles returns the candles of a market opening in [from, to], oldest first;
// with limit > 0 only the newest limit candles.
func (s *Service) Candles(ctx context.Context, marketID, timeframeID uint, from, to *time.Time, limit int) ([]models.MarketCandle, error) {
	if s.store == nil {
		return nil, ErrNoStorage
	}
	return s.store.Range(ctx, marketID, timeframeID, from, to, limit)
}

// BuildResult is the result of building the candles of one market.
type BuildResult struct {
	// Empty is true when the market had no trades in the range.
	Empty   bool
	Candles int64
	Chunks  int
}

// BuildMarket builds the candles of a market of a job in its timeframes. The
// days of the job with trades are widened to whole weeks, so every candle is
// built from all of its trades, and built in chunks of whole weeks; monthly
// candles are built apart, in chunks of whole Jalali months.
func (s *Service) BuildMarket(ctx context.Context, job *models.CandleJob, timeframes []repositories.CandleTimeframe, market repositories.TSEMarketSource) (BuildResult, error) {
	var result BuildResult
	if s.store == nil {
		return result, ErrNoStorage
	}
	first, last, found, err := s.store.TSETradeRange(ctx, market.InsCodes, job.FromDate, job.ToDate)
	if err != nil {
		return result, fmt.Errorf("read trade days: %w", err)
	}
	if !found {
		result.Empty = true
		return result, nil
	}
	version := uint64(time.Now().UnixNano())
	var weekly, monthly []repositories.CandleTimeframe
	for _, timeframe := range timeframes {
		if timeframe.Monthly() {
			monthly = append(monthly, timeframe)
		} else {
			weekly = append(weekly, timeframe)
		}
	}
	builds := make([]repositories.TSECandleBuild, 0)
	if len(weekly) > 0 {
		chunkDays := max((s.cfg.ChunkDays+6)/7*7, 7)
		end := weekStart(last).AddDate(0, 0, 6)
		for from := weekStart(first); !from.After(end); from = from.AddDate(0, 0, chunkDays) {
			to := from.AddDate(0, 0, chunkDays-1)
			if to.After(end) {
				to = end
			}
			builds = append(builds, repositories.TSECandleBuild{From: from, To: to, Timeframes: weekly})
		}
	}
	if len(monthly) > 0 {
		builds = append(builds, monthBuilds(first, last, max(s.cfg.ChunkDays/30, 1), monthly)...)
	}
	for _, build := range builds {
		build.ProviderID, build.MarketID, build.InsCodes, build.Version = job.ProviderID, market.MarketID, market.InsCodes, version
		written, err := s.buildChunk(ctx, build)
		if err != nil {
			return result, fmt.Errorf("build %s to %s: %w", build.From.Format(dateLayout), build.To.Format(dateLayout), err)
		}
		result.Candles += written
		result.Chunks++
	}
	return result, nil
}

// monthBuilds splits the Jalali months from the month of first to the month
// of last into builds of chunkMonths whole months.
func monthBuilds(first, last time.Time, chunkMonths int, timeframes []repositories.CandleTimeframe) []repositories.TSECandleBuild {
	builds := make([]repositories.TSECandleBuild, 0)
	for from := jalali.MonthStart(first); !from.After(last); {
		starts := []time.Time{from}
		next := from
		for range chunkMonths {
			next = jalali.NextMonthStart(next)
			starts = append(starts, next)
			if next.After(last) {
				break
			}
		}
		builds = append(builds, repositories.TSECandleBuild{
			From: from, To: next.AddDate(0, 0, -1), Timeframes: timeframes, MonthStarts: starts,
		})
		from = next
	}
	return builds
}

// buildChunk runs a build, again after a failure up to MaxAttempts times.
func (s *Service) buildChunk(ctx context.Context, build repositories.TSECandleBuild) (int64, error) {
	attempts := max(s.cfg.MaxAttempts, 1)
	for attempt := 1; ; attempt++ {
		written, err := s.store.BuildTSECandles(ctx, build)
		if err == nil || ctx.Err() != nil || attempt == attempts {
			return written, err
		}
		s.logger.WithError(err).WithFields(logrus.Fields{"market_id": build.MarketID, "attempt": attempt}).
			Warn("retrying candle build")
		timer := time.NewTimer(s.cfg.RetryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return 0, ctx.Err()
		case <-timer.C:
		}
	}
}
