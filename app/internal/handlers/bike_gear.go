package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewGearHandler(svc GearManager) *GearHandler {
	return &GearHandler{svc: svc}
}

func (h *GearHandler) List(c *gin.Context) {
	listHandler(h.svc.List)(c)
}

func (h *GearHandler) Create(c *gin.Context) {
	createHandler(h.svc.Create)(c)
}

func (h *GearHandler) Update(c *gin.Context) {
	updateHandler(h.svc.Update)(c)
}

func (h *GearHandler) Delete(c *gin.Context) {
	deleteHandler(h.svc.Delete)(c)
}

func NewBottleHandler(svc BottleManager) *BottleHandler {
	return &BottleHandler{svc: svc}
}

func (h *BottleHandler) List(c *gin.Context) {
	listHandler(h.svc.List)(c)
}

func (h *BottleHandler) Create(c *gin.Context) {
	createHandler(h.svc.Create)(c)
}

func (h *BottleHandler) Update(c *gin.Context) {
	updateHandler(h.svc.Update)(c)
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
	deleteHandler(h.svc.Delete)(c)
}

func NewSupplyHandler(svc SupplyManager) *SupplyHandler {
	return &SupplyHandler{svc: svc}
}

func (h *SupplyHandler) List(c *gin.Context) {
	listHandler(h.svc.List)(c)
}

func (h *SupplyHandler) Create(c *gin.Context) {
	createHandler(h.svc.Create)(c)
}

func (h *SupplyHandler) Update(c *gin.Context) {
	updateHandler(h.svc.Update)(c)
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
	deleteHandler(h.svc.Delete)(c)
}

func (h *SupplyHandler) ListHistory(c *gin.Context) {
	listHandler(h.svc.ListHistory)(c)
}
