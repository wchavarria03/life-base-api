package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
	"life-base-api/app/internal/services"
)

// NoteTaskLinker turns a note's checklist item into a linked task.
type NoteTaskLinker interface {
	CreateTaskFromChecklistItem(ctx context.Context, noteID string, lineIndex int, in models.ChecklistTaskInput) (*models.ChecklistTaskResult, error)
}

// NoteTaskLinkHandler serves the Notes <-> TODO linking endpoints.
type NoteTaskLinkHandler struct {
	svc NoteTaskLinker
}

// NewNoteTaskLinkHandler constructs a NoteTaskLinkHandler.
func NewNoteTaskLinkHandler(svc NoteTaskLinker) *NoteTaskLinkHandler {
	return &NoteTaskLinkHandler{svc: svc}
}

// CreateTask handles POST /v1/notes/:id/checklist/:index/task. The body
// (due_date, priority) is optional; :index is the zero-based line index of
// the checklist item in the note content.
func (h *NoteTaskLinkHandler) CreateTask(c *gin.Context) {
	index, err := strconv.Atoi(c.Param("index"))
	if err != nil || index < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "index must be a non-negative integer"})
		return
	}
	var in models.ChecklistTaskInput
	if err := c.ShouldBindJSON(&in); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.svc.CreateTaskFromChecklistItem(c.Request.Context(), c.Param("id"), index, in)
	if err != nil {
		if errors.Is(err, services.ErrNoteNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, result)
}
