package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/auth"
)

func NewPreferencesHandler(svc PreferenceManager) *PreferencesHandler {
	return &PreferencesHandler{svc: svc}
}

func (h *PreferencesHandler) Get(c *gin.Context) {
	userID := auth.UserIDFromContext(c.Request.Context())
	prefs, err := h.svc.Get(c.Request.Context(), userID)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, prefs)
}

type setPreferencesRequest struct {
	PushEnabled        bool `json:"push_enabled"`
	EmailDigestEnabled bool `json:"email_digest_enabled"`
}

func (h *PreferencesHandler) Set(c *gin.Context) {
	var req setPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := auth.UserIDFromContext(c.Request.Context())
	prefs, err := h.svc.Set(c.Request.Context(), userID, req.PushEnabled, req.EmailDigestEnabled)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, prefs)
}
