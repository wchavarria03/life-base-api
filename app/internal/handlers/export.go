package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
)

// exportTxLimit is generous for a personal-scale dataset — this endpoint
// returns one JSON blob, not a paginated stream, so it fetches everything
// in one page per account rather than building real pagination for export.
const exportTxLimit = 100_000

// NewExportHandler constructs an ExportHandler.
func NewExportHandler(
	accounts AccountLister,
	transactions TransactionLister,
	categories CategoryManager,
	budgets BudgetManager,
	envelopes EnvelopeManager,
	reminders ReminderManager,
	social SocialPoster,
) *ExportHandler {
	return &ExportHandler{
		accounts:     accounts,
		transactions: transactions,
		categories:   categories,
		budgets:      budgets,
		envelopes:    envelopes,
		reminders:    reminders,
		social:       social,
	}
}

type exportBundle struct {
	ExportedAt   time.Time                   `json:"exported_at"`
	Accounts     []*models.Account           `json:"accounts"`
	Transactions []*models.Transaction       `json:"transactions"`
	Categories   []*models.Category          `json:"categories"`
	Budgets      []models.BudgetStatus       `json:"budgets"`
	Envelopes    []models.EnvelopeStatus     `json:"envelopes"`
	Reminders    []models.ReminderWithStatus `json:"reminders"`
	SocialPosts  []*models.SocialPost        `json:"social_posts"`
}

// Handle serves GET /v1/export — a single JSON dump of everything the
// calling user owns, for backup/portability. Best-effort per section: one
// failing section doesn't block the rest of the export.
func (h *ExportHandler) Handle(c *gin.Context) {
	ctx := c.Request.Context()
	bundle := exportBundle{ExportedAt: time.Now().UTC()}

	if accounts, err := h.accounts.List(ctx); err == nil {
		bundle.Accounts = accounts
		for _, a := range accounts {
			txs, _, err := h.transactions.ListFiltered(ctx, a.ID, models.TxFilter{Limit: exportTxLimit})
			if err != nil {
				continue
			}
			bundle.Transactions = append(bundle.Transactions, txs...)
		}
	}
	if cats, err := h.categories.List(ctx); err == nil {
		bundle.Categories = cats
	}
	if budgets, err := h.budgets.List(ctx, time.Now().UTC().Format("2006-01")); err == nil {
		bundle.Budgets = budgets
	}
	if envelopes, err := h.envelopes.List(ctx); err == nil {
		bundle.Envelopes = envelopes
	}
	if reminders, err := h.reminders.List(ctx); err == nil {
		bundle.Reminders = reminders
	}
	if posts, err := h.social.List(ctx, exportTxLimit, 0, nil); err == nil {
		bundle.SocialPosts = posts
	}

	c.Header("Content-Disposition", `attachment; filename="life-base-export.json"`)
	c.JSON(http.StatusOK, bundle)
}
