package tgaccount

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/h-varmazyar/gatec/internal/services/tgaccount"
	"github.com/h-varmazyar/gatec/pkg/paginator"
)

// Handler handles telegram account REST API endpoints.
type Handler struct {
	service *tgaccount.Service
}

// New creates a new telegram account handler.
func New(service *tgaccount.Service) *Handler {
	return &Handler{service: service}
}

// HasService returns true if the handler has a valid service.
func (h *Handler) HasService() bool {
	return h.service != nil
}

// CreateAccount godoc
// @Summary      Create telegram account session
// @Description  Start a new telegram account authentication session
// @Tags         TelegramV2
// @Accept       json
// @Produce      json
// @Param        request body createAccountReq true "Session type"
// @Success      200 {object} createAccountResp
// @Failure      400 {object} errorResp
// @Failure      408 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v2/telegram/sessions [post]
func (h *Handler) CreateAccount(ctx *gin.Context) {
	var req createAccountReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request body"})
		return
	}

	resp, err := h.service.CreateAccount(ctx, req.SessionType)
	if err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, createAccountResp{
		SessionID: resp.SessionID,
		State:     resp.State,
	})
}

// SubmitPhone godoc
// @Summary      Submit phone number
// @Description  Submit phone number for an active telegram session
// @Tags         TelegramV2
// @Accept       json
// @Produce      json
// @Param        request body submitPhoneReq true "Phone number"
// @Success      200 {object} stateResp
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      408 {object} errorResp
// @Failure      409 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v2/telegram/sessions/phone [post]
func (h *Handler) SubmitPhone(ctx *gin.Context) {
	var req submitPhoneReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request body"})
		return
	}

	resp, err := h.service.SubmitPhone(ctx, req.SessionID, req.PhoneNumber)
	if err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, stateResp{
		SessionID: resp.SessionID,
		State:     resp.State,
	})
}

// SubmitBotToken godoc
// @Summary      Submit bot token
// @Description  Submit a bot token (from @BotFather) for an active bot session
// @Tags         TelegramV2
// @Accept       json
// @Produce      json
// @Param        request body submitBotTokenReq true "Bot token"
// @Success      200 {object} stateResp
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      408 {object} errorResp
// @Failure      409 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v2/telegram/sessions/bot-token [post]
func (h *Handler) SubmitBotToken(ctx *gin.Context) {
	var req submitBotTokenReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request body"})
		return
	}

	resp, err := h.service.SubmitBotToken(ctx, req.SessionID, req.BotToken)
	if err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, stateResp{
		SessionID: resp.SessionID,
		State:     resp.State,
	})
}

// SubmitCode godoc
// @Summary      Submit OTP code
// @Description  Submit verification code for an active telegram session
// @Tags         TelegramV2
// @Accept       json
// @Produce      json
// @Param        request body submitCodeReq true "OTP code"
// @Success      200 {object} stateResp
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      408 {object} errorResp
// @Failure      409 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v2/telegram/sessions/code [post]
func (h *Handler) SubmitCode(ctx *gin.Context) {
	var req submitCodeReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request body"})
		return
	}

	resp, err := h.service.SubmitCode(ctx, req.SessionID, req.Code)
	if err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, stateResp{
		SessionID: resp.SessionID,
		State:     resp.State,
	})
}

// SubmitPassword godoc
// @Summary      Submit 2FA password
// @Description  Submit two-factor authentication password for an active telegram session
// @Tags         TelegramV2
// @Accept       json
// @Produce      json
// @Param        request body submitPasswordReq true "Password"
// @Success      200 {object} stateResp
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      408 {object} errorResp
// @Failure      409 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v2/telegram/sessions/password [post]
func (h *Handler) SubmitPassword(ctx *gin.Context) {
	var req submitPasswordReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request body"})
		return
	}

	resp, err := h.service.SubmitPassword(ctx, req.SessionID, req.Password)
	if err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, stateResp{
		SessionID: resp.SessionID,
		State:     resp.State,
	})
}

// ReAuthenticate godoc
// @Summary      Re-authenticate an existing account
// @Description  Start re-authentication for an account that needs login. Automatically submits the stored phone number and returns the next state.
// @Tags         TelegramV2
// @Accept       json
// @Produce      json
// @Param        request body reAuthReq true "Account ID"
// @Success      200 {object} stateResp
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      408 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v2/telegram/sessions/re-auth [post]
func (h *Handler) ReAuthenticate(ctx *gin.Context) {
	var req reAuthReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request body"})
		return
	}

	resp, err := h.service.ReAuthenticate(ctx, req.AccountID)
	if err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, stateResp{
		SessionID: resp.SessionID,
		State:     resp.State,
	})
}

// GetState godoc
// @Summary      Get session state
// @Description  Get the current state of a telegram session
// @Tags         TelegramV2
// @Produce      json
// @Param        session_id path string true "Session ID"
// @Success      200 {object} stateResp
// @Failure      404 {object} errorResp
// @Router       /v2/telegram/sessions/{session_id}/state [get]
func (h *Handler) GetState(ctx *gin.Context) {
	sessionID := ctx.Param("session_id")
	if sessionID == "" {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "session_id is required"})
		return
	}

	state, err := h.service.GetSessionState(ctx, sessionID)
	if err != nil {
		ctx.JSON(httpStatusFromError(err), errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, stateResp{
		SessionID: sessionID,
		State:     state,
	})
}

// ListAccounts godoc
// @Summary      List telegram accounts
// @Description  Get paginated list of telegram accounts from database
// @Tags         TelegramV2
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Items per page" default(20)
// @Success      200 {object} map[string]interface{}
// @Failure      500 {object} errorResp
// @Router       /v2/telegram/accounts [get]
func (h *Handler) ListAccounts(ctx *gin.Context) {
	pg := paginator.FromContext(ctx)
	items, total, err := h.service.ListAccounts(ctx, pg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"data":       items,
		"pagination": pg.ToResponse(total),
	})
}

// CreateSubscriptionFilter godoc
// @Summary      Create subscription filter
// @Description  Create a new subscription filter for telegram updates
// @Tags         TelegramV2
// @Accept       json
// @Produce      json
// @Param        request body createFilterReq true "Filter info"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v2/telegram/subscription-filters [post]
func (h *Handler) CreateSubscriptionFilter(ctx *gin.Context) {
	var req createFilterReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request body"})
		return
	}

	filter, err := h.service.CreateSubscriptionFilter(ctx, req.AccountID, req.ChatID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, filter)
}

// ListSubscriptionFilters godoc
// @Summary      List subscription filters
// @Description  Get paginated list of telegram subscription filters
// @Tags         TelegramV2
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Items per page" default(20)
// @Success      200 {object} map[string]interface{}
// @Failure      500 {object} errorResp
// @Router       /v2/telegram/subscription-filters [get]
func (h *Handler) ListSubscriptionFilters(ctx *gin.Context) {
	pg := paginator.FromContext(ctx)
	items, total, err := h.service.ListSubscriptionFilters(ctx, pg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"data":       items,
		"pagination": pg.ToResponse(total),
	})
}

// DeleteSubscriptionFilter godoc
// @Summary      Delete subscription filter
// @Description  Delete a telegram subscription filter by ID
// @Tags         TelegramV2
// @Produce      json
// @Param        id path int true "Filter ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v2/telegram/subscription-filters/{id} [delete]
func (h *Handler) DeleteSubscriptionFilter(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid id"})
		return
	}

	if err := h.service.DeleteSubscriptionFilter(ctx, uint(id)); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// httpStatusFromError maps service-level errors to appropriate HTTP status codes.
func httpStatusFromError(err error) int {
	msg := err.Error()

	switch {
	case strings.Contains(msg, "not found"):
		return http.StatusNotFound
	case strings.Contains(msg, "timeout"):
		return http.StatusRequestTimeout
	case strings.Contains(msg, "not waiting for"),
		strings.Contains(msg, "is not waiting for"),
		strings.Contains(msg, "already exists"),
		strings.Contains(msg, "channel is full"):
		return http.StatusConflict
	case strings.Contains(msg, "invalid"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
