package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/auth"
)

// NewMenuPreferencesHandler constructs a MenuPreferencesHandler.
func NewMenuPreferencesHandler(svc MenuPreferenceManager) *MenuPreferencesHandler {
	return &MenuPreferencesHandler{svc: svc}
}

// Get handles GET /v1/menu-preferences.
func (h *MenuPreferencesHandler) Get(c *gin.Context) {
	userID := auth.UserIDFromContext(c.Request.Context())
	hidden, err := h.svc.Get(c.Request.Context(), userID)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"hidden_page_keys": hidden})
}

type setMenuPreferenceRequest struct {
	PageKey string `json:"page_key" binding:"required"`
	Hidden  bool   `json:"hidden"`
}

// Set handles PUT /v1/menu-preferences.
func (h *MenuPreferencesHandler) Set(c *gin.Context) {
	var req setMenuPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := auth.UserIDFromContext(c.Request.Context())
	hidden, err := h.svc.Set(c.Request.Context(), userID, req.PageKey, req.Hidden)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"hidden_page_keys": hidden})
}
