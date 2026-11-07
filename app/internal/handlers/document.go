package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/services"
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

	doc, err := h.svc.Upload(c.Request.Context(), file, title, header.Filename, contentType, category, c.PostForm("sensitive") == "true")
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

// linkRequest is the body of POST /v1/documents/:id/links.
type linkRequest struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
}

// documentLinkStatus maps a link/sign error to an HTTP status.
func documentLinkStatus(err error) int {
	switch {
	case errors.Is(err, services.ErrDocumentNotFound):
		return http.StatusNotFound
	case errors.Is(err, services.ErrInvalidEntityType), errors.Is(err, services.ErrInvalidEntityID):
		return http.StatusBadRequest
	case errors.Is(err, services.ErrSigningDisabled):
		return http.StatusServiceUnavailable
	default:
		return http.StatusUnprocessableEntity
	}
}

// AddLink handles POST /v1/documents/:id/links.
func (h *DocumentHandler) AddLink(c *gin.Context) {
	var req linkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "entity_type and entity_id are required"})
		return
	}
	link, err := h.svc.AddLink(c.Request.Context(), c.Param("id"), req.EntityType, req.EntityID)
	if err != nil {
		c.JSON(documentLinkStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, link)
}

// ListLinks handles GET /v1/documents/:id/links.
func (h *DocumentHandler) ListLinks(c *gin.Context) {
	links, err := h.svc.ListLinks(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list links"})
		return
	}
	c.JSON(http.StatusOK, links)
}

// RemoveLink handles DELETE /v1/documents/:id/links/:linkId.
func (h *DocumentHandler) RemoveLink(c *gin.Context) {
	if err := h.svc.RemoveLink(c.Request.Context(), c.Param("id"), c.Param("linkId")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not remove link"})
		return
	}
	c.Status(http.StatusNoContent)
}

// ByEntity handles GET /v1/documents/by-entity?type=&id=.
func (h *DocumentHandler) ByEntity(c *gin.Context) {
	docs, err := h.svc.ListByEntity(c.Request.Context(), c.Query("type"), c.Query("id"))
	if err != nil {
		c.JSON(documentLinkStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, docs)
}

// SignedURL handles GET /v1/documents/:id/signed-url.
func (h *DocumentHandler) SignedURL(c *gin.Context) {
	signed, err := h.svc.SignedURL(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(documentLinkStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, signed)
}

// PublicDownload handles GET /public/documents/:id?exp=&sig= — no JWT; the
// HMAC signature is the credential. Every failure looks the same.
func (h *DocumentHandler) PublicDownload(c *gin.Context) {
	doc, data, err := h.svc.DownloadSigned(c.Request.Context(), c.Param("id"), c.Query("exp"), c.Query("sig"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.Header("Cache-Control", "private, max-age=60")
	c.Header("Content-Disposition", fmt.Sprintf(`%s; filename=%q`, publicDisposition(doc.ContentType), doc.FileName))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, doc.ContentType, data)
}

// publicDisposition serves only raster images and PDFs inline; anything
// else (HTML, SVG, ...) is forced to download so an uploaded file can never
// run script on the API origin.
func publicDisposition(contentType string) string {
	switch contentType {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "application/pdf":
		return "inline"
	}
	return "attachment"
}
