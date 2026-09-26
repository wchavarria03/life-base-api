package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
)

// NewAdminHandler constructs an AdminHandler.
func NewAdminHandler(svc AdminManager) *AdminHandler {
	return &AdminHandler{svc: svc}
}

// ListMembers handles GET /v1/admin/users.
func (h *AdminHandler) ListMembers(c *gin.Context) {
	members, err := h.svc.ListMembers(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, members)
}

type setRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin member"`
}

// SetMemberRole handles PATCH /v1/admin/users/:id/role.
func (h *AdminHandler) SetMemberRole(c *gin.Context) {
	var req setRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.SetMemberRole(c.Request.Context(), c.Param("id"), req.Role); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListPageAccess handles GET /v1/admin/page-access.
func (h *AdminHandler) ListPageAccess(c *gin.Context) {
	entries, err := h.svc.ListPageAccess(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, entries)
}

// SetPageAccess handles PUT /v1/admin/page-access.
func (h *AdminHandler) SetPageAccess(c *gin.Context) {
	var entry models.PageAccessEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.SetPageAccess(c.Request.Context(), &entry); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
