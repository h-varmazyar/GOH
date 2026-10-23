package applog

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	applogservice "github.com/h-varmazyar/gatec/internal/services/applog"
	"github.com/h-varmazyar/gatec/pkg/paginator"
)

type Handler struct {
	service *applogservice.Service
}

func New(service *applogservice.Service) Handler { return Handler{service: service} }
func (h Handler) HasService() bool               { return h.service != nil }

// List godoc
// @Summary      List application logs by component
// @Description  Returns matching JSON log entries from newest to oldest
// @Tags         Logs
// @Produce      json
// @Param        component path string true "Component, worker, or adapter name" example(tse-market-history-sync)
// @Param        page query int false "Reverse page number" default(1)
// @Param        page_size query int false "Items per page" default(20) maximum(100)
// @Success      200 {object} listResponse
// @Failure      400 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Security     BearerTokenAuth
// @Router       /v1/logs/{component} [get]
func (h Handler) List(ctx *gin.Context) {
	pg := paginator.FromContext(ctx)
	items, total, err := h.service.List(ctx, ctx.Param("component"), pg)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, applogservice.ErrComponentRequired) {
			status = http.StatusBadRequest
		}
		ctx.JSON(status, errorResponse{Error: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, listResponse{Data: items, Pagination: pg.ToResponse(total)})
}
