package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
)

func NewActivityHandler(svc ActivityManager) *ActivityHandler {
	return &ActivityHandler{svc: svc}
}

func (h *ActivityHandler) ListByBike(c *gin.Context) {
	activities, err := h.svc.ListByBikeID(c.Request.Context(), c.Param("id"))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, activities)
}

func (h *ActivityHandler) Create(c *gin.Context) {
	createForBike(func(in *models.ActivityInput, id string) { in.BikeID = id }, h.svc.Create)(c)
}

func (h *ActivityHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("activityId")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
