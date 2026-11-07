package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
)

func NewReportHandler(accounts AccountLister, summarizer ReportSummarizer) *ReportHandler {
	return &ReportHandler{accounts: accounts, summarizer: summarizer}
}

func (h *ReportHandler) GetSummary(c *gin.Context) {
	ctx := c.Request.Context()

	from, to, fromStr, toStr, ok := parseDateRange(c)
	if !ok {
		return
	}

	accountIDs, ok := h.summaryAccountIDs(c, c.Query("account_id"), c.Query("currency"))
	if !ok {
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

// summaryAccountIDs resolves which accounts a summary covers: one account by
// id, or every open, non-external account in a currency. It writes the error
// response itself and returns ok=false when neither is usable.
func (h *ReportHandler) summaryAccountIDs(c *gin.Context, accountID, currency string) ([]string, bool) {
	ctx := c.Request.Context()
	switch {
	case accountID != "":
		acct, err := h.accounts.GetByID(ctx, accountID)
		if err != nil || acct == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return nil, false
		}
		return []string{accountID}, true
	case currency != "":
		accounts, err := h.accounts.List(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list accounts"})
			return nil, false
		}
		var ids []string
		for _, a := range accounts {
			if a.Currency == currency && !a.Locked && !a.External {
				ids = append(ids, a.ID)
			}
		}
		return ids, true
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id or currency is required"})
		return nil, false
	}
}

// NetWorth handles GET /v1/reports/net-worth?from=&to= — one balance-history
// summary per currency the caller has accounts in, side by side. No FX
// conversion: each currency is its own independent total, same as the
// existing per-currency filter in GetSummary, just run for every currency
// present instead of requiring the caller to pick one.
func (h *ReportHandler) NetWorth(c *gin.Context) {
	ctx := c.Request.Context()

	from, to, _, _, ok := parseDateRange(c)
	if !ok {
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
