package configuration

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/h-varmazyar/gatec/internal/services/dynamicconfig"
)

type Handler struct{ service *dynamicconfig.Service }

func New(service *dynamicconfig.Service) Handler { return Handler{service: service} }
func (h Handler) HasService() bool               { return h.service != nil }

func (h Handler) ListSections(ctx *gin.Context) {
	sections := ctx.QueryArray("items")
	result, err := h.service.ListSections(ctx, sections)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"sections": result})
}

// func (h Handler) UpdateSection(ctx *gin.Context) {

// 	var req UpdateSectionReq
// 	if err := ctx.ShouldBindJSON(&req); err != nil {
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
// 		return
// 	}
// 	if err := h.service.UpdateSection(ctx, ctx.Param("section"), req.Values); err != nil {
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	ctx.JSON(http.StatusOK, gin.H{"message": "configuration updated successfully"})
// }

// func (h Handler) ListWorkers(ctx *gin.Context) {
// 	result, err := h.service.ListWorkers()
// 	if err != nil {
// 		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}
// 	ctx.JSON(http.StatusOK, gin.H{"workers": result})
// }

// func (h Handler) SetWorkerEnabled(ctx *gin.Context) {
// 	var req SetWorkerEnabledReq
// 	if err := ctx.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "enabled is required"})
// 		return
// 	}
// 	if err := h.service.SetWorkerEnabled(ctx, ctx.Param("name"), *req.Enabled); err != nil {
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	ctx.JSON(http.StatusOK, gin.H{"message": "worker state updated successfully"})
// }
