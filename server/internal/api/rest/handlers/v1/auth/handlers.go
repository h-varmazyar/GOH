package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/h-varmazyar/gatec/internal/services/auth"
	"github.com/h-varmazyar/gatec/pkg/logger"
)

// LoginByPassword godoc
// @Summary      Login by password
// @Description  Authenticate user with mobile and password
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body LoginByPasswordReq true "Login credentials"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Router       /v1/auth/login [post]
func (h Handler) LoginByPassword(ctx *gin.Context) {
	var req LoginByPasswordReq

	if err := ctx.BindJSON(&req); err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("failed to bind login by password request")
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	dto := &auth.LoginByPasswordReq{
		Mobile:   req.Mobile,
		Password: req.Password,
	}
	resp, err := h.authService.LoginByPassword(ctx, dto)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("failed to login by password")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// Register godoc
// @Summary      Register new user
// @Description  Create a new user account
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body RegisterReq true "Registration info"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Router       /v1/auth/register [post]
func (h Handler) Register(ctx *gin.Context) {
	var req RegisterReq

	if err := ctx.BindJSON(&req); err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("failed to bind register request")
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	dto := &auth.RegisterReq{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Username:  req.Username,
		Mobile:    req.Mobile,
		Password:  req.Password,
	}
	resp, err := h.authService.Register(ctx, dto)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("failed to register user")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// Logout godoc
// @Summary      Logout user
// @Description  Invalidate user session
// @Tags         Auth
// @Produce      json
// @Security     BearerTokenAuth
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Router       /v1/auth/logout [delete]
func (h Handler) Logout(ctx *gin.Context) {
	if err := h.authService.Logout(ctx, ctx.GetHeader("Authorization")); err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("failed to logout user")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}
