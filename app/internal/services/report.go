package services

import (
	"context"
	"math"
	"sort"
	"time"

	"life-base-api/app/internal/models"
)

// uncategorizedID/uncategorizedColor label the synthetic bucket used for
// expenses with no category assigned, so they still surface in "by_category"
// totals instead of silently disappearing from the chart.
const (
	uncategorizedID    = "uncategorized"
	uncategorizedColor = "#6b7280"
)

func NewReportService(repo TransactionRepository, cats CategoryRepository) *ReportService {
	return &ReportService{repo: repo, categories: cats}
}

func (s *ReportService) Summarize(ctx context.Context, accountIDs []string, from, to time.Time) (*models.ReportSummary, error) {
	// Last known balance per account strictly before the period — the starting point.
	carryPerAccount, err := s.repo.GetLastBalancePerAccount(ctx, accountIDs, from)
	if err != nil {
		return nil, err
	}

	txs, err := s.repo.GetByAccountIDsInRange(ctx, accountIDs, from, to)
	if err != nil {
		return nil, err
	}

	allCats, err := s.categories.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	catByID := make(map[string]*models.Category, len(allCats))
	for _, c := range allCats {
		catByID[c.ID] = c
	}

	summary := &models.ReportSummary{
		PeriodStart:    from.Format("2006-01-02"),
		PeriodEnd:      to.Format("2006-01-02"),
		BalanceHistory: []models.DailyBalance{},
		DailyChanges:   []models.DailyChange{},
		ByCategory:     []models.CategorySpend{},
	}

	acc := newReportAccumulator(summary, catByID, accountIDs, carryPerAccount)
	for _, tx := range txs { // sorted date asc by the repo query
		acc.add(tx)
	}

	totalBalance := 0.0
	for _, bal := range acc.lastBalPerAccount {
		totalBalance += bal
	}
	summary.TotalBalance = totalBalance
	summary.PeriodChange = summary.TotalIncome - summary.TotalExpenses

	carryTotal := 0.0
	for _, bal := range carryPerAccount {
		carryTotal += bal
	}
	acc.fillDaily(from, to, carryTotal)
	acc.fillCategories()

	return summary, nil
}

// reportAccumulator gathers the per-day and per-category figures while
// Summarize walks the period's transactions. dailyBalance is the summed
// balance across all accounts for each day that had at least one transaction;
// lastBalPerAccount is seeded from the carry values so accounts with no
// in-period transactions still contribute to the running total.
type reportAccumulator struct {
	summary           *models.ReportSummary
	catByID           map[string]*models.Category
	dailyIncome       map[string]float64
	dailyExpenses     map[string]float64
	dailyBalance      map[string]float64
	categoryTotals    map[string]*models.CategorySpend
	lastBalPerAccount map[string]float64
}

func newReportAccumulator(summary *models.ReportSummary, catByID map[string]*models.Category, accountIDs []string, carry map[string]float64) *reportAccumulator {
	last := make(map[string]float64, len(accountIDs))
	for _, id := range accountIDs {
		last[id] = carry[id]
	}
	return &reportAccumulator{
		summary:           summary,
		catByID:           catByID,
		dailyIncome:       map[string]float64{},
		dailyExpenses:     map[string]float64{},
		dailyBalance:      map[string]float64{},
		categoryTotals:    map[string]*models.CategorySpend{},
		lastBalPerAccount: last,
	}
}

// add folds one transaction into the running figures.
func (a *reportAccumulator) add(tx *models.Transaction) {
	day := tx.Date.Format("2006-01-02")
	amount, _ := tx.Amount.Float64()
	bal, _ := tx.Balance.Float64()

	a.lastBalPerAccount[tx.AccountID] = bal

	// Sum all accounts' latest balance to get the portfolio total for this day.
	dayTotal := 0.0
	for _, b := range a.lastBalPerAccount {
		dayTotal += b
	}
	a.dailyBalance[day] = dayTotal

	switch tx.Type {
	case models.TypeIncome:
		a.summary.TotalIncome += amount
		a.dailyIncome[day] += amount
	case models.TypeExpense:
		a.addExpense(day, math.Abs(amount), tx.Categories) // expenses are stored as negatives; normalize to positive
	case models.TypeTransfer:
		if amount >= 0 {
			a.summary.Transfers.IncomingCount++
			a.summary.Transfers.IncomingTotal += amount
		} else {
			a.summary.Transfers.OutgoingCount++
			a.summary.Transfers.OutgoingTotal += math.Abs(amount)
		}
	}
}

// addExpense books a positive expense amount against the day and against each
// root category (or "Uncategorized" when the transaction has none).
func (a *reportAccumulator) addExpense(day string, absAmount float64, cats []*models.Category) {
	a.summary.TotalExpenses += absAmount
	a.dailyExpenses[day] += absAmount
	roots := resolveRootCategories(cats, a.catByID)
	if len(roots) == 0 {
		a.addCategorySpend(uncategorizedID, "Uncategorized", uncategorizedColor, absAmount)
	}
	for _, root := range roots {
		a.addCategorySpend(root.ID, root.Name, root.Color, absAmount)
	}
}

func (a *reportAccumulator) addCategorySpend(id, name, color string, absAmount float64) {
	if _, ok := a.categoryTotals[id]; !ok {
		a.categoryTotals[id] = &models.CategorySpend{
			CategoryID:   id,
			CategoryName: name,
			Color:        color,
		}
	}
	a.categoryTotals[id].Total += absAmount
}

// fillDaily emits one DailyChange and one DailyBalance per calendar day in
// [from, to]; days without transactions carry the previous day's balance.
func (a *reportAccumulator) fillDaily(from, to time.Time, carryTotal float64) {
	prevTotal := carryTotal
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		day := d.Format("2006-01-02")
		a.summary.DailyChanges = append(a.summary.DailyChanges, models.DailyChange{
			Date:     day,
			Income:   a.dailyIncome[day],
			Expenses: a.dailyExpenses[day],
		})
		if total, ok := a.dailyBalance[day]; ok {
			prevTotal = total
		}
		a.summary.BalanceHistory = append(a.summary.BalanceHistory, models.DailyBalance{
			Date:    day,
			Balance: prevTotal,
		})
	}
}

// fillCategories copies the category totals into the summary, largest first.
func (a *reportAccumulator) fillCategories() {
	for _, cs := range a.categoryTotals {
		a.summary.ByCategory = append(a.summary.ByCategory, *cs)
	}
	sort.Slice(a.summary.ByCategory, func(i, j int) bool {
		return a.summary.ByCategory[i].Total > a.summary.ByCategory[j].Total
	})
}

// resolveRootCategories walks each assigned category up one level to its
// parent (root), de-duplicating so a transaction tagged with two children of
// the same parent only counts once against that parent. A transaction with
// multiple distinct roots contributes its full amount to each — spend totals
// intentionally overlap rather than being split arbitrarily between tags.
func resolveRootCategories(cats []*models.Category, catByID map[string]*models.Category) []*models.Category {
	seen := make(map[string]*models.Category, len(cats))
	for _, cat := range cats {
		root := cat
		if cat.ParentID != "" {
			if parent, ok := catByID[cat.ParentID]; ok {
				root = parent
			}
		}
		seen[root.ID] = root
	}

	roots := make([]*models.Category, 0, len(seen))
	for _, root := range seen {
		roots = append(roots, root)
	}
	return roots
}
