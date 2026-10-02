package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
)

// TaskListManager is the service surface TaskListHandler depends on.
type TaskListManager interface {
	List(ctx context.Context, category string) ([]*models.TaskList, error)
	Create(ctx context.Context, input models.TaskListInput) (*models.TaskList, error)
	Delete(ctx context.Context, id string) error
}

// TaskListHandler serves /v1/task-lists.
type TaskListHandler struct {
	svc TaskListManager
}

// NewTaskListHandler constructs a TaskListHandler.
func NewTaskListHandler(svc TaskListManager) *TaskListHandler {
	return &TaskListHandler{svc: svc}
}

// List handles GET /v1/task-lists.
func (h *TaskListHandler) List(c *gin.Context) {
	lists, err := h.svc.List(c.Request.Context(), c.Query("category"))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, lists)
}

// Create handles POST /v1/task-lists.
func (h *TaskListHandler) Create(c *gin.Context) {
	input, ok := bindJSON[models.TaskListInput](c)
	if !ok {
		return
	}
	list, err := h.svc.Create(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, list)
}

// Delete handles DELETE /v1/task-lists/:id.
func (h *TaskListHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
