package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/auth"
)

func NewPushHandler(svc PushManager) *PushHandler {
	return &PushHandler{svc: svc}
}

type subscribeRequest struct {
	Endpoint string `json:"endpoint" binding:"required"`
	P256dh   string `json:"p256dh" binding:"required"`
	Auth     string `json:"auth" binding:"required"`
}

// Subscribe handles POST /v1/push-subscriptions — registers a browser
// endpoint for this user, upserting on re-subscription.
func (h *PushHandler) Subscribe(c *gin.Context) {
	var req subscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := auth.UserIDFromContext(c.Request.Context())
	if err := h.svc.Subscribe(c.Request.Context(), userID, req.Endpoint, req.P256dh, req.Auth); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type unsubscribeRequest struct {
	Endpoint string `json:"endpoint" binding:"required"`
}

// Unsubscribe handles DELETE /v1/push-subscriptions.
func (h *PushHandler) Unsubscribe(c *gin.Context) {
	var req unsubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.Unsubscribe(c.Request.Context(), req.Endpoint); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
