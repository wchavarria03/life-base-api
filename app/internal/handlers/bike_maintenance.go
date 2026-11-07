package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
)

func NewServiceLogHandler(svc ServiceLogManager) *ServiceLogHandler {
	return &ServiceLogHandler{svc: svc}
}

func (h *ServiceLogHandler) ListByBike(c *gin.Context) {
	logs, err := h.svc.ListByBikeID(c.Request.Context(), c.Param("id"))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, logs)
}

func (h *ServiceLogHandler) Create(c *gin.Context) {
	createForBike(func(in *models.ServiceLogInput, id string) { in.BikeID = id }, h.svc.Create)(c)
}

func (h *ServiceLogHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("logId")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func NewMaintenanceTaskHandler(svc MaintenanceTaskManager) *MaintenanceTaskHandler {
	return &MaintenanceTaskHandler{svc: svc}
}

func (h *MaintenanceTaskHandler) ListByBike(c *gin.Context) {
	tasks, err := h.svc.ListByBikeID(c.Request.Context(), c.Param("id"))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, tasks)
}

func (h *MaintenanceTaskHandler) Create(c *gin.Context) {
	createForBike(func(in *models.MaintenanceTaskInput, id string) { in.BikeID = id }, h.svc.Create)(c)
}

func (h *MaintenanceTaskHandler) Update(c *gin.Context) {
	updateByParam("taskId", h.svc.Update)(c)
}

func (h *MaintenanceTaskHandler) Complete(c *gin.Context) {
	task, err := h.svc.Complete(c.Request.Context(), c.Param("taskId"), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *MaintenanceTaskHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("taskId")); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
