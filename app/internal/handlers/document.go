package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

const maxDocumentBytes = 25 << 20 // 25 MB

// NewDocumentHandler constructs a DocumentHandler.
func NewDocumentHandler(svc DocumentManager) *DocumentHandler {
	return &DocumentHandler{svc: svc}
}

// List handles GET /v1/documents.
func (h *DocumentHandler) List(c *gin.Context) {
	listHandler(h.svc.List)(c)
}

// Upload handles POST /v1/documents — multipart upload with a "file" field
// (required), "title" (required), and optional "category".
func (h *DocumentHandler) Upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxDocumentBytes)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file field is required"})
		return
	}
	defer func() { _ = file.Close() }()

	title := c.PostForm("title")
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	var category *string
	if v := c.PostForm("category"); v != "" {
		category = &v
	}

	doc, err := h.svc.Upload(c.Request.Context(), file, title, header.Filename, contentType, category)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, doc)
}

// Download handles GET /v1/documents/:id/download.
func (h *DocumentHandler) Download(c *gin.Context) {
	doc, data, err := h.svc.Download(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, doc.FileName))
	c.Data(http.StatusOK, doc.ContentType, data)
}

// Delete handles DELETE /v1/documents/:id.
func (h *DocumentHandler) Delete(c *gin.Context) {
	deleteHandler(h.svc.Delete)(c)
}
