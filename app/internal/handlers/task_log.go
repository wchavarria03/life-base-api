package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
)

// TaskLogManager is the service surface the task-log handlers need.
type TaskLogManager interface {
	Create(ctx context.Context, input models.TaskLogInput) (*models.TaskLog, error)
	List(ctx context.Context, filter models.TaskLogFilter) ([]*models.TaskLog, error)
	Delete(ctx context.Context, id string) error
	Summary(ctx context.Context, year int) (*models.TaskLogSummary, error)
}

// TaskLogHandler serves the home-maintenance history endpoints.
type TaskLogHandler struct {
	svc TaskLogManager
}

// NewTaskLogHandler constructs a TaskLogHandler.
func NewTaskLogHandler(svc TaskLogManager) *TaskLogHandler {
	return &TaskLogHandler{svc: svc}
}

// Create handles POST /v1/tasks/logs.
func (h *TaskLogHandler) Create(c *gin.Context) {
	createHandler(h.svc.Create)(c)
}

// List handles GET /v1/tasks/logs?from=&to=&area=&category=.
func (h *TaskLogHandler) List(c *gin.Context) {
	logs, err := h.svc.List(c.Request.Context(), models.TaskLogFilter{
		From:     c.Query("from"),
		To:       c.Query("to"),
		Area:     c.Query("area"),
		Category: c.Query("category"),
	})
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}

// Summary handles GET /v1/tasks/logs/summary?year= (defaults to this year).
func (h *TaskLogHandler) Summary(c *gin.Context) {
	year := time.Now().Year()
	if v := c.Query("year"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "year must be a number"})
			return
		}
		year = parsed
	}
	summary, err := h.svc.Summary(c.Request.Context(), year)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// Delete handles DELETE /v1/tasks/logs/:id.
func (h *TaskLogHandler) Delete(c *gin.Context) {
	deleteHandler(h.svc.Delete)(c)
}
