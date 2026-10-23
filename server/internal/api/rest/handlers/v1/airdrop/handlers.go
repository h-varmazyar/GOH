package airdrop

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/h-varmazyar/gatec/pkg/paginator"
)

// CreateRoute godoc
// @Summary      Create airdrop route
// @Description  Create a new airdrop route with phrase and route
// @Tags         Airdrop
// @Accept       json
// @Produce      json
// @Security     BearerTokenAuth
// @Param        request body CreateRouteReq true "Route info"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Router       /v1/airdrops/routes [post]
func (h Handler) CreateRoute(ctx *gin.Context) {
	var req CreateRouteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.service.CreateRoute(ctx, req.Phrase, req.Route)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, item)
}

// ListRoutes godoc
// @Summary      List airdrop routes
// @Description  Get paginated list of airdrop routes
// @Tags         Airdrop
// @Produce      json
// @Security     BearerTokenAuth
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Items per page" default(20)
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Router       /v1/airdrops/routes [get]
func (h Handler) ListRoutes(ctx *gin.Context) {
	pg := paginator.FromContext(ctx)
	items, total, err := h.service.ListRoutes(ctx, pg)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"data":       items,
		"pagination": pg.ToResponse(total),
	})
}

// ListGifts godoc
// @Summary      List airdrop gifts
// @Description  Get paginated list of airdrop gifts
// @Tags         Airdrop
// @Produce      json
// @Security     BearerTokenAuth
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Items per page" default(20)
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Router       /v1/airdrops/gifts [get]
func (h Handler) ListGifts(ctx *gin.Context) {
	pg := paginator.FromContext(ctx)
	items, total, err := h.service.ListGifts(ctx, pg)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"data":       items,
		"pagination": pg.ToResponse(total),
	})
}

// GetGift godoc
// @Summary      Get airdrop gift by ID
// @Description  Get a specific airdrop gift by its ID
// @Tags         Airdrop
// @Produce      json
// @Security     BearerTokenAuth
// @Param        id path int true "Gift ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Router       /v1/airdrops/gifts/{id} [get]
func (h Handler) GetGift(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.service.GetGift(ctx, uint(id))
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, item)
}

// GetGift godoc
// @Summary      Get airdrop gift by ID
// @Description  Get a specific airdrop gift by its ID
// @Tags         Airdrop
// @Produce      json
// @Security     BearerTokenAuth
// @Param        id path int true "Gift ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Router       /v1/airdrops/gifts/{id} [get]
func (h Handler) RetryCollectGift(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.service.RetryCollectGift(ctx, uint(id))
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, item)
}
