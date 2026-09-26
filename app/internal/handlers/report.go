package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
)

func NewReportHandler(accounts AccountLister, summarizer ReportSummarizer) *ReportHandler {
	return &ReportHandler{accounts: accounts, summarizer: summarizer}
}

func (h *ReportHandler) GetSummary(c *gin.Context) {
	ctx := c.Request.Context()

	fromStr := c.Query("from")
	toStr := c.Query("to")
	accountID := c.Query("account_id")
	currency := c.Query("currency")

	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to are required"})
		return
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from date, expected YYYY-MM-DD"})
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to date, expected YYYY-MM-DD"})
		return
	}

	var accountIDs []string

	if accountID != "" {
		acct, err := h.accounts.GetByID(ctx, accountID)
		if err != nil || acct == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		accountIDs = []string{accountID}
	} else if currency != "" {
		accounts, err := h.accounts.List(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list accounts"})
			return
		}
		for _, a := range accounts {
			if a.Currency == currency && !a.Locked && !a.External {
				accountIDs = append(accountIDs, a.ID)
			}
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id or currency is required"})
		return
	}

	if len(accountIDs) == 0 {
		c.JSON(http.StatusOK, &models.ReportSummary{
			PeriodStart:    fromStr,
			PeriodEnd:      toStr,
			BalanceHistory: []models.DailyBalance{},
			DailyChanges:   []models.DailyChange{},
			ByCategory:     []models.CategorySpend{},
		})
		return
	}

	summary, err := h.summarizer.Summarize(ctx, accountIDs, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate report"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// NetWorth handles GET /v1/reports/net-worth?from=&to= — one balance-history
// summary per currency the caller has accounts in, side by side. No FX
// conversion: each currency is its own independent total, same as the
// existing per-currency filter in GetSummary, just run for every currency
// present instead of requiring the caller to pick one.
func (h *ReportHandler) NetWorth(c *gin.Context) {
	ctx := c.Request.Context()

	fromStr := c.Query("from")
	toStr := c.Query("to")
	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to are required"})
		return
	}
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from date, expected YYYY-MM-DD"})
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to date, expected YYYY-MM-DD"})
		return
	}

	accounts, err := h.accounts.List(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list accounts"})
		return
	}

	byCurrency := make(map[string][]string)
	for _, a := range accounts {
		if a.Locked || a.External {
			continue
		}
		byCurrency[a.Currency] = append(byCurrency[a.Currency], a.ID)
	}

	result := make(map[string]*models.ReportSummary, len(byCurrency))
	for currency, accountIDs := range byCurrency {
		summary, err := h.summarizer.Summarize(ctx, accountIDs, from, to)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate report"})
			return
		}
		result[currency] = summary
	}

	c.JSON(http.StatusOK, result)
}
