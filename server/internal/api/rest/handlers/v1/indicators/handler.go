// Package indicators serves the indicator definitions of TSE markets and the
// catalog of indicators they are added from.
package indicators

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	service "github.com/h-varmazyar/gatec/internal/services/indicators"
	"github.com/h-varmazyar/gatec/pkg/paginator"
)

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
}

type catalogResp struct {
	Indicators []service.Definition   `json:"indicators"`
	Categories []service.CategoryInfo `json:"categories"`
	Timeframes []timeframeResp        `json:"timeframes"`
}

// Catalog godoc
// @Summary      List the indicators that can be added
// @Description  The indicators of pkg/indicators with their params, the categories and the TSE timeframes.
// @Tags         Indicators
// @Produce      json
// @Success      200 {object} catalogResp
// @Router       /v1/tse/indicators/catalog [get]
func (h *Handler) Catalog(ctx *gin.Context) {
	timeframes, err := h.service.Timeframes(ctx)
	if err != nil {
		respondError(ctx, err)
		return
	}
	resp := catalogResp{Indicators: service.Catalog(), Categories: service.Categories, Timeframes: make([]timeframeResp, len(timeframes))}
	for i, timeframe := range timeframes {
		resp.Timeframes[i] = timeframeResp{ID: timeframe.ID, Label: timeframe.Label, DurationSeconds: int64(timeframe.Duration / time.Second)}
	}
	ctx.JSON(http.StatusOK, resp)
}

// Markets godoc
// @Summary      List the TSE markets an indicator can be defined on
// @Description  One row per market, by label, with the details of its main instrument.
// @Tags         Indicators
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Items per page" default(20)
// @Param        q query string false "Market label, or symbol, name, insCode or ISIN of an instrument"
// @Param        group query string false "Industry group code"
// @Param        include_delisted query bool false "Keep markets whose instruments are all delisted"
// @Success      200 {object} map[string]interface{}
// @Router       /v1/tse/indicators/markets [get]
func (h *Handler) Markets(ctx *gin.Context) {
	filter := service.MarketFilter{Query: ctx.Query("q"), IndustryGroupCode: ctx.Query("group")}
	if value := ctx.Query("include_delisted"); value != "" {
		include, err := strconv.ParseBool(value)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid include_delisted"})
			return
		}
		filter.IncludeDelisted = include
	}
	pg := paginator.FromContext(ctx)
	markets, total, err := h.service.Markets(ctx, filter, pg)
	if err != nil {
		respondError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": markets, "pagination": pg.ToResponse(total)})
}

type indicatorResp struct {
	ID       uint   `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Title    string `json:"title"`
	Category string `json:"category"`
	// Params are the stored params, with defaults filled in.
	Params      map[string]any `json:"params"`
	Outputs     []string       `json:"outputs"`
	TimeframeID uint           `json:"timeframe_id"`
	Timeframe   string         `json:"timeframe"`
	// Scope is ALL for every TSE market, GROUP for a group of markets and
	// MARKET for one market.
	Scope string `json:"scope" enums:"ALL,GROUP,MARKET"`
	// Markets are the markets of GROUP and MARKET; empty for ALL.
	Markets   []service.MarketRef `json:"markets"`
	Enabled   bool                `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toResp(item service.Item) indicatorResp {
	resp := indicatorResp{
		ID: item.ID, Type: item.Type, Name: item.Type, Title: item.Type, Params: item.Params, Outputs: []string{},
		TimeframeID: item.TimeframeID, Timeframe: item.Timeframe, Scope: string(item.Scope), Markets: item.Markets,
		Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
	if item.Definition != nil {
		resp.Name, resp.Title, resp.Category, resp.Outputs = item.Definition.Name, item.Definition.Title, string(item.Definition.Category), item.Definition.Outputs
	}
	return resp
}

// List godoc
// @Summary      List TSE indicators
// @Tags         Indicators
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Items per page" default(20)
// @Param        q query string false "Type, name or title of the indicator"
// @Param        category query string false "TREND, MOMENTUM, VOLATILITY, VOLUME or PRICE_ACTION"
// @Param        type query string false "Indicator type"
// @Param        timeframe_id query int false "Timeframe ID"
// @Param        market_id query int false "Indicators of the market and of the groups that include it"
// @Param        scope query string false "ALL, GROUP or MARKET"
// @Param        enabled query bool false "Enabled or disabled indicators"
// @Success      200 {object} map[string]interface{}
// @Router       /v1/tse/indicators [get]
func (h *Handler) List(ctx *gin.Context) {
	filter := service.Filter{
		Query: ctx.Query("q"), Category: service.Category(ctx.Query("category")), Type: ctx.Query("type"),
		Scope: service.Scope(ctx.Query("scope")),
	}
	var ok bool
	if filter.TimeframeID, ok = queryID(ctx, "timeframe_id"); !ok {
		return
	}
	if filter.MarketID, ok = queryID(ctx, "market_id"); !ok {
		return
	}
	if value := ctx.Query("enabled"); value != "" {
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid enabled"})
			return
		}
		filter.Enabled = &enabled
	}
	pg := paginator.FromContext(ctx)
	items, total, err := h.service.List(ctx, filter, pg)
	if err != nil {
		respondError(ctx, err)
		return
	}
	data := make([]indicatorResp, len(items))
	for i, item := range items {
		data[i] = toResp(item)
	}
	ctx.JSON(http.StatusOK, gin.H{"data": data, "pagination": pg.ToResponse(total)})
}

// Get godoc
// @Summary      Get a TSE indicator
// @Tags         Indicators
// @Produce      json
// @Param        id path int true "Indicator ID"
// @Success      200 {object} indicatorResp
// @Failure      404 {object} errorResp
// @Router       /v1/tse/indicators/{id} [get]
func (h *Handler) Get(ctx *gin.Context) {
	id, ok := pathID(ctx)
	if !ok {
		return
	}
	item, err := h.service.Get(ctx, id)
	if err != nil {
		respondError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, toResp(*item))
}

// marketsReq selects the markets of an indicator: no market IDs for ALL,
// one for MARKET and the markets of a GROUP; a group of one market is stored
// as that market.
type marketsReq struct {
	Scope     string `json:"scope" binding:"required" enums:"ALL,GROUP,MARKET" example:"ALL"`
	MarketIDs []uint `json:"market_ids"`
}

func (r marketsReq) selection() service.MarketSelection {
	return service.MarketSelection{Scope: service.Scope(r.Scope), MarketIDs: r.MarketIDs}
}

// createReq adds an indicator in each of the timeframes.
type createReq struct {
	Type string `json:"type" binding:"required" example:"RSI"`
	// Params missing from the request take their defaults.
	Params       map[string]any `json:"params"`
	TimeframeIDs []uint         `json:"timeframe_ids" binding:"required"`
	marketsReq
}

// Create godoc
// @Summary      Add a TSE indicator
// @Description  One indicator is added per timeframe; nothing is added when one of them exists already.
// @Tags         Indicators
// @Accept       json
// @Produce      json
// @Param        request body createReq true "Indicator, params, timeframes and market"
// @Success      201 {array} indicatorResp
// @Failure      400 {object} errorResp
// @Failure      409 {object} errorResp
// @Router       /v1/tse/indicators [post]
func (h *Handler) Create(ctx *gin.Context) {
	var body createReq
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request body"})
		return
	}
	items, err := h.service.Create(ctx, service.CreateRequest{
		Type: body.Type, Params: body.Params, TimeframeIDs: body.TimeframeIDs, Markets: body.selection(),
	})
	if err != nil {
		respondError(ctx, err)
		return
	}
	resp := make([]indicatorResp, len(items))
	for i, item := range items {
		resp[i] = toResp(item)
	}
	ctx.JSON(http.StatusCreated, resp)
}

// updateReq replaces the params and markets of an indicator; its type and
// timeframe do not change.
type updateReq struct {
	// Params missing from the request take their defaults.
	Params map[string]any `json:"params"`
	marketsReq
}

// Update godoc
// @Summary      Change the params and markets of a TSE indicator
// @Tags         Indicators
// @Accept       json
// @Produce      json
// @Param        id path int true "Indicator ID"
// @Param        request body updateReq true "Params and markets"
// @Success      200 {object} indicatorResp
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      409 {object} errorResp
// @Router       /v1/tse/indicators/{id} [put]
func (h *Handler) Update(ctx *gin.Context) {
	id, ok := pathID(ctx)
	if !ok {
		return
	}
	var body updateReq
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request body"})
		return
	}
	item, err := h.service.Update(ctx, id, service.UpdateRequest{Params: body.Params, Markets: body.selection()})
	if err != nil {
		respondError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, toResp(*item))
}

type setEnabledReq struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

// SetEnabled godoc
// @Summary      Enable or disable a TSE indicator
// @Tags         Indicators
// @Accept       json
// @Produce      json
// @Param        id path int true "Indicator ID"
// @Param        request body setEnabledReq true "Enabled"
// @Success      200 {object} indicatorResp
// @Failure      404 {object} errorResp
// @Router       /v1/tse/indicators/{id}/enabled [patch]
func (h *Handler) SetEnabled(ctx *gin.Context) {
	id, ok := pathID(ctx)
	if !ok {
		return
	}
	var body setEnabledReq
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "enabled is required"})
		return
	}
	item, err := h.service.SetEnabled(ctx, id, *body.Enabled)
	if err != nil {
		respondError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, toResp(*item))
}

// Delete godoc
// @Summary      Delete a TSE indicator
// @Tags         Indicators
// @Param        id path int true "Indicator ID"
// @Success      204
// @Failure      404 {object} errorResp
// @Router       /v1/tse/indicators/{id} [delete]
func (h *Handler) Delete(ctx *gin.Context) {
	id, ok := pathID(ctx)
	if !ok {
		return
	}
	if err := h.service.Delete(ctx, id); err != nil {
		respondError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
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
	case errors.Is(err, service.ErrConflict):
		status = http.StatusConflict
	}
	ctx.JSON(status, errorResp{Error: err.Error()})
}
