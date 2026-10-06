package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewHouseTimerHandler constructs a HouseTimerHandler.
func NewHouseTimerHandler(svc HouseTimerManager) *HouseTimerHandler {
	return &HouseTimerHandler{svc: svc}
}

// List handles GET /v1/house-timers.
func (h *HouseTimerHandler) List(c *gin.Context) {
	listHandler(h.svc.List)(c)
}

// Create handles POST /v1/house-timers.
func (h *HouseTimerHandler) Create(c *gin.Context) {
	var req struct {
		Label   string `json:"label" binding:"required"`
		Message string `json:"message"`
		Minutes int    `json:"minutes" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	timer, err := h.svc.Create(c.Request.Context(), req.Label, req.Message, req.Minutes)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, timer)
}

// MarkAnnounced handles POST /v1/house-timers/:id/announced.
func (h *HouseTimerHandler) MarkAnnounced(c *gin.Context) {
	deleteHandler(h.svc.MarkAnnounced)(c)
}

// Delete handles DELETE /v1/house-timers/:id.
func (h *HouseTimerHandler) Delete(c *gin.Context) {
	deleteHandler(h.svc.Delete)(c)
}
