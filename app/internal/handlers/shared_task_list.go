package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
)

// SharedTaskListManager is the subset of SharedTaskListService the handler
// needs — owner-facing management plus the public (unauthenticated) views.
type SharedTaskListManager interface {
	List(ctx context.Context) ([]*models.SharedTaskList, error)
	Create(ctx context.Context, category models.TaskCategory, email string) (*models.SharedTaskList, error)
	Revoke(ctx context.Context, id string) error
	PublicListTasks(ctx context.Context, token string) ([]models.TaskWithStatus, error)
	PublicCompleteTask(ctx context.Context, token, taskID string) (*models.Task, error)
}

// SharedTaskListHandler serves both the authenticated /v1/shared-lists
// management endpoints and the public /public/shared/:token endpoints.
type SharedTaskListHandler struct {
	svc SharedTaskListManager
}

// NewSharedTaskListHandler constructs a SharedTaskListHandler.
func NewSharedTaskListHandler(svc SharedTaskListManager) *SharedTaskListHandler {
	return &SharedTaskListHandler{svc: svc}
}

type createSharedTaskListRequest struct {
	Category string `json:"category"`
	Email    string `json:"email"`
}

// List handles GET /v1/shared-lists.
func (h *SharedTaskListHandler) List(c *gin.Context) {
	shares, err := h.svc.List(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, shares)
}

// Create handles POST /v1/shared-lists.
func (h *SharedTaskListHandler) Create(c *gin.Context) {
	var req createSharedTaskListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	share, err := h.svc.Create(c.Request.Context(), models.TaskCategory(req.Category), req.Email)
	if err != nil {
		if share != nil {
			// Link created but the invite email failed to send — still a
			// partial success, so return 201 with the share (and its token)
			// rather than swallowing it as a hard failure.
			c.JSON(http.StatusCreated, gin.H{"share": share, "warning": err.Error()})
			return
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, share)
}

// Revoke handles DELETE /v1/shared-lists/:id.
func (h *SharedTaskListHandler) Revoke(c *gin.Context) {
	if err := h.svc.Revoke(c.Request.Context(), c.Param("id")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// PublicTasks handles GET /public/shared/:token — no auth, the token is the
// credential.
func (h *SharedTaskListHandler) PublicTasks(c *gin.Context) {
	tasks, err := h.svc.PublicListTasks(c.Request.Context(), c.Param("token"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
		return
	}
	c.JSON(http.StatusOK, tasks)
}

// PublicComplete handles POST /public/shared/:token/tasks/:id/complete — no
// auth, the token is the credential.
func (h *SharedTaskListHandler) PublicComplete(c *gin.Context) {
	task, err := h.svc.PublicCompleteTask(c.Request.Context(), c.Param("token"), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	c.JSON(http.StatusOK, task)
}
