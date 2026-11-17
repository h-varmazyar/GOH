// Package candles serves the candles built from trades and the candle jobs
// that build them; the candle-builder worker runs the queued jobs.
package candles

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	service "github.com/h-varmazyar/gatec/internal/services/candles"
	"github.com/h-varmazyar/gatec/pkg/paginator"
)

const dateLayout = "2006-01-02"

type Handler struct {
	service *service.Service
}

func New(service *service.Service) Handler { return Handler{service: service} }

func (h *Handler) HasService() bool { return h.service != nil }

type errorResp struct {
	Error string `json:"error"`
}

type timeframeResp struct {
	ID              uint   `json:"id"`
	Label           string `json:"label"`
	DurationSeconds int64  `json:"duration_seconds"`
	ProviderID      uint   `json:"provider_id"`
}

// Timeframes godoc
// @Summary      List the timeframes candles are built in
// @Description  The supported timeframes of the provider (5m, 15m, 1h, 4h, 1w and 1M, the Jalali month); missing ones are created.
// @Tags         Candles
// @Produce      json
// @Param        provider_id query int false "Provider ID; the TSE provider when not set"
// @Success      200 {array} timeframeResp
// @Router       /v1/candles/timeframes [get]
func (h *Handler) Timeframes(ctx *gin.Context) {
	providerID, ok := queryID(ctx, "provider_id")
	if !ok {
		return
	}
	if providerID == 0 {
		provider, err := h.service.TSEProvider(ctx)
		if err != nil {
			respondError(ctx, err)
			return
		}
		providerID = provider.ID
	} else if _, err := h.service.Provider(ctx, providerID); err != nil {
		respondError(ctx, err)
		return
	}
	timeframes, err := h.service.EnsureTimeframes(ctx, providerID)
	if err != nil {
		respondError(ctx, err)
		return
	}
	resp := make([]timeframeResp, len(timeframes))
	for i, timeframe := range timeframes {
		resp[i] = timeframeResp{ID: timeframe.ID, Label: timeframe.Label,
			DurationSeconds: int64(timeframe.Duration / time.Second), ProviderID: timeframe.ProviderID}
	}
	ctx.JSON(http.StatusOK, resp)
}

// Candles godoc
// @Summary      List the candles of a market
// @Description  Candles built from trades, oldest first; time is the Unix time the candle opens at. A candle
// @Description  with closed false is still open, such as the weekly candle of the current week.
// @Tags         Candles
// @Produce      json
// @Param        id path int true "Market ID"
// @Param        timeframe_id query int true "Timeframe ID"
// @Param        from query string false "YYYY-MM-DD"
// @Param        to query string false "YYYY-MM-DD, included"
// @Param        limit query int false "Only the newest candles" default(500)
// @Success      200 {array} models.MarketCandle
// @Failure      400 {object} errorResp
// @Router       /v1/candles/markets/{id} [get]
func (h *Handler) Candles(ctx *gin.Context) {
	marketID, ok := pathID(ctx)
	if !ok {
		return
	}
	timeframeID, ok := queryID(ctx, "timeframe_id")
	if !ok {
		return
	}
	if timeframeID == 0 {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "timeframe_id is required"})
		return
	}
	from, err := optionalDate(ctx.Query("from"))
	if err != nil {
		respondError(ctx, err)
		return
	}
	to, err := optionalDate(ctx.Query("to"))
	if err != nil {
		respondError(ctx, err)
		return
	}
	if to != nil {
		// The candles of the whole day are included.
		end := to.AddDate(0, 0, 1).Add(-time.Second)
		to = &end
	}
	limit := 500
	if value := ctx.Query("limit"); value != "" {
		if limit, err = strconv.Atoi(value); err != nil || limit < 0 {
			ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid limit"})
			return
		}
	}
	candles, err := h.service.Candles(ctx, marketID, timeframeID, from, to, limit)
	if err != nil {
		respondError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, candles)
}

// createJobReq queues a candle build. Exactly one of market_ids and
// all_markets selects the markets; the markets and timeframes must belong to
// the provider. With from = to a single day is built; the weekly candle of
// that day is built from its whole week.
type createJobReq struct {
	// ProviderID is the TSE provider when not set.
	ProviderID uint   `json:"provider_id,omitempty"`
	MarketIDs  []uint `json:"market_ids,omitempty"`
	AllMarkets bool   `json:"all_markets,omitempty"`
	// TimeframeIDs default to every supported timeframe of the provider.
	TimeframeIDs []uint `json:"timeframe_ids,omitempty"`
	From         string `json:"from,omitempty" example:"2026-11-07"`
	To           string `json:"to,omitempty" example:"2026-11-11"`
	// AfterTradeHistoryJobID holds the job until that trade history job has finished.
	AfterTradeHistoryJobID *uint `json:"after_trade_history_job_id,omitempty"`
}

// CreateJob godoc
// @Summary      Queue a candle build
// @Description  The candle-builder worker builds the candles of the markets from their stored trades.
// @Tags         Candles
// @Accept       json
// @Produce      json
// @Param        request body createJobReq true "Markets, timeframes and days"
// @Success      201 {object} jobResp
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Router       /v1/candles/jobs [post]
func (h *Handler) CreateJob(ctx *gin.Context) {
	var body createJobReq
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request body"})
		return
	}
	req := service.JobRequest{
		ProviderID: body.ProviderID, MarketIDs: body.MarketIDs, AllMarkets: body.AllMarkets,
		TimeframeIDs: body.TimeframeIDs, AfterTradeHistoryJobID: body.AfterTradeHistoryJobID,
	}
	if req.ProviderID == 0 {
		provider, err := h.service.TSEProvider(ctx)
		if err != nil {
			respondError(ctx, err)
			return
		}
		req.ProviderID = provider.ID
	}
	var err error
	if req.From, err = optionalDate(body.From); err != nil {
		respondError(ctx, err)
		return
	}
	if req.To, err = optionalDate(body.To); err != nil {
		respondError(ctx, err)
		return
	}
	job, err := h.service.CreateJob(ctx, req)
	if err != nil {
		respondError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, toJobResp(job))
}

// ListJobs godoc
// @Summary      List candle jobs
// @Tags         Candles
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Items per page" default(20)
// @Param        active query bool false "Only queued and running jobs"
// @Param        status query string false "PENDING, RUNNING, DONE, FAILED or CANCELED"
// @Param        market_id query int false "Jobs of this market and of every market"
// @Success      200 {object} map[string]interface{}
// @Router       /v1/candles/jobs [get]
func (h *Handler) ListJobs(ctx *gin.Context) {
	filter := repositories.CandleJobFilter{}
	if active, _ := strconv.ParseBool(ctx.Query("active")); active {
		filter.Statuses = consts.JobActive
	} else if status := ctx.Query("status"); status != "" {
		filter.Statuses = []consts.JobStatus{consts.JobStatus(status)}
	}
	marketID, ok := queryID(ctx, "market_id")
	if !ok {
		return
	}
	filter.MarketID = marketID
	pg := paginator.FromContext(ctx)
	jobs, total, err := h.service.ListJobs(ctx, filter, pg)
	if err != nil {
		respondError(ctx, err)
		return
	}
	data := make([]jobResp, len(jobs))
	for i, job := range jobs {
		data[i] = toJobResp(job)
	}
	ctx.JSON(http.StatusOK, gin.H{"data": data, "pagination": pg.ToResponse(total)})
}

// GetJob godoc
// @Summary      Get a candle job
// @Tags         Candles
// @Produce      json
// @Param        id path int true "Job ID"
// @Success      200 {object} jobResp
// @Failure      404 {object} errorResp
// @Router       /v1/candles/jobs/{id} [get]
func (h *Handler) GetJob(ctx *gin.Context) {
	id, ok := pathID(ctx)
	if !ok {
		return
	}
	job, err := h.service.GetJob(ctx, id)
	if err != nil {
		respondError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, toJobResp(job))
}

// CancelJob godoc
// @Summary      Cancel a candle job
// @Description  A queued job is canceled at once; a running job stops within seconds.
// @Tags         Candles
// @Produce      json
// @Param        id path int true "Job ID"
// @Success      200 {object} jobResp
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Router       /v1/candles/jobs/{id}/cancel [post]
func (h *Handler) CancelJob(ctx *gin.Context) {
	id, ok := pathID(ctx)
	if !ok {
		return
	}
	job, err := h.service.CancelJob(ctx, id)
	if err != nil {
		respondError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, toJobResp(job))
}

type jobResp struct {
	ID                     uint       `json:"id"`
	Status                 string     `json:"status"`
	Title                  string     `json:"title"`
	ProviderID             uint       `json:"provider_id"`
	AllMarkets             bool       `json:"all_markets"`
	MarketIDs              []uint     `json:"market_ids"`
	TimeframeIDs           []uint     `json:"timeframe_ids"`
	From                   *string    `json:"from"`
	To                     *string    `json:"to"`
	AfterTradeHistoryJobID *uint      `json:"after_trade_history_job_id"`
	TotalMarkets           int        `json:"total_markets"`
	DoneMarkets            int        `json:"done_markets"`
	EmptyMarkets           int        `json:"empty_markets"`
	FailedMarkets          int        `json:"failed_markets"`
	Candles                int64      `json:"candles"`
	CurrentMarket          string     `json:"current_market"`
	CancelRequested        bool       `json:"cancel_requested"`
	Error                  string     `json:"error"`
	Worker                 string     `json:"worker"`
	CreatedAt              time.Time  `json:"created_at"`
	StartedAt              *time.Time `json:"started_at"`
	HeartbeatAt            *time.Time `json:"heartbeat_at"`
	FinishedAt             *time.Time `json:"finished_at"`
}

func toJobResp(job *models.CandleJob) jobResp {
	nonNil := func(ids []uint) []uint {
		if ids == nil {
			return []uint{}
		}
		return ids
	}
	return jobResp{
		ID: job.ID, Status: string(job.Status), Title: job.Title, ProviderID: job.ProviderID,
		AllMarkets: job.AllMarkets, MarketIDs: nonNil(job.MarketIDs), TimeframeIDs: nonNil(job.TimeframeIDs),
		From: formatDate(job.FromDate), To: formatDate(job.ToDate), AfterTradeHistoryJobID: job.AfterTradeHistoryJobID,
		TotalMarkets: job.TotalMarkets, DoneMarkets: job.DoneMarkets, EmptyMarkets: job.EmptyMarkets,
		FailedMarkets: job.FailedMarkets, Candles: job.Candles, CurrentMarket: job.CurrentMarket,
		CancelRequested: job.CancelRequested, Error: job.Error, Worker: job.Worker, CreatedAt: job.CreatedAt,
		StartedAt: job.StartedAt, HeartbeatAt: job.HeartbeatAt, FinishedAt: job.FinishedAt,
	}
}

func formatDate(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(dateLayout)
	return &formatted
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

func pathID(ctx *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid id"})
		return 0, false
	}
	return uint(id), true
}

// queryID reads an optional ID query parameter; 0 when it is not set.
func queryID(ctx *gin.Context, name string) (uint, bool) {
	value := ctx.Query(name)
	if value == "" {
		return 0, true
	}
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid " + name})
		return 0, false
	}
	return uint(id), true
}

func respondError(ctx *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, service.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, service.ErrNoStorage):
		status = http.StatusServiceUnavailable
	}
	ctx.JSON(status, errorResp{Error: err.Error()})
}
