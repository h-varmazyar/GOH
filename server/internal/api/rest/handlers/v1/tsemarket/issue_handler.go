package tsemarket

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/h-varmazyar/gatec/internal/consts"
	"github.com/h-varmazyar/gatec/internal/repositories"
	"github.com/h-varmazyar/gatec/pkg/paginator"
)

// ListIssues godoc
// @Summary      List TSE catalog issues
// @Description  Issues that need a decision, such as insCodes linked to the wrong market, with the insCodes they are about
// @Tags         TSE Market Issue
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Items per page" default(20)
// @Param        q query string false "Symbol, name, detail or insCode"
// @Param        type query string false "Issue type"
// @Param        severity query string false "ERROR, WARNING or INFO"
// @Param        status query string false "OPEN, IGNORED or RESOLVED"
// @Param        ins_code query string false "Issues about this insCode"
// @Param        market_id query int false "Issues about this market"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/tse/market-issues [get]
func (h *Handler) ListIssues(ctx *gin.Context) {
	var req issueFilterReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid filter"})
		return
	}
	pg := paginator.FromContext(ctx)
	views, members, total, err := h.service.ListIssues(ctx, req.toFilter(), pg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}
	data := make([]issueResp, len(views))
	for i, view := range views {
		data[i] = toIssueResp(view, members)
	}
	ctx.JSON(http.StatusOK, gin.H{
		"data":       data,
		"pagination": pg.ToResponse(total),
	})
}

// IssueSummary godoc
// @Summary      Count TSE catalog issues
// @Description  Issue counts by type, severity and status, and the Persian labels of the issue types
// @Tags         TSE Market Issue
// @Produce      json
// @Success      200 {object} issueSummaryResp
// @Failure      500 {object} errorResp
// @Router       /v1/tse/market-issues/summary [get]
func (h *Handler) IssueSummary(ctx *gin.Context) {
	counts, err := h.service.IssueSummary(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}
	resp := issueSummaryResp{Types: []issueTypeResp{}, Counts: make([]issueCountResp, len(counts))}
	for _, issueType := range append(append([]consts.TSEIssueType{}, consts.TSEIssueCheckTypes...), consts.TSEIssueSyncTypes...) {
		resp.Types = append(resp.Types, issueTypeResp{Type: string(issueType), Label: issueType.PersianLabel()})
	}
	for i, count := range counts {
		resp.Counts[i] = issueCountResp{Type: string(count.Type), Severity: string(count.Severity), Status: string(count.Status), Count: count.Count}
	}
	ctx.JSON(http.StatusOK, resp)
}

// DetectIssues godoc
// @Summary      Check the stored TSE markets for issues
// @Description  Finds insCodes that may be linked to the wrong market and prices that share candles; issues no longer found are resolved
// @Tags         TSE Market Issue
// @Produce      json
// @Success      200 {object} detectResp
// @Failure      500 {object} errorResp
// @Router       /v1/tse/market-issues/detect [post]
func (h *Handler) DetectIssues(ctx *gin.Context) {
	result, err := h.service.DetectIssues(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, detectResp{Found: result.Found, Resolved: result.Resolved})
}

// UpdateIssueStatus godoc
// @Summary      Open or ignore TSE catalog issues
// @Description  Changes the issues in ids, or every issue matching filter when ids is empty. Ignored issues stay ignored when found again.
// @Tags         TSE Market Issue
// @Accept       json
// @Produce      json
// @Param        request body issueStatusReq true "Issues and status"
// @Success      200 {object} issueStatusResp
// @Failure      400 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/tse/market-issues/status [patch]
func (h *Handler) UpdateIssueStatus(ctx *gin.Context) {
	var req issueStatusReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "status must be OPEN or IGNORED"})
		return
	}
	var filter *repositories.TSEIssueFilter
	if len(req.IDs) == 0 && req.Filter != nil {
		value := req.Filter.toFilter()
		filter = &value
	}
	updated, err := h.service.SetIssueStatus(ctx, req.IDs, filter, consts.TSEIssueStatus(req.Status), req.Note)
	if err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, issueStatusResp{Updated: updated})
}

// GetMarket godoc
// @Summary      Get a TSE market
// @Tags         TSE Market
// @Produce      json
// @Param        id path int true "TSE market ID"
// @Success      200 {object} marketResp
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/tse/markets/{id} [get]
func (h *Handler) GetMarket(ctx *gin.Context) {
	id, ok := pathID(ctx)
	if !ok {
		return
	}
	market, err := h.service.Get(ctx, id)
	if err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}
	h.respond(ctx, http.StatusOK, market)
}
