package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/auth"
)

func NewMeHandler() *MeHandler {
	return &MeHandler{}
}

func (h *MeHandler) GetMe(c *gin.Context) {
	userID := auth.UserIDFromContext(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"user_id": userID})
}
