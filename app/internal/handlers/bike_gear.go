package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
)

func NewGearHandler(svc GearManager) *GearHandler {
	return &GearHandler{svc: svc}
}

func (h *GearHandler) List(c *gin.Context) {
	gear, err := h.svc.List(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gear)
}

func (h *GearHandler) Create(c *gin.Context) {
	var input models.GearInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gear, err := h.svc.Create(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gear)
}

func (h *GearHandler) Update(c *gin.Context) {
	var fields map[string]any
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gear, err := h.svc.Update(c.Request.Context(), c.Param("id"), fields)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gear)
}

func (h *GearHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func NewBottleHandler(svc BottleManager) *BottleHandler {
	return &BottleHandler{svc: svc}
}

func (h *BottleHandler) List(c *gin.Context) {
	bottles, err := h.svc.List(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, bottles)
}

func (h *BottleHandler) Create(c *gin.Context) {
	var input models.BottleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	bottle, err := h.svc.Create(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, bottle)
}

func (h *BottleHandler) Update(c *gin.Context) {
	var fields map[string]any
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	bottle, err := h.svc.Update(c.Request.Context(), c.Param("id"), fields)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bottle)
}

func (h *BottleHandler) MarkCleaned(c *gin.Context) {
	bottle, err := h.svc.MarkCleaned(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bottle)
}

func (h *BottleHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func NewSupplyHandler(svc SupplyManager) *SupplyHandler {
	return &SupplyHandler{svc: svc}
}

func (h *SupplyHandler) List(c *gin.Context) {
	supplies, err := h.svc.List(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, supplies)
}

func (h *SupplyHandler) Create(c *gin.Context) {
	var input models.SupplyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	supply, err := h.svc.Create(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, supply)
}

func (h *SupplyHandler) Update(c *gin.Context) {
	var fields map[string]any
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	supply, err := h.svc.Update(c.Request.Context(), c.Param("id"), fields)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, supply)
}

func (h *SupplyHandler) Deplete(c *gin.Context) {
	entry, err := h.svc.Deplete(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entry)
}

func (h *SupplyHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SupplyHandler) ListHistory(c *gin.Context) {
	history, err := h.svc.ListHistory(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, history)
}
