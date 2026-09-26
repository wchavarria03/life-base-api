package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
)

// createResource binds a JSON body of type In, calls create, and responds
// 201 with the result or 422 with the error — the shape shared by every
// simple "create a resource" handler below.
func createResource[In, Out any](c *gin.Context, create func(context.Context, In) (Out, error)) {
	var input In
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := create(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, out)
}

// updateResource binds a JSON body of type In, calls update with the :id
// param, and responds 200 with the result or 422 with the error.
func updateResource[In, Out any](c *gin.Context, update func(context.Context, string, In) (Out, error)) {
	var input In
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := update(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

// NewDogHandler constructs a DogHandler.
func NewDogHandler(svc DogManager) *DogHandler {
	return &DogHandler{svc: svc}
}

// ── Dogs ─────────────────────────────────────────────────────────────────────

// ListDogs handles GET /v1/dogs.
func (h *DogHandler) ListDogs(c *gin.Context) {
	dogs, err := h.svc.ListDogs(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, dogs)
}

// CreateDog handles POST /v1/dogs.
func (h *DogHandler) CreateDog(c *gin.Context) {
	createResource(c, h.svc.CreateDog)
}

// UpdateDog handles PATCH /v1/dogs/:id.
func (h *DogHandler) UpdateDog(c *gin.Context) {
	updateResource(c, h.svc.UpdateDog)
}

// DeleteDog handles DELETE /v1/dogs/:id.
func (h *DogHandler) DeleteDog(c *gin.Context) {
	if err := h.svc.DeleteDog(c.Request.Context(), c.Param("id")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ── Recipient types ──────────────────────────────────────────────────────────

// ListRecipientTypes handles GET /v1/dog-recipient-types.
func (h *DogHandler) ListRecipientTypes(c *gin.Context) {
	types, err := h.svc.ListRecipientTypes(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, types)
}

// CreateRecipientType handles POST /v1/dog-recipient-types.
func (h *DogHandler) CreateRecipientType(c *gin.Context) {
	createResource(c, h.svc.CreateRecipientType)
}

// UpdateRecipientType handles PATCH /v1/dog-recipient-types/:id.
func (h *DogHandler) UpdateRecipientType(c *gin.Context) {
	updateResource(c, h.svc.UpdateRecipientType)
}

// DeleteRecipientType handles DELETE /v1/dog-recipient-types/:id.
func (h *DogHandler) DeleteRecipientType(c *gin.Context) {
	if err := h.svc.DeleteRecipientType(c.Request.Context(), c.Param("id")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ── Recipients ───────────────────────────────────────────────────────────────

// ListRecipients handles GET /v1/dog-recipients?status=.
func (h *DogHandler) ListRecipients(c *gin.Context) {
	recipients, err := h.svc.ListRecipients(c.Request.Context(), c.Query("status"))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, recipients)
}

// PortionBatch handles POST /v1/dog-recipients/portion.
func (h *DogHandler) PortionBatch(c *gin.Context) {
	var req struct {
		Requests []models.PortionRequest `json:"requests" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.PortionBatch(c.Request.Context(), req.Requests); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// MarkFed handles POST /v1/dog-recipients/:id/feed.
func (h *DogHandler) MarkFed(c *gin.Context) {
	if err := h.svc.MarkFed(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ── Bulk bags ────────────────────────────────────────────────────────────────

// ListBulkBags handles GET /v1/dog-bulk-bags.
func (h *DogHandler) ListBulkBags(c *gin.Context) {
	bags, err := h.svc.ListBulkBags(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, bags)
}

// CreateBulkBag handles POST /v1/dog-bulk-bags.
func (h *DogHandler) CreateBulkBag(c *gin.Context) {
	createResource(c, h.svc.CreateBulkBag)
}

// DeleteBulkBag handles DELETE /v1/dog-bulk-bags/:id.
func (h *DogHandler) DeleteBulkBag(c *gin.Context) {
	if err := h.svc.DeleteBulkBag(c.Request.Context(), c.Param("id")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ── Settings ─────────────────────────────────────────────────────────────────

// GetSettings handles GET /v1/dog-settings.
func (h *DogHandler) GetSettings(c *gin.Context) {
	settings, err := h.svc.GetSettings(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, settings)
}

// SetSettings handles PUT /v1/dog-settings.
func (h *DogHandler) SetSettings(c *gin.Context) {
	var req struct {
		LowStockThresholdGrams int `json:"low_stock_threshold_grams"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	settings, err := h.svc.SetSettings(c.Request.Context(), req.LowStockThresholdGrams)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, settings)
}
