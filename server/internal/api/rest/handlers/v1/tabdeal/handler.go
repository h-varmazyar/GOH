package tabdeal

import (
	// "net/http"

	// "github.com/gin-gonic/gin"
	"github.com/h-varmazyar/gatec/internal/services/dynamicconfig"
)

type Handler struct {
	service *dynamicconfig.Service
}

func New(service *dynamicconfig.Service) Handler {
	return Handler{service: service}
}

func (h Handler) HasService() bool {
	return h.service != nil
}

// UpdateAuthToken godoc
// @Summary      Update Tabdeal auth token
// @Description  Persist and apply the Tabdeal auth token at runtime
// @Tags         Tabdeal
// @Accept       json
// @Produce      json
// @Security     BearerTokenAuth
// @Param        request body UpdateAuthTokenReq true "New auth token"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Router       /v1/tabdeal/auth-token [patch]
// func (h Handler) UpdateAuthToken(ctx *gin.Context) {
// 	var req UpdateAuthTokenReq
// 	if err := ctx.ShouldBindJSON(&req); err != nil {
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
// 		return
// 	}
// 	if err := h.service.UpdateTabdealAuthToken(ctx, req.Token); err != nil {
// 		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	ctx.JSON(http.StatusOK, gin.H{"message": "tabdeal auth token updated successfully"})
// }
