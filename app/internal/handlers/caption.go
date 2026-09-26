package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewCaptionHandler(svc CaptionManager) *CaptionHandler {
	return &CaptionHandler{svc: svc}
}

// ── Categories ───────────────────────────────────────────────────────────────

func (h *CaptionHandler) ListCategories(c *gin.Context) {
	cats, err := h.svc.ListCategories(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, cats)
}

type categoryRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *CaptionHandler) CreateCategory(c *gin.Context) {
	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cat, err := h.svc.CreateCategory(c.Request.Context(), req.Name)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusCreated, cat)
}

func (h *CaptionHandler) UpdateCategory(c *gin.Context) {
	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cat, err := h.svc.UpdateCategory(c.Request.Context(), c.Param("id"), req.Name)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, cat)
}

func (h *CaptionHandler) DeleteCategory(c *gin.Context) {
	if err := h.svc.DeleteCategory(c.Request.Context(), c.Param("id")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ── Templates ────────────────────────────────────────────────────────────────

func (h *CaptionHandler) ListTemplates(c *gin.Context) {
	templates, err := h.svc.ListTemplates(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, templates)
}

type createTemplateRequest struct {
	Title       string   `json:"title" binding:"required"`
	Body        string   `json:"body" binding:"required"`
	CategoryIDs []string `json:"category_ids"`
}

func (h *CaptionHandler) CreateTemplate(c *gin.Context) {
	var req createTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	template, err := h.svc.CreateTemplate(c.Request.Context(), req.Title, req.Body, req.CategoryIDs)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusCreated, template)
}

type updateTemplateRequest struct {
	Title       *string   `json:"title"`
	CategoryIDs *[]string `json:"category_ids"`
}

func (h *CaptionHandler) UpdateTemplate(c *gin.Context) {
	var req updateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.UpdateTemplateMeta(c.Request.Context(), c.Param("id"), req.Title, req.CategoryIDs); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *CaptionHandler) DeleteTemplate(c *gin.Context) {
	if err := h.svc.DeleteTemplate(c.Request.Context(), c.Param("id")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ── Versions ─────────────────────────────────────────────────────────────────

func (h *CaptionHandler) ListVersions(c *gin.Context) {
	versions, err := h.svc.ListVersions(c.Request.Context(), c.Param("id"))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, versions)
}

type addVersionRequest struct {
	Body string `json:"body" binding:"required"`
}

func (h *CaptionHandler) AddVersion(c *gin.Context) {
	var req addVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	version, err := h.svc.AddVersion(c.Request.Context(), c.Param("id"), req.Body)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusCreated, version)
}

func (h *CaptionHandler) RevertVersion(c *gin.Context) {
	if err := h.svc.RevertToVersion(c.Request.Context(), c.Param("id"), c.Param("versionId")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
