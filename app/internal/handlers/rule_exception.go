package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRuleExceptionHandler(exceptions RuleExceptionManager, categories CategoryManager) *RuleExceptionHandler {
	return &RuleExceptionHandler{exceptions: exceptions, categories: categories}
}

// ListByAccount returns the rule IDs that are disabled for the given account.
func (h *RuleExceptionHandler) ListByAccount(c *gin.Context) {
	accountID := c.Param("id")
	ids, err := h.exceptions.FindByAccount(c.Request.Context(), accountID)
	if err != nil {
		internalError(c, err)
		return
	}
	if ids == nil {
		ids = []string{}
	}
	c.JSON(http.StatusOK, ids)
}

// Disable marks a global rule as disabled for the given account.
func (h *RuleExceptionHandler) Disable(c *gin.Context) {
	accountID := c.Param("id")
	var body struct {
		RuleID string `json:"rule_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rule_id is required"})
		return
	}
	if err := h.exceptions.Create(c.Request.Context(), accountID, body.RuleID); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Enable removes the exception, re-enabling the global rule for the given account.
func (h *RuleExceptionHandler) Enable(c *gin.Context) {
	accountID := c.Param("id")
	ruleID := c.Param("rule_id")
	if err := h.exceptions.Delete(c.Request.Context(), accountID, ruleID); err != nil {
		internalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
