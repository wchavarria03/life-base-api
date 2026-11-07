package bac

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"life-base-api/app/internal/models"
	"life-base-api/app/internal/parser"
)

func init() {
	parser.Register(&creditCardParser{})
}

type creditCardParser struct{}

func (p *creditCardParser) Name() string { return "bac/creditcard" }

// Detect identifies BAC credit card statements — a different layout from the
// checking/savings statements: dual-currency (Local/Dólares) columns, no
// running balance per row, headed by "Transacciones del periodo".
func (p *creditCardParser) Detect(text string) bool {
	return strings.Contains(text, "Transacciones del periodo") &&
		strings.Contains(text, "Producto:") &&
		strings.Contains(text, "Fecha de corte:")
}

var (
	ccProductPattern     = regexp.MustCompile(`Producto:\s*(\d{4}-\d{2}\*\*-\*{4}-\d{4})`)
	ccClosingDatePattern = regexp.MustCompile(`Fecha de corte:\s*(\d{2}/\d{2}/\d{4})`)
	// The interest charged this period isn't itemized as its own row in the
	// transactions table — it only shows up in this footer summary — so it's
	// pulled out separately and synthesized as its own expense transaction
	// below, dated at the statement's closing date.
	ccInterestLocalPattern   = regexp.MustCompile(`MES/LOCAL:\s*(-?[\d,]+\.\d{2})`)
	ccInterestDollarsPattern = regexp.MustCompile(`MES/D[OÓ]LARES:\s*(-?[\d,]+\.\d{2})`)
	// Applied globally (FindAllStringSubmatch) across the whole cell-delimited
	// statement text (see pdf.ExtractCellsFromBytes — this format's PDF has no
	// per-line structure to split on, so rows aren't isolated by newlines).
	// The description is non-greedy so it stops at the first point a valid
	// amount pair follows — safe now that cell boundaries are real whitespace,
	// unlike the raw PDF extraction where adjacent cells (e.g. a reference
	// number immediately followed by an amount) have no separator at all and
	// can't be reliably told apart.
	ccRowPattern = regexp.MustCompile(`\s*(?:(\d{2}/\d{2}/\d{4})\s+)?(.+?)\s+(-?[\d,]+\.\d{2})\s+(-?[\d,]+\.\d{2})`)
)

// ccRow is one parsed row before it's split into per-currency transactions.
type ccRow struct {
	date        time.Time
	description string
	local       decimal.Decimal
	dollars     decimal.Decimal
}

func (p *creditCardParser) Parse(text string) (*models.Statement, error) {
	productMatch := ccProductPattern.FindStringSubmatch(text)
	if productMatch == nil {
		return nil, fmt.Errorf("bac/creditcard: could not find product number")
	}
	productNumber := productMatch[1]
	closingDate := ccClosingDate(text)

	rows, prevBalanceLocal, prevBalanceDollars, havePrevBalance := parseCCRows(text, closingDate)
	if !havePrevBalance {
		return nil, fmt.Errorf("bac/creditcard: could not find previous balance")
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("bac/creditcard: no transactions found — verify the PDF matches this format")
	}

	// The PDF's own sign convention is inverted from the rest of this app: a
	// purchase (which increases what you owe) prints as positive, and a
	// payment/credit (which reduces it) prints as negative. Negating puts
	// purchases in the red like every other expense and payments/credits in
	// the green, consistent with checking/savings accounts. Balances are
	// tracked in the same inverted sign, so a running balance is a negative
	// "amount owed" — like a liability.
	runningLocal := prevBalanceLocal.Neg()
	runningDollars := prevBalanceDollars.Neg()

	transactions := buildCCTransactions(rows, &runningLocal, &runningDollars)

	if tx := ccInterestTx(text, ccInterestLocalPattern, closingDate, &runningLocal, "CRC"); tx != nil {
		transactions = append(transactions, *tx)
	}
	if tx := ccInterestTx(text, ccInterestDollarsPattern, closingDate, &runningDollars, "USD"); tx != nil {
		transactions = append(transactions, *tx)
	}

	stampImportSeq(transactions)

	return &models.Statement{
		AccountNumber: productNumber,
		Transactions:  transactions,
	}, nil
}

// ccClosingDate returns the statement's closing date, or the zero time when
// it is absent or unparseable.
func ccClosingDate(text string) time.Time {
	var closingDate time.Time
	if m := ccClosingDatePattern.FindStringSubmatch(text); m != nil {
		if t, err := time.Parse("02/01/2006", m[1]); err == nil {
			closingDate = t
		}
	}
	return closingDate
}

// parseCCRows extracts the monetary rows and the previous-balance row from
// the cell-delimited statement text.
func parseCCRows(text string, closingDate time.Time) (rows []ccRow, prevLocal, prevDollars decimal.Decimal, havePrev bool) {
	for _, m := range ccRowPattern.FindAllStringSubmatch(text, -1) {
		dateStr := m[1]
		description := strings.TrimSpace(m[2])
		local := parseAmount(m[3])
		dollars := parseAmount(m[4])

		if strings.Contains(description, "Previous balance") {
			prevLocal, prevDollars = local, dollars
			havePrev = true
			continue
		}

		if local.IsZero() && dollars.IsZero() {
			continue // informational match (headers, rates, cash-back, limits...)
		}

		// Undated-but-monetary rows (e.g. "REVERSION INTERES CORRIENTES
		// PERIODO") apply to the whole billing period — stamp them with the
		// statement's closing date.
		date := closingDate
		if dateStr != "" {
			t, err := time.Parse("02/01/2006", dateStr)
			if err != nil {
				continue
			}
			date = t
		}

		rows = append(rows, ccRow{date: date, description: description, local: local, dollars: dollars})
	}
	return rows, prevLocal, prevDollars, havePrev
}

// buildCCTransactions splits each row into per-currency transactions,
// advancing the running balances in place. Local (CRC) comes before dollars
// (USD) for every row, as in the statement.
func buildCCTransactions(rows []ccRow, runningLocal, runningDollars *decimal.Decimal) []models.Transaction {
	var transactions []models.Transaction
	for _, r := range rows {
		if !r.local.IsZero() {
			transactions = append(transactions, ccRowTx(r, r.local, runningLocal, "CRC"))
		}
		if !r.dollars.IsZero() {
			transactions = append(transactions, ccRowTx(r, r.dollars, runningDollars, "USD"))
		}
	}
	return transactions
}

// ccRowTx builds one transaction for a row in one currency, negating the
// printed amount and advancing running.
func ccRowTx(r ccRow, printed decimal.Decimal, running *decimal.Decimal, currency string) models.Transaction {
	amount := printed.Neg()
	*running = running.Add(amount)
	return models.Transaction{
		Date:        r.date,
		Type:        deriveCreditCardType(r.description, amount),
		Description: r.description,
		Amount:      amount,
		Balance:     *running,
		Currency:    currency,
	}
}

// ccInterestTx synthesizes the period's interest expense from the footer
// summary, or returns nil when absent or not positive. It advances running.
func ccInterestTx(text string, pattern *regexp.Regexp, closingDate time.Time, running *decimal.Decimal, currency string) *models.Transaction {
	m := pattern.FindStringSubmatch(text)
	if m == nil {
		return nil
	}
	interest := parseAmount(m[1])
	if !interest.IsPositive() {
		return nil
	}
	amount := interest.Neg()
	*running = running.Add(amount)
	return &models.Transaction{
		Date:        closingDate,
		Type:        models.TypeExpense,
		Description: "Intereses corrientes del mes",
		Amount:      amount,
		Balance:     *running,
		Currency:    currency,
	}
}

// deriveCreditCardType classifies a credit-card statement row. amount is
// already sign-inverted to match the app's convention (purchases negative).
//
// "SU PAGO RECIBIDO" rows are the credit-card side of a payment already
// recorded as an outgoing transaction on a checking/savings account, so
// they're a transfer, not income. Any other credit (e.g. an interest
// reversal) is treated as a refund (income); ordinary purchases are expenses.
func deriveCreditCardType(description string, amount decimal.Decimal) models.TransactionType {
	if strings.Contains(strings.ToUpper(description), "SU PAGO RECIBIDO") {
		return models.TypeTransfer
	}
	if amount.IsPositive() {
		return models.TypeIncome
	}
	return models.TypeExpense
}
