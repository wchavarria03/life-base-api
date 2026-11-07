package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
)

type TaskManager interface {
	List(ctx context.Context, category *models.TaskCategory) ([]models.TaskWithStatus, error)
	ListByNote(ctx context.Context, noteID string) ([]models.TaskWithStatus, error)
	Create(ctx context.Context, input models.TaskInput) (*models.Task, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Task, error)
	Delete(ctx context.Context, id string) error
}

// TaskCompleter is the single entry point for completing a task — plain
// TaskService.Complete for most tasks, but services.WalletService.CompleteTask
// additionally authorizes and awards coins when the task is assigned to a
// child. See WalletService.CompleteTask's doc comment.
type TaskCompleter interface {
	CompleteTask(ctx context.Context, id string, opts models.TaskCompletion) (*models.Task, error)
}

type TaskHandler struct {
	svc       TaskManager
	completer TaskCompleter
}

// NewTaskHandler constructs a TaskHandler.
func NewTaskHandler(svc TaskManager, completer TaskCompleter) *TaskHandler {
	return &TaskHandler{svc: svc, completer: completer}
}

func (h *TaskHandler) List(c *gin.Context) {
	if noteID := c.Query("note_id"); noteID != "" {
		linked, err := h.svc.ListByNote(c.Request.Context(), noteID)
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, linked)
		return
	}
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

// Complete handles POST /v1/tasks/:id/complete. The JSON body is optional
// ({performed_at, note, cost, currency}); no body behaves as before.
func (h *TaskHandler) Complete(c *gin.Context) {
	var opts models.TaskCompletion
	if err := c.ShouldBindJSON(&opts); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	task, err := h.completer.CompleteTask(c.Request.Context(), c.Param("id"), opts)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) Delete(c *gin.Context) {
	deleteHandler(h.svc.Delete)(c)
}
