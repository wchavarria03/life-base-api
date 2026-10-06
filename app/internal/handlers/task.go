package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
)

type TaskManager interface {
	List(ctx context.Context, category *models.TaskCategory) ([]models.TaskWithStatus, error)
	Create(ctx context.Context, input models.TaskInput) (*models.Task, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Task, error)
	Delete(ctx context.Context, id string) error
	Complete(ctx context.Context, id string) (*models.Task, error)
}

type TaskHandler struct {
	svc TaskManager
}

func NewTaskHandler(svc TaskManager) *TaskHandler {
	return &TaskHandler{svc: svc}
}

func (h *TaskHandler) List(c *gin.Context) {
	var category *models.TaskCategory
	if v := c.Query("category"); v != "" {
		cat := models.TaskCategory(v)
		category = &cat
	}
	tasks, err := h.svc.List(c.Request.Context(), category)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, tasks)
}

func (h *TaskHandler) Create(c *gin.Context) {
	createHandler(h.svc.Create)(c)
}

func (h *TaskHandler) Update(c *gin.Context) {
	updateHandler(h.svc.Update)(c)
}

func (h *TaskHandler) Complete(c *gin.Context) {
	task, err := h.svc.Complete(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) Delete(c *gin.Context) {
	deleteHandler(h.svc.Delete)(c)
}
