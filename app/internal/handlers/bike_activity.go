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
	var input models.ActivityInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.BikeID = c.Param("id")
	activity, err := h.svc.Create(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, activity)
}

func (h *ActivityHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("activityId")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
