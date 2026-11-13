package tradehistory

import (
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	service "github.com/h-varmazyar/gatec/internal/services/tradehistory"
)

const dateLayout = "2006-01-02"

type errorResp struct {
	Error string `json:"error"`
}

// --- Requests ---

type dayFilterReq struct {
	// Status is FETCHED, EMPTY, FAILED, MISSING or MISMATCHED.
	Status string `form:"status"`
	From   string `form:"from"`
	To     string `form:"to"`
}

type gapFilterReq struct {
	Query string `json:"q,omitempty" form:"q"`
	// Problem is empty (missing or failed days), missing, failed, empty, mismatched, any or all.
	Problem        string `json:"problem,omitempty" form:"problem"`
	Delisted       *bool  `json:"delisted,omitempty" form:"delisted"`
	DataCollection *bool  `json:"data_collection,omitempty" form:"data_collection"`
}

func (r gapFilterReq) toFilter() repositories.TradeGapFilter {
	return repositories.TradeGapFilter{
		Query:                 r.Query,
		Problem:               repositories.TradeGapProblem(r.Problem),
		Delisted:              r.Delisted,
		DataCollectionEnabled: r.DataCollection,
	}
}

// createJobReq queues a collection. Exactly one of tse_market_ids, all_markets
// and gaps selects the markets. With dates, only those days are fetched;
// otherwise the days between from and to whose status is in day_statuses.
type createJobReq struct {
	TSEMarketIDs []uint        `json:"tse_market_ids,omitempty"`
	AllMarkets   bool          `json:"all_markets,omitempty"`
	Gaps         *gapFilterReq `json:"gaps,omitempty"`
	Dates        []string      `json:"dates,omitempty" example:"2024-05-04"`
	From         string        `json:"from,omitempty" example:"2010-03-21"`
	To           string        `json:"to,omitempty" example:"2024-03-19"`
	// DayStatuses default to MISSING and FAILED.
	DayStatuses []string `json:"day_statuses,omitempty" example:"MISSING,FAILED"`
}

func (r createJobReq) toRequest() (service.JobRequest, error) {
	req := service.JobRequest{TSEMarketIDs: r.TSEMarketIDs, AllMarkets: r.AllMarkets}
	if r.Gaps != nil {
		filter := r.Gaps.toFilter()
		req.Gaps = &filter
	}
	for _, value := range r.Dates {
		date, err := service.ParseDate(value)
		if err != nil {
			return req, err
		}
		req.Dates = append(req.Dates, date)
	}
	var err error
	if req.From, err = optionalDate(r.From); err != nil {
		return req, err
	}
	if req.To, err = optionalDate(r.To); err != nil {
		return req, err
	}
	for _, status := range r.DayStatuses {
		req.DayStatuses = append(req.DayStatuses, consts.TSETradeDayStatus(status))
	}
	return req, nil
}

func optionalDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	date, err := service.ParseDate(value)
	if err != nil {
		return nil, err
	}
	return &date, nil
}

// --- Responses ---

func formatDate(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(dateLayout)
	return &formatted
}

type jobResp struct {
	ID              uint       `json:"id"`
	Status          string     `json:"status"`
	Title           string     `json:"title"`
	AllMarkets      bool       `json:"all_markets"`
	TSEMarketIDs    []uint     `json:"tse_market_ids"`
	Dates           []string   `json:"dates"`
	From            *string    `json:"from"`
	To              *string    `json:"to"`
	DayStatuses     []string   `json:"day_statuses"`
	TotalMarkets    int        `json:"total_markets"`
	DoneMarkets     int        `json:"done_markets"`
	PlannedDays     int64      `json:"planned_days"`
	ProcessedDays   int64      `json:"processed_days"`
	FetchedDays     int64      `json:"fetched_days"`
	EmptyDays       int64      `json:"empty_days"`
	FailedDays      int64      `json:"failed_days"`
	CurrentMarket   string     `json:"current_market"`
	CancelRequested bool       `json:"cancel_requested"`
	Error           string     `json:"error"`
	Worker          string     `json:"worker"`
	CreatedAt       time.Time  `json:"created_at"`
	StartedAt       *time.Time `json:"started_at"`
	HeartbeatAt     *time.Time `json:"heartbeat_at"`
	FinishedAt      *time.Time `json:"finished_at"`
}

func toJobResp(job *models.TSETradeHistoryJob) jobResp {
	statuses := make([]string, len(job.DayStatuses))
	for i, status := range job.DayStatuses {
		statuses[i] = string(status)
	}
	ids := job.TSEMarketIDs
	if ids == nil {
		ids = []uint{}
	}
	dates := job.Dates
	if dates == nil {
		dates = []string{}
	}
	return jobResp{
		ID: job.ID, Status: string(job.Status), Title: job.Title, AllMarkets: job.AllMarkets,
		TSEMarketIDs: ids, Dates: dates, From: formatDate(job.FromDate), To: formatDate(job.ToDate),
		DayStatuses: statuses, TotalMarkets: job.TotalMarkets, DoneMarkets: job.DoneMarkets,
		PlannedDays: job.PlannedDays, ProcessedDays: job.ProcessedDays, FetchedDays: job.FetchedDays,
		EmptyDays: job.EmptyDays, FailedDays: job.FailedDays, CurrentMarket: job.CurrentMarket,
		CancelRequested: job.CancelRequested, Error: job.Error, Worker: job.Worker, CreatedAt: job.CreatedAt,
		StartedAt: job.StartedAt, HeartbeatAt: job.HeartbeatAt, FinishedAt: job.FinishedAt,
	}
}

func toJobResps(jobs []*models.TSETradeHistoryJob) []jobResp {
	data := make([]jobResp, len(jobs))
	for i, job := range jobs {
		data[i] = toJobResp(job)
	}
	return data
}

type dayCountsResp struct {
	TotalDays      int     `json:"total_days"`
	FetchedDays    int     `json:"fetched_days"`
	EmptyDays      int     `json:"empty_days"`
	FailedDays     int     `json:"failed_days"`
	MissingDays    int     `json:"missing_days"`
	MismatchedDays int     `json:"mismatched_days"`
	FirstDate      *string `json:"first_date"`
	LastDate       *string `json:"last_date"`
}

func toDayCountsResp(counts repositories.TradeDayCounts) dayCountsResp {
	return dayCountsResp{
		TotalDays: counts.TotalDays, FetchedDays: counts.FetchedDays, EmptyDays: counts.EmptyDays,
		FailedDays: counts.FailedDays, MissingDays: counts.MissingDays, MismatchedDays: counts.MismatchedDays,
		FirstDate: formatDate(counts.FirstDate), LastDate: formatDate(counts.LastDate),
	}
}

type summaryResp struct {
	TSEMarketID uint   `json:"tse_market_id"`
	InsCode     string `json:"ins_code"`
	Symbol      string `json:"symbol"`
	dayCountsResp
	ActiveJobs []jobResp `json:"active_jobs"`
}

func toSummaryResp(summary *service.MarketSummary) summaryResp {
	return summaryResp{
		TSEMarketID: summary.Market.ID, InsCode: summary.Market.InsCode, Symbol: summary.Market.Label,
		dayCountsResp: toDayCountsResp(summary.TradeDayCounts), ActiveJobs: toJobResps(summary.ActiveJobs),
	}
}

// dayResp is a trading day with the figures of its closing price and of its
// fetched trade history, for comparison.
type dayResp struct {
	Date    string `json:"date"`
	Weekday int    `json:"weekday"`
	// Status is MISSING when the trade history of the day was not fetched yet.
	Status string `json:"status"`
	// Mismatched: the trade count or volume of the trade history, canceled
	// trades left out, differs from the closing price.
	Mismatched bool    `json:"mismatched"`
	TradeCount int64   `json:"trade_count"`
	Volume     float64 `json:"volume"`
	Value      int64   `json:"value"`
	// The History fields are the totals of the received trades.
	HistoryTradeCount    int64      `json:"history_trade_count"`
	HistoryCanceledCount int64      `json:"history_canceled_count"`
	HistoryVolume        int64      `json:"history_volume"`
	HistoryNetVolume     int64      `json:"history_net_volume"`
	HistoryNetValue      int64      `json:"history_net_value"`
	Attempts             int        `json:"attempts"`
	Error                string     `json:"error"`
	JobID                *uint      `json:"job_id"`
	FetchedAt            *time.Time `json:"fetched_at"`
}

func toDayResp(day repositories.TradeDay) dayResp {
	resp := dayResp{
		Date: day.TradeDate.UTC().Format(dateLayout), Weekday: int(day.TradeDate.UTC().Weekday()),
		Status: string(consts.TSETradeDayMissing), Mismatched: day.Mismatched,
		TradeCount: day.TradeCount, Volume: day.Volume, Value: day.Value,
		HistoryTradeCount: day.HistoryTradeCount, HistoryCanceledCount: day.HistoryCanceledCount,
		HistoryVolume: day.HistoryVolume, HistoryNetVolume: day.HistoryNetVolume, HistoryNetValue: day.HistoryNetValue,
		Attempts: day.HistoryAttempts, Error: day.HistoryError, JobID: day.HistoryJobID, FetchedAt: day.HistoryFetchedAt,
	}
	if day.HistoryStatus != nil {
		resp.Status = string(*day.HistoryStatus)
	}
	return resp
}

type fetchDayResp struct {
	Date          string `json:"date"`
	Status        string `json:"status"`
	TradeCount    int64  `json:"trade_count"`
	CanceledCount int64  `json:"canceled_count"`
	Volume        int64  `json:"volume"`
	NetVolume     int64  `json:"net_volume"`
	Attempts      int    `json:"attempts"`
	Error         string `json:"error"`
}

type gapResp struct {
	TSEMarketID           uint   `json:"tse_market_id"`
	MarketID              uint   `json:"market_id"`
	InsCode               string `json:"ins_code"`
	Symbol                string `json:"symbol"`
	Name                  string `json:"name"`
	Delisted              bool   `json:"delisted"`
	DataCollectionEnabled bool   `json:"data_collection_enabled"`
	dayCountsResp
	LastFetchedAt *time.Time `json:"last_fetched_at"`
}

func toGapResp(gap repositories.TradeGap) gapResp {
	return gapResp{
		TSEMarketID: gap.TSEMarketID, MarketID: gap.MarketID, InsCode: gap.InsCode, Symbol: gap.Label, Name: gap.Name,
		Delisted: gap.Delisted, DataCollectionEnabled: gap.DataCollectionEnabled,
		dayCountsResp: toDayCountsResp(gap.TradeDayCounts), LastFetchedAt: gap.LastFetchedAt,
	}
}

type gapTotalsResp struct {
	Markets        int64 `json:"markets"`
	TotalDays      int64 `json:"total_days"`
	FetchedDays    int64 `json:"fetched_days"`
	EmptyDays      int64 `json:"empty_days"`
	FailedDays     int64 `json:"failed_days"`
	MissingDays    int64 `json:"missing_days"`
	MismatchedDays int64 `json:"mismatched_days"`
	// WithoutDays counts the markets without daily closing prices, which have no trading days yet.
	WithoutDays int64 `json:"without_days"`
}
