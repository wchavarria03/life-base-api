package parser

import "life-base-api/app/internal/models"

// BankParser is implemented by each bank/format-specific parser.
type BankParser interface {
	Name() string
	Detect(text string) bool
	Parse(text string) (*models.Statement, error)
}
