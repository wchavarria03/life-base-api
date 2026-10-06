package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
)

type NoteManager interface {
	List(ctx context.Context) ([]*models.Note, error)
	FindByID(ctx context.Context, id string) (*models.Note, error)
	Create(ctx context.Context, input models.NoteInput) (*models.Note, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Note, error)
	Delete(ctx context.Context, id string) error
	ListVersions(ctx context.Context, noteID string) ([]*models.NoteVersion, error)
}

type NoteHandler struct {
	svc NoteManager
}

func NewNoteHandler(svc NoteManager) *NoteHandler {
	return &NoteHandler{svc: svc}
}

func (h *NoteHandler) List(c *gin.Context) {
	listHandler(h.svc.List)(c)
}

func (h *NoteHandler) Get(c *gin.Context) {
	note, err := h.svc.FindByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		internalError(c, err)
		return
	}
	if note == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}
	c.JSON(http.StatusOK, note)
}

func (h *NoteHandler) Create(c *gin.Context) {
	createHandler(h.svc.Create)(c)
}

func (h *NoteHandler) Update(c *gin.Context) {
	updateHandler(h.svc.Update)(c)
}

func (h *NoteHandler) Delete(c *gin.Context) {
	deleteHandler(h.svc.Delete)(c)
}

// ListVersions handles GET /v1/notes/:id/versions.
func (h *NoteHandler) ListVersions(c *gin.Context) {
	versions, err := h.svc.ListVersions(c.Request.Context(), c.Param("id"))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, versions)
}
