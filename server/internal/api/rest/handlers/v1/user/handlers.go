package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/h-varmazyar/gatec/pkg/logger"
)

// UserInfo godoc
// @Summary      Get user info
// @Description  Get current authenticated user information
// @Tags         User
// @Produce      json
// @Security     BearerTokenAuth
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Router       /v1/user/info [get]
func (h Handler) UserInfo(ctx *gin.Context) {
	logger.Logger.Infof("user info handler called: %d", ctx.GetUint("user_id"))
	resp, err := h.userService.GetUserInfo(ctx, ctx.GetUint("user_id"))
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("failed to get user info")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, resp)
}
