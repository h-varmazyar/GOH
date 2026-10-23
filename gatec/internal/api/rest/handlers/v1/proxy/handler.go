package proxy

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/h-varmazyar/gatec/internal/models" // Referenced by Swagger annotations.
	"github.com/h-varmazyar/gatec/internal/services/proxy"
	"github.com/h-varmazyar/gatec/pkg/paginator"
)

// Handler handles proxy management REST API endpoints.
type Handler struct {
	service *proxy.Service
}

// New creates a new proxy handler.
func New(service *proxy.Service) Handler {
	return Handler{service: service}
}

// HasService returns true if the handler has a valid service.
func (h *Handler) HasService() bool {
	return h.service != nil
}

// ListProxies godoc
// @Summary      List proxies
// @Description  Get paginated list of all proxies
// @Tags         Proxy
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Items per page" default(20)
// @Success      200 {object} map[string]interface{}
// @Failure      500 {object} errorResp
// @Router       /v1/proxies [get]
func (h *Handler) ListProxies(ctx *gin.Context) {
	pg := paginator.FromContext(ctx)
	items, total, err := h.service.List(ctx, pg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"data":       items,
		"pagination": pg.ToResponse(total),
	})
}

// ListActiveProxies godoc
// @Summary      List active proxies
// @Description  Get list of all active (enabled + active status) proxies
// @Tags         Proxy
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      500 {object} errorResp
// @Router       /v1/proxies/active [get]
func (h *Handler) ListActiveProxies(ctx *gin.Context) {
	items, err := h.service.ListActive(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": items})
}

// ListEnabledProxies godoc
// @Summary      List enabled proxies
// @Description  Get list of all enabled proxies (regardless of status)
// @Tags         Proxy
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      500 {object} errorResp
// @Router       /v1/proxies/enabled [get]
func (h *Handler) ListEnabledProxies(ctx *gin.Context) {
	items, err := h.service.ListEnabled(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": items})
}

// GetProxy godoc
// @Summary      Get proxy details
// @Description  Get detailed information about a proxy
// @Tags         Proxy
// @Produce      json
// @Param        id path int true "Proxy ID"
// @Success      200 {object} models.Proxy
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/proxies/{id} [get]
func (h *Handler) GetProxy(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid id"})
		return
	}

	proxy, err := h.service.GetByID(ctx, uint(id))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			ctx.JSON(http.StatusNotFound, errorResp{Error: err.Error()})
		} else {
			ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		}
		return
	}

	ctx.JSON(http.StatusOK, proxy)
}

// CreateProxy godoc
// @Summary      Create a new proxy
// @Description  Add a new proxy server to the system
// @Tags         Proxy
// @Accept       json
// @Produce      json
// @Param        request body createProxyReq true "Proxy configuration"
// @Success      201 {object} models.Proxy
// @Failure      400 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/proxies [post]
func (h *Handler) CreateProxy(ctx *gin.Context) {
	var req createProxyReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request body"})
		return
	}

	proxy := req.ToModel()
	if err := h.service.Create(ctx, proxy); err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, proxy)
}

// UpdateProxy godoc
// @Summary      Update a proxy
// @Description  Update an existing proxy configuration
// @Tags         Proxy
// @Accept       json
// @Produce      json
// @Param        id path int true "Proxy ID"
// @Param        request body updateProxyReq true "Updated proxy configuration"
// @Success      200 {object} models.Proxy
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/proxies/{id} [put]
func (h *Handler) UpdateProxy(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid id"})
		return
	}

	var req updateProxyReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request body"})
		return
	}

	proxy := req.ToModel()
	proxy.ID = uint(id)
	if err := h.service.Update(ctx, proxy); err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}

	updated, err := h.service.GetByID(ctx, uint(id))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, updated)
}

// DeleteProxy godoc
// @Summary      Delete a proxy
// @Description  Delete a proxy (soft delete - marks as disabled)
// @Tags         Proxy
// @Produce      json
// @Param        id path int true "Proxy ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      409 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/proxies/{id} [delete]
func (h *Handler) DeleteProxy(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid id"})
		return
	}

	if err := h.service.Delete(ctx, uint(id)); err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "proxy deleted successfully"})
}

// GetStats godoc
// @Summary      Get proxy statistics
// @Description  Get statistics about proxies (total, active, assignments, etc.)
// @Tags         Proxy
// @Produce      json
// @Success      200 {object} proxyStatsResp
// @Failure      500 {object} errorResp
// @Router       /v1/proxies/stats [get]
func (h *Handler) GetStats(ctx *gin.Context) {
	stats, err := h.service.GetStats(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, proxyStatsResp{
		Total:       stats.Total,
		Enabled:     stats.Enabled,
		Active:      stats.Active,
		Assignments: stats.Assignments,
	})
}

// ToggleProxy godoc
// @Summary      Toggle proxy status
// @Description  Enable or disable a proxy
// @Tags         Proxy
// @Accept       json
// @Produce      json
// @Param        id path int true "Proxy ID"
// @Param        request body toggleProxyReq true "Enable/disable flag"
// @Success      200 {object} models.Proxy
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/proxies/{id}/toggle [patch]
func (h *Handler) ToggleProxy(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid id"})
		return
	}

	var req toggleProxyReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request body"})
		return
	}

	if err := h.service.ToggleStatus(ctx, uint(id), req.Enable); err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}

	proxy, err := h.service.GetByID(ctx, uint(id))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, proxy)
}

// ForceCheck godoc
// @Summary      Force health check
// @Description  Trigger an immediate health check for a proxy
// @Tags         Proxy
// @Produce      json
// @Param        id path int true "Proxy ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/proxies/{id}/force-check [post]
func (h *Handler) ForceCheck(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid id"})
		return
	}

	if err := h.service.ForceCheck(ctx, uint(id)); err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "health check scheduled"})
}

// httpStatusFromError maps service-level errors to appropriate HTTP status codes.
func httpStatusFromError(err error) int {
	msg := err.Error()

	switch {
	case strings.Contains(msg, "not found"):
		return http.StatusNotFound
	case strings.Contains(msg, "cannot delete"):
		return http.StatusConflict
	case strings.Contains(msg, "invalid") || strings.Contains(msg, "is required"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
