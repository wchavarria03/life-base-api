package handlers

import (
	"context"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
)

// NewAdminHandler constructs an AdminHandler.
func NewAdminHandler(svc AdminManager) *AdminHandler {
	return &AdminHandler{svc: svc}
}

// ListMembers handles GET /v1/admin/users.
func (h *AdminHandler) ListMembers(c *gin.Context) {
	listHandler(h.svc.ListMembers)(c)
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
	listHandler(h.svc.ListPageAccess)(c)
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

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Gate exposes the feature check used by middleware.FeatureGate.
func (h *AdminHandler) Gate() interface {
	PageAllowedAny(ctx context.Context, userID, role string, keys []string) (bool, error)
} {
	return h.svc
}

// ListUserPageAccess handles GET /v1/admin/users/:id/page-access, returning
// that account's per-page overrides of the role matrix.
func (h *AdminHandler) ListUserPageAccess(c *gin.Context) {
	userID := c.Param("id")
	if !uuidPattern.MatchString(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	entries, err := h.svc.ListUserPageAccess(c.Request.Context(), userID)
	if err != nil {
		internalError(c, err)
		return
	}
	if entries == nil {
		entries = []*models.UserPageAccessEntry{}
	}
	c.JSON(http.StatusOK, entries)
}

type setUserPageAccessRequest struct {
	PageKey string `json:"page_key" binding:"required"`
	// Allowed true/false forces the page on/off for the account; null
	// removes the override so the role default applies again.
	Allowed *bool `json:"allowed"`
}

// SetUserPageAccess handles PUT /v1/admin/users/:id/page-access.
func (h *AdminHandler) SetUserPageAccess(c *gin.Context) {
	userID := c.Param("id")
	if !uuidPattern.MatchString(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	var req setUserPageAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.SetUserPageAccess(c.Request.Context(), userID, req.PageKey, req.Allowed); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
