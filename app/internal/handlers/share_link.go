package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
	"life-base-api/app/internal/services"
)

// ShareLinkManager is the subset of ShareLinkService the handler needs —
// owner-facing management plus the public (unauthenticated) resolver.
type ShareLinkManager interface {
	List(ctx context.Context) ([]*models.ShareLink, error)
	Create(ctx context.Context, resourceType models.ShareResourceType, resourceID string, expiresAt *time.Time) (*models.ShareLink, error)
	Revoke(ctx context.Context, id string) error
	Resolve(ctx context.Context, token string) (*models.SharedResource, error)
}

// ShareLinkHandler serves both the authenticated /v1/share-links management
// endpoints and the public /public/shared/:token resolver.
type ShareLinkHandler struct {
	svc ShareLinkManager
}

// NewShareLinkHandler constructs a ShareLinkHandler.
func NewShareLinkHandler(svc ShareLinkManager) *ShareLinkHandler {
	return &ShareLinkHandler{svc: svc}
}

// List handles GET /v1/share-links.
func (h *ShareLinkHandler) List(c *gin.Context) {
	links, err := h.svc.List(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, links)
}

// Create handles POST /v1/share-links.
func (h *ShareLinkHandler) Create(c *gin.Context) {
	var req struct {
		ResourceType string  `json:"resource_type" binding:"required"`
		ResourceID   string  `json:"resource_id" binding:"required"`
		ExpiresAt    *string `json:"expires_at,omitempty"` // RFC3339, optional
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "expires_at must be RFC3339"})
			return
		}
		expiresAt = &t
	}

	link, err := h.svc.Create(c.Request.Context(), models.ShareResourceType(req.ResourceType), req.ResourceID, expiresAt)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, link)
}

// Revoke handles DELETE /v1/share-links/:id.
func (h *ShareLinkHandler) Revoke(c *gin.Context) {
	if err := h.svc.Revoke(c.Request.Context(), c.Param("id")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// PublicResolve handles GET /public/shared/:token — no auth, the token is
// the credential.
func (h *ShareLinkHandler) PublicResolve(c *gin.Context) {
	resource, err := h.svc.Resolve(c.Request.Context(), c.Param("token"))
	if err != nil {
		if errors.Is(err, services.ErrShareLinkNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "share link not found"})
			return
		}
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, resource)
}
