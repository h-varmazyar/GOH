package tradehistory

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/repositories"
	service "github.com/h-varmazyar/gatec/internal/services/tradehistory"
	"github.com/h-varmazyar/gatec/pkg/paginator"
)

// Handler serves the TSE trade history pages of the panel. Quick actions call
// the service; long collections are queued as jobs for the tse-trade-history worker.
type Handler struct {
	service *service.Service
}

func New(service *service.Service) Handler {
	return Handler{service: service}
}

func (h *Handler) HasService() bool { return h.service != nil }

// MarketSummary godoc
// @Summary      Count the trading days of a TSE market by trade history status
// @Description  The trading days are the daily closing price rows of the market.
// @Tags         TSE Trade History
// @Produce      json
// @Param        id path int true "TSE market ID"
// @Success      200 {object} summaryResp
// @Failure      404 {object} errorResp
// @Router       /v1/tse/trade-history/markets/{id}/summary [get]
func (h *Handler) MarketSummary(ctx *gin.Context) {
	id, ok := pathID(ctx)
	if !ok {
		return
	}
	summary, err := h.service.MarketSummary(ctx, id)
	if err != nil {
		respondError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, toSummaryResp(summary))
}

// ListDays godoc
// @Summary      List the trading days of a TSE market with their trade history
// @Description  The daily closing price rows of the market, newest first, with the totals of their fetched
// @Description  trade history (canceled trades included and left out) for comparison. MISSING days were not fetched yet.
// @Tags         TSE Trade History
// @Produce      json
// @Param        id path int true "TSE market ID"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Items per page" default(20)
// @Param        status query string false "FETCHED, EMPTY, FAILED, MISSING or MISMATCHED"
// @Param        from query string false "YYYY-MM-DD"
// @Param        to query string false "YYYY-MM-DD"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} errorResp
// @Router       /v1/tse/trade-history/markets/{id}/days [get]
func (h *Handler) ListDays(ctx *gin.Context) {
	id, ok := pathID(ctx)
	if !ok {
		return
	}
	var req dayFilterReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid filter"})
		return
	}
	filter := repositories.TradeDayFilter{Status: consts.TSETradeDayStatus(req.Status)}
	var err error
	if filter.From, err = optionalDate(req.From); err != nil {
		respondError(ctx, err)
		return
	}
	if filter.To, err = optionalDate(req.To); err != nil {
		respondError(ctx, err)
		return
	}
	pg := paginator.FromContext(ctx)
	days, total, err := h.service.ListDays(ctx, id, filter, pg)
	if err != nil {
		respondError(ctx, err)
		return
	}
	data := make([]dayResp, len(days))
	for i, day := range days {
		data[i] = toDayResp(day)
	}
	ctx.JSON(http.StatusOK, gin.H{"data": data, "pagination": pg.ToResponse(total)})
}

// FetchDay godoc
// @Summary      Fetch one trade history day of a TSE market now
// @Description  Fetches the trades of the day right away and records the result; a failure is recorded as FAILED.
// @Tags         TSE Trade History
// @Produce      json
// @Param        id path int true "TSE market ID"
// @Param        date path string true "YYYY-MM-DD"
// @Success      200 {object} fetchDayResp
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      503 {object} errorResp
// @Router       /v1/tse/trade-history/markets/{id}/days/{date}/fetch [post]
func (h *Handler) FetchDay(ctx *gin.Context) {
	id, ok := pathID(ctx)
	if !ok {
		return
	}
	date, err := service.ParseDate(ctx.Param("date"))
	if err != nil {
		respondError(ctx, err)
		return
	}
	result, err := h.service.FetchDay(ctx, id, date)
	if err != nil {
		respondError(ctx, err)
		return
	}
	resp := fetchDayResp{
		Date: result.Date.Format(dateLayout), Status: string(result.Status), TradeCount: result.TradeCount,
		CanceledCount: result.CanceledCount, Volume: result.Volume, NetVolume: result.NetVolume, Attempts: result.Attempts,
	}
	if result.Err != nil {
		resp.Error = result.Err.Error()
	}
	ctx.JSON(http.StatusOK, resp)
}

// ListGaps godoc
// @Summary      List TSE markets with incomplete trade history
// @Description  Markets with their expected, fetched, empty, failed and missing days, the most incomplete first.
// @Tags         TSE Trade History
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Items per page" default(20)
// @Param        q query string false "Symbol, name or insCode"
// @Param        problem query string false "Empty for missing or failed days; missing, failed, empty, mismatched, any or all"
// @Param        delisted query bool false "Delisted markets"
// @Param        data_collection query bool false "Data collection flag"
// @Success      200 {object} map[string]interface{}
// @Router       /v1/tse/trade-history/gaps [get]
func (h *Handler) ListGaps(ctx *gin.Context) {
	var req gapFilterReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid filter"})
		return
	}
	pg := paginator.FromContext(ctx)
	page, err := h.service.ListGaps(ctx, req.toFilter(), pg)
	if err != nil {
		respondError(ctx, err)
		return
	}
	data := make([]gapResp, len(page.Gaps))
	for i, gap := range page.Gaps {
		data[i] = toGapResp(gap)
	}
	totals := page.Totals
	ctx.JSON(http.StatusOK, gin.H{
		"data":       data,
		"pagination": pg.ToResponse(page.Total),
		"totals": gapTotalsResp{
			Markets: totals.Markets, TotalDays: totals.TotalDays, FetchedDays: totals.FetchedDays,
			EmptyDays: totals.EmptyDays, FailedDays: totals.FailedDays, MissingDays: totals.MissingDays,
			MismatchedDays: totals.MismatchedDays, WithoutDays: page.WithoutDays,
		},
	})
}

// CreateJob godoc
// @Summary      Queue a trade history collection
// @Description  The tse-trade-history worker runs queued jobs one by one.
// @Tags         TSE Trade History
// @Accept       json
// @Produce      json
// @Param        request body createJobReq true "Markets and days"
// @Success      201 {object} jobResp
// @Failure      400 {object} errorResp
// @Router       /v1/tse/trade-history/jobs [post]
func (h *Handler) CreateJob(ctx *gin.Context) {
	var body createJobReq
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request body"})
		return
	}
	req, err := body.toRequest()
	if err != nil {
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
// @Summary      List trade history jobs
// @Tags         TSE Trade History
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Items per page" default(20)
// @Param        active query bool false "Only queued and running jobs"
// @Param        status query string false "PENDING, RUNNING, DONE, FAILED or CANCELED"
// @Param        tse_market_id query int false "Jobs of this market and of every market"
// @Success      200 {object} map[string]interface{}
// @Router       /v1/tse/trade-history/jobs [get]
func (h *Handler) ListJobs(ctx *gin.Context) {
	filter := repositories.TradeHistoryJobFilter{}
	if active, _ := strconv.ParseBool(ctx.Query("active")); active {
		filter.Statuses = consts.TSETradeHistoryJobActive
	} else if status := ctx.Query("status"); status != "" {
		filter.Statuses = []consts.TSETradeHistoryJobStatus{consts.TSETradeHistoryJobStatus(status)}
	}
	if value := ctx.Query("tse_market_id"); value != "" {
		id, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid tse_market_id"})
			return
		}
		filter.TSEMarketID = uint(id)
	}
	pg := paginator.FromContext(ctx)
	jobs, total, err := h.service.ListJobs(ctx, filter, pg)
	if err != nil {
		respondError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": toJobResps(jobs), "pagination": pg.ToResponse(total)})
}

// GetJob godoc
// @Summary      Get a trade history job
// @Tags         TSE Trade History
// @Produce      json
// @Param        id path int true "Job ID"
// @Success      200 {object} jobResp
// @Failure      404 {object} errorResp
// @Router       /v1/tse/trade-history/jobs/{id} [get]
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
// @Summary      Cancel a trade history job
// @Description  A queued job is canceled at once; a running job stops within seconds.
// @Tags         TSE Trade History
// @Produce      json
// @Param        id path int true "Job ID"
// @Success      200 {object} jobResp
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Router       /v1/tse/trade-history/jobs/{id}/cancel [post]
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

func pathID(ctx *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid id"})
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
	case errors.Is(err, service.ErrNoAdapter), errors.Is(err, service.ErrNoStorage):
		status = http.StatusServiceUnavailable
	}
	ctx.JSON(status, errorResp{Error: err.Error()})
}
