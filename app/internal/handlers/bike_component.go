package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
)

func NewComponentHandler(svc ComponentManager) *ComponentHandler {
	return &ComponentHandler{svc: svc}
}

func (h *ComponentHandler) ListByBike(c *gin.Context) {
	components, err := h.svc.ListByBikeID(c.Request.Context(), c.Param("id"))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, components)
}

func (h *ComponentHandler) Create(c *gin.Context) {
	var input models.ComponentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.BikeID = c.Param("id")
	component, err := h.svc.Create(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, component)
}

func (h *ComponentHandler) Update(c *gin.Context) {
	var fields map[string]any
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	component, err := h.svc.Update(c.Request.Context(), c.Param("componentId"), fields)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, component)
}

func (h *ComponentHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("componentId")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type replaceComponentRequest struct {
	ReplacedDate         string   `json:"replaced_date"`
	MileageAtReplacement *float64 `json:"mileage_at_replacement,omitempty"`
	Notes                *string  `json:"notes,omitempty"`
}

func (h *ComponentHandler) Replace(c *gin.Context) {
	var req replaceComponentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	component, err := h.svc.Replace(c.Request.Context(), c.Param("componentId"), req.ReplacedDate, req.MileageAtReplacement, req.Notes)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, component)
}

func (h *ComponentHandler) ListHistory(c *gin.Context) {
	history, err := h.svc.ListHistory(c.Request.Context(), c.Param("componentId"))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, history)
}
