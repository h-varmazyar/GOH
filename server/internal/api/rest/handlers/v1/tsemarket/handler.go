package tsemarket

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/internal/services/tsemarket"
	"github.com/h-varmazyar/gatec/pkg/paginator"
)

// Handler handles TSE market management REST API endpoints.
type Handler struct {
	service *tsemarket.Service
}

// New creates a new TSE market handler.
func New(service *tsemarket.Service) Handler {
	return Handler{service: service}
}

// HasService returns true if the handler has a valid service.
func (h *Handler) HasService() bool {
	return h.service != nil
}

// ListMarkets godoc
// @Summary      List TSE markets
// @Description  Paginated TSE stocks, funds and rights with their data collection and evaluation flags
// @Tags         TSE Market
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Items per page" default(20)
// @Param        q query string false "Symbol, name, insCode or instrument ID"
// @Param        market_id query int false "Market ID; lists the insCodes of one market"
// @Param        market_type query string false "BOURSE, IFB_FIRST, IFB_SECOND, IFB_BASE, IFB_NOAFARIN, IFB_AGREEMENT or IFB"
// @Param        kind query string false "STOCK, FUND or RIGHT"
// @Param        group query string false "Industry group code"
// @Param        delisted query bool false "Delisted instruments"
// @Param        data_collection query bool false "Data collection flag"
// @Param        evaluation query bool false "Evaluation flag"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/tse/markets [get]
func (h *Handler) ListMarkets(ctx *gin.Context) {
	var req filterReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid filter"})
		return
	}
	pg := paginator.FromContext(ctx)
	items, members, total, err := h.service.List(ctx, req.toFilter(), pg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}
	data := make([]marketResp, len(items))
	for i, item := range items {
		data[i] = toMarketResp(item, members[item.MarketID])
	}
	ctx.JSON(http.StatusOK, gin.H{
		"data":       data,
		"pagination": pg.ToResponse(total),
	})
}

// ListIndustryGroups godoc
// @Summary      List TSE industry groups
// @Tags         TSE Market
// @Produce      json
// @Success      200 {array} industryGroupResp
// @Failure      500 {object} errorResp
// @Router       /v1/tse/markets/industry-groups [get]
func (h *Handler) ListIndustryGroups(ctx *gin.Context) {
	groups, err := h.service.IndustryGroups(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}
	data := make([]industryGroupResp, len(groups))
	for i, group := range groups {
		data[i] = industryGroupResp{Code: group.Code, Name: group.Name}
	}
	ctx.JSON(http.StatusOK, gin.H{"data": data})
}

// CreateMarket godoc
// @Summary      Add a TSE market by hand
// @Description  Adds a symbol by its insCode; the other data is read from TSETMC and the TSE and IFB listings.
// @Description  The insCode joins the market of its symbol.
// @Tags         TSE Market
// @Accept       json
// @Produce      json
// @Param        request body createMarketReq true "Symbol and insCode"
// @Success      201 {object} marketResp
// @Failure      400 {object} errorResp
// @Failure      409 {object} errorResp
// @Failure      500 {object} errorResp
// @Failure      503 {object} errorResp
// @Router       /v1/tse/markets [post]
func (h *Handler) CreateMarket(ctx *gin.Context) {
	var req createMarketReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "symbol and ins_code are required"})
		return
	}
	market, err := h.service.AddManual(ctx, req.Symbol, req.InsCode)
	if err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}
	h.respond(ctx, http.StatusCreated, market)
}

// RefreshMarket godoc
// @Summary      Read a TSE market again from its sources
// @Tags         TSE Market
// @Produce      json
// @Param        id path int true "TSE market ID"
// @Success      200 {object} marketResp
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      500 {object} errorResp
// @Failure      503 {object} errorResp
// @Router       /v1/tse/markets/{id}/refresh [post]
func (h *Handler) RefreshMarket(ctx *gin.Context) {
	id, ok := pathID(ctx)
	if !ok {
		return
	}
	market, err := h.service.Refresh(ctx, id)
	if err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}
	h.respond(ctx, http.StatusOK, market)
}

// UpdateDelisted godoc
// @Summary      Mark a TSE market as delisted or listed
// @Description  Catalog syncs set delisted marks but never clear them
// @Tags         TSE Market
// @Accept       json
// @Produce      json
// @Param        id path int true "TSE market ID"
// @Param        request body delistedReq true "Delisted state"
// @Success      200 {object} marketResp
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/tse/markets/{id}/delisted [patch]
func (h *Handler) UpdateDelisted(ctx *gin.Context) {
	id, ok := pathID(ctx)
	if !ok {
		return
	}
	var req delistedReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "delisted is required"})
		return
	}
	market, err := h.service.SetDelisted(ctx, id, *req.Delisted)
	if err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}
	h.respond(ctx, http.StatusOK, market)
}

// LinkMarket godoc
// @Summary      Move a TSE market to another market
// @Description  Links the insCode to the market with the given label, or to a new market with that label.
// @Description  Only insCodes without stored prices can move.
// @Tags         TSE Market
// @Accept       json
// @Produce      json
// @Param        id path int true "TSE market ID"
// @Param        request body linkReq true "Market label"
// @Success      200 {object} marketResp
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/tse/markets/{id}/market [patch]
func (h *Handler) LinkMarket(ctx *gin.Context) {
	id, ok := pathID(ctx)
	if !ok {
		return
	}
	var req linkReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "market_label is required"})
		return
	}
	market, err := h.service.Link(ctx, id, req.MarketLabel)
	if err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}
	h.respond(ctx, http.StatusOK, market)
}

// UpdateFlags godoc
// @Summary      Change the flags of a TSE market
// @Tags         TSE Market
// @Accept       json
// @Produce      json
// @Param        id path int true "TSE market ID"
// @Param        request body flagsReq true "Flags to change"
// @Success      200 {object} marketResp
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/tse/markets/{id}/flags [patch]
func (h *Handler) UpdateFlags(ctx *gin.Context) {
	id, ok := pathID(ctx)
	if !ok {
		return
	}
	var req flagsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request body"})
		return
	}
	market, err := h.service.SetFlags(ctx, id, tsemarket.Flags{
		DataCollectionEnabled: req.DataCollectionEnabled,
		EvaluationEnabled:     req.EvaluationEnabled,
	})
	if err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}
	h.respond(ctx, http.StatusOK, market)
}

// BulkUpdateFlags godoc
// @Summary      Change the flags of several TSE markets
// @Description  Changes the markets in ids, or every market matching filter when ids is empty
// @Tags         TSE Market
// @Accept       json
// @Produce      json
// @Param        request body bulkFlagsReq true "Markets and flags"
// @Success      200 {object} bulkFlagsResp
// @Failure      400 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/tse/markets/flags [patch]
func (h *Handler) BulkUpdateFlags(ctx *gin.Context) {
	var req bulkFlagsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request body"})
		return
	}
	var filter *repositories.TSEMarketFilter
	if len(req.IDs) == 0 && req.Filter != nil {
		value := req.Filter.toFilter()
		filter = &value
	}
	updated, err := h.service.BulkSetFlags(ctx, req.IDs, filter, tsemarket.Flags{
		DataCollectionEnabled: req.DataCollectionEnabled,
		EvaluationEnabled:     req.EvaluationEnabled,
	})
	if err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, bulkFlagsResp{Updated: updated})
}

// respond writes a market with the number of instruments of its market.
// DeleteMarket godoc
// @Summary      Delete a TSE market
// @Description  Deletes the insCode with its daily prices and candles, or with whole_market every insCode of its market.
// @Description  A market left without insCodes is deleted with all of its data. insCodes of the TSE catalog are added again by the next catalog sync.
// @Tags         TSE Market
// @Produce      json
// @Param        id path int true "TSE market ID"
// @Param        whole_market query bool false "Delete every insCode of the market"
// @Success      200 {object} deleteResp
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/tse/markets/{id} [delete]
func (h *Handler) DeleteMarket(ctx *gin.Context) {
	id, ok := pathID(ctx)
	if !ok {
		return
	}
	wholeMarket, _ := strconv.ParseBool(ctx.Query("whole_market"))
	result, err := h.service.Delete(ctx, id, wholeMarket)
	if err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, deleteResp{DeletedInsCodes: result.InsCodes, DeletedMarkets: result.Markets})
}

func (h *Handler) respond(ctx *gin.Context, status int, market *models.TSEMarket) {
	members, err := h.service.MemberCount(ctx, market.MarketID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}
	ctx.JSON(status, toMarketResp(market, members))
}

func pathID(ctx *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid id"})
		return 0, false
	}
	return uint(id), true
}

func httpStatusFromError(err error) int {
	switch {
	case errors.Is(err, tsemarket.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, tsemarket.ErrAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, tsemarket.ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, tsemarket.ErrNoAdapter):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
