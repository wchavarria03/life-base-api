package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	defaultAuditLogLimit = 50
	maxAuditLogLimit     = 200
)

// NewAuditLogHandler constructs an AuditLogHandler.
func NewAuditLogHandler(svc AuditLogManager) *AuditLogHandler {
	return &AuditLogHandler{svc: svc}
}

// List handles GET /v1/admin/audit-log?limit=&offset=.
func (h *AuditLogHandler) List(c *gin.Context) {
	limit := defaultAuditLogLimit
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > maxAuditLogLimit {
		limit = maxAuditLogLimit
	}

	offset := 0
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	entries, err := h.svc.List(c.Request.Context(), limit, offset)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, entries)
}
