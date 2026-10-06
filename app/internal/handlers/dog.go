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
	listHandler(h.svc.ListDogs)(c)
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
	deleteHandler(h.svc.DeleteDog)(c)
}

// ── Recipient types ──────────────────────────────────────────────────────────

// ListRecipientTypes handles GET /v1/dog-recipient-types.
func (h *DogHandler) ListRecipientTypes(c *gin.Context) {
	listHandler(h.svc.ListRecipientTypes)(c)
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
	deleteHandler(h.svc.DeleteRecipientType)(c)
}

// ── Recipient allocations ────────────────────────────────────────────────────

// ListAllocations handles GET /v1/dog-recipient-allocations.
func (h *DogHandler) ListAllocations(c *gin.Context) {
	listHandler(h.svc.ListAllocations)(c)
}

// PortionBatch handles POST /v1/dog-recipient-allocations/portion.
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

// MarkFed handles POST /v1/dogs/:id/feed.
func (h *DogHandler) MarkFed(c *gin.Context) {
	fed, err := h.svc.MarkFed(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	if !fed {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "no portioned food available for this dog"})
		return
	}
	c.Status(http.StatusNoContent)
}

// CatchUpFeeds handles POST /v1/dogs/:id/catch-up-feed. It batch-logs up to
// count missed feeds (used by the feed-review banner's "log the missing
// feeds now" action) and reports how many were actually logged, since
// available allocated/bulk stock may not cover the full shortfall.
func (h *DogHandler) CatchUpFeeds(c *gin.Context) {
	var req struct {
		Count int `json:"count" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	logged, err := h.svc.CatchUpFeeds(c.Request.Context(), c.Param("id"), req.Count)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logged": logged})
}

// FeedReview handles GET /v1/dogs/feed-review — the shortfall between
// expected and actually-logged feeds per dog over the last
// services.DogFeedReviewLookbackDays days.
func (h *DogHandler) FeedReview(c *gin.Context) {
	listHandler(h.svc.FeedReview)(c)
}

// ── Bulk bags ────────────────────────────────────────────────────────────────

// ListBulkBags handles GET /v1/dog-bulk-bags.
func (h *DogHandler) ListBulkBags(c *gin.Context) {
	listHandler(h.svc.ListBulkBags)(c)
}

// CreateBulkBag handles POST /v1/dog-bulk-bags.
func (h *DogHandler) CreateBulkBag(c *gin.Context) {
	createResource(c, h.svc.CreateBulkBag)
}

// DeleteBulkBag handles DELETE /v1/dog-bulk-bags/:id.
func (h *DogHandler) DeleteBulkBag(c *gin.Context) {
	deleteHandler(h.svc.DeleteBulkBag)(c)
}

// ── Settings ─────────────────────────────────────────────────────────────────

// GetSettings handles GET /v1/dog-settings.
func (h *DogHandler) GetSettings(c *gin.Context) {
	listHandler(h.svc.GetSettings)(c)
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
