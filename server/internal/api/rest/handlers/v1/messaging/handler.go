package messaging

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/h-varmazyar/gatec/internal/models"
	messagingService "github.com/h-varmazyar/gatec/internal/services/messaging"
)

// Handler handles messaging REST API endpoints.
type Handler struct {
	service *messagingService.Service
}

// New creates a new messaging handler.
func New(service *messagingService.Service) Handler {
	return Handler{service: service}
}

// HasService returns true if the handler has a valid service.
func (h *Handler) HasService() bool {
	return h.service != nil
}

// --- Message Sending ---

// SendMessage godoc
// @Summary      Send a message
// @Description  Send a message through a registered sender
// @Tags         Messaging
// @Accept       json
// @Produce      json
// @Param        request body sendMessageReq true "Message to send"
// @Success      200 {object} models.SendResult
// @Failure      400 {object} errorResp
// @Failure      403 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/messaging/send [post]
func (h *Handler) SendMessage(ctx *gin.Context) {
	var req sendMessageReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request: " + err.Error()})
		return
	}

	message := &models.Message{
		ID:          uuid.New(),
		SenderID:    req.SenderID,
		ClientID:    req.ClientID,
		ChatID:      req.ChatID,
		Content:     req.Content,
		ContentType: models.MessageType(req.ContentType),
	}

	if req.ContentType == "" {
		message.ContentType = models.MessageTypeText
	}

	result, err := h.service.SendMessage(ctx.Request.Context(), req.SenderID, message)
	if err != nil {
		if result != nil && result.Error == "access denied" {
			ctx.JSON(http.StatusForbidden, errorResp{Error: err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, result)
}

// ScheduleMessage godoc
// @Summary      Schedule a message
// @Description  Schedule a message for future delivery
// @Tags         Messaging
// @Accept       json
// @Produce      json
// @Param        request body scheduleMessageReq true "Scheduled message"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} errorResp
// @Failure      403 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/messaging/schedule [post]
func (h *Handler) ScheduleMessage(ctx *gin.Context) {
	var req scheduleMessageReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request: " + err.Error()})
		return
	}

	scheduledAt := time.Unix(req.ScheduledAtUnix, 0)
	message := &models.Message{
		ID:          uuid.New(),
		SenderID:    req.SenderID,
		ClientID:    req.ClientID,
		ChatID:      req.ChatID,
		Content:     req.Content,
		ContentType: models.MessageType(req.ContentType),
		ScheduledAt: &scheduledAt,
		Status:      models.MessageStatusScheduled,
	}

	if req.ContentType == "" {
		message.ContentType = models.MessageTypeText
	}

	if err := h.service.ScheduleMessage(ctx.Request.Context(), req.SenderID, message); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message_id":   message.ID,
		"scheduled_at": scheduledAt,
		"status":       message.Status,
	})
}

// CancelScheduledMessage godoc
// @Summary      Cancel a scheduled message
// @Description  Cancel a message that was scheduled for future delivery
// @Tags         Messaging
// @Produce      json
// @Param        id path string true "Message ID (UUID)"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/messaging/schedule/{id} [delete]
func (h *Handler) CancelScheduledMessage(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid message id"})
		return
	}

	if err := h.service.CancelScheduledMessage(ctx.Request.Context(), id); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "scheduled message cancelled"})
}

// --- Access Control ---

// ListAccessRules godoc
// @Summary      List access rules
// @Description  Get all active access control rules
// @Tags         Messaging
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /v1/messaging/access/rules [get]
func (h *Handler) ListAccessRules(ctx *gin.Context) {
	rules := h.service.ListAccessRules()
	ctx.JSON(http.StatusOK, gin.H{"data": rules})
}

// GrantAccess godoc
// @Summary      Grant access
// @Description  Grant a sender access to use a specific Telegram client
// @Tags         Messaging
// @Accept       json
// @Produce      json
// @Param        request body grantAccessReq true "Access grant request"
// @Success      201 {object} models.AccessRule
// @Failure      400 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/messaging/access/rules [post]
func (h *Handler) GrantAccess(ctx *gin.Context) {
	var req grantAccessReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid request: " + err.Error()})
		return
	}

	permission := models.Permission(req.Permission)

	var expiresAt *time.Time
	if req.ExpiresAtUnix > 0 {
		t := time.Unix(req.ExpiresAtUnix, 0)
		expiresAt = &t
	}

	rule, err := h.service.GrantAccess(
		ctx.Request.Context(),
		req.SenderID,
		req.ClientID,
		req.GrantedBy,
		req.Reason,
		permission,
		expiresAt,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, rule)
}

// RevokeAccess godoc
// @Summary      Revoke access
// @Description  Revoke an access rule by its ID
// @Tags         Messaging
// @Produce      json
// @Param        id path string true "Rule ID (UUID)"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} errorResp
// @Failure      404 {object} errorResp
// @Failure      500 {object} errorResp
// @Router       /v1/messaging/access/rules/{id} [delete]
func (h *Handler) RevokeAccess(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "invalid rule id"})
		return
	}

	revoked, err := h.service.RevokeAccess(ctx.Request.Context(), id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
		return
	}

	if !revoked {
		ctx.JSON(http.StatusNotFound, errorResp{Error: "rule not found"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "access revoked"})
}

// CheckAccess godoc
// @Summary      Check access
// @Description  Check if a sender has permission to use a specific client
// @Tags         Messaging
// @Produce      json
// @Param        sender_id query string true "Sender ID"
// @Param        client_id query string true "Client ID"
// @Param        permission query string false "Permission type" default(send)
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} errorResp
// @Router       /v1/messaging/access/check [get]
func (h *Handler) CheckAccess(ctx *gin.Context) {
	senderID := ctx.Query("sender_id")
	clientID := ctx.Query("client_id")
	permStr := ctx.DefaultQuery("permission", "send")

	if senderID == "" || clientID == "" {
		ctx.JSON(http.StatusBadRequest, errorResp{Error: "sender_id and client_id are required"})
		return
	}

	permission := models.Permission(permStr)
	allowed := h.service.CheckAccess(senderID, clientID, permission)

	ctx.JSON(http.StatusOK, gin.H{
		"sender_id":  senderID,
		"client_id":  clientID,
		"permission": permission,
		"allowed":    allowed,
	})
}

// --- Senders ---

// ListSenders godoc
// @Summary      List registered senders
// @Description  Get all registered message senders
// @Tags         Messaging
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /v1/messaging/senders [get]
func (h *Handler) ListSenders(ctx *gin.Context) {
	senders := h.service.ListSenders()
	ctx.JSON(http.StatusOK, gin.H{"data": senders})
}
