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
}

type NoteHandler struct {
	svc NoteManager
}

func NewNoteHandler(svc NoteManager) *NoteHandler {
	return &NoteHandler{svc: svc}
}

func (h *NoteHandler) List(c *gin.Context) {
	notes, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, notes)
}

func (h *NoteHandler) Get(c *gin.Context) {
	note, err := h.svc.FindByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if note == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}
	c.JSON(http.StatusOK, note)
}

func (h *NoteHandler) Create(c *gin.Context) {
	var input models.NoteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	note, err := h.svc.Create(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, note)
}

func (h *NoteHandler) Update(c *gin.Context) {
	var fields map[string]any
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	note, err := h.svc.Update(c.Request.Context(), c.Param("id"), fields)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, note)
}

func (h *NoteHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
