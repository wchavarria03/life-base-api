package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/auth"
)

func NewMeHandler(admin AdminManager) *MeHandler {
	return &MeHandler{admin: admin}
}

func (h *MeHandler) GetMe(c *gin.Context) {
	ctx := c.Request.Context()
	userID := auth.UserIDFromContext(ctx)
	role := auth.RoleFromContext(ctx)

	pageKeys, err := h.admin.AllowedPageKeys(ctx, role)
	if err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":   userID,
		"role":      role,
		"page_keys": pageKeys,
	})
}
