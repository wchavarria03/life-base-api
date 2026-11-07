package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
)

func NewBikeHandler(svc BikeManager) *BikeHandler {
	return &BikeHandler{svc: svc}
}

func (h *BikeHandler) List(c *gin.Context) {
	listHandler(h.svc.List)(c)
}

// Summary handles GET /v1/bikes/summary.
func (h *BikeHandler) Summary(c *gin.Context) {
	summary, err := h.svc.DashboardSummary(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *BikeHandler) Get(c *gin.Context) {
	bike, err := h.svc.FindByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		internalError(c, err)
		return
	}
	if bike == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bike not found"})
		return
	}
	c.JSON(http.StatusOK, bike)
}

func (h *BikeHandler) Create(c *gin.Context) {
	createHandler(h.svc.Create)(c)
}

func (h *BikeHandler) Update(c *gin.Context) {
	updateHandler(h.svc.Update)(c)
}

func (h *BikeHandler) Delete(c *gin.Context) {
	deleteHandler(h.svc.Delete)(c)
}

func NewBikeFitHistoryHandler(svc BikeFitHistoryManager) *BikeFitHistoryHandler {
	return &BikeFitHistoryHandler{svc: svc}
}

func (h *BikeFitHistoryHandler) ListByBike(c *gin.Context) {
	history, err := h.svc.ListByBikeID(c.Request.Context(), c.Param("id"))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, history)
}

func (h *BikeFitHistoryHandler) Create(c *gin.Context) {
	createForBike(func(in *models.BikeFitHistoryInput, id string) { in.BikeID = id }, h.svc.Create)(c)
}

func (h *BikeFitHistoryHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("fitId")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
