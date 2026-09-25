package parser

import (
	"bufio"
	"fmt"
	"os"

	"life-base-api/app/internal/models"
)

func WriteTransactions(path string, transactions []models.Transaction) error {
	f, err := os.Create(path) //nolint:gosec // path is CLI-supplied (--output flag), not HTTP-reachable
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, tx := range transactions {
		line := fmt.Sprintf("%s--%s--%s--%s--%s--%s--%s--%s\n",
			tx.Date.Format("2006-01-02"),
			tx.Reference,
			tx.Code,
			string(tx.Type),
			tx.Description,
			tx.Amount.StringFixed(2),
			tx.Balance.StringFixed(2),
			tx.Currency,
		)
		if _, err := w.WriteString(line); err != nil {
			return err
		}
	}
	return w.Flush()
}
