package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/models"
	"life-base-api/app/internal/parser"
	_ "life-base-api/app/internal/parser/parsers/bac" // registers the BAC statement parsers
	"life-base-api/app/internal/pdf"
)

const maxUploadBytes = 10 << 20 // 10 MB

func NewUploadHandler(importer StatementImporter) *UploadHandler {
	return &UploadHandler{importer: importer}
}

// Import handles POST /v1/import.
// With ?dry_run=true it parses the PDF and returns a preview without storing anything.
// Without dry_run it imports and returns an ImportSummary.
func (h *UploadHandler) Import(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes)

	data, filename, ok := readUploadedPDF(c)
	if !ok {
		return
	}

	stmt, parserName, ok := parseUploadedStatement(c, data)
	if !ok {
		return
	}
	stmt.SourceFile = filename

	ctx := c.Request.Context()
	if c.Query("dry_run") == "true" {
		c.JSON(http.StatusOK, h.dryRunPreview(ctx, stmt, parserName))
		return
	}

	catOverrides := applyFormOverrides(stmt, c.PostForm("overrides"))

	summary, err := h.importer.ImportWithSummary(ctx, stmt, parserName, catOverrides)
	if err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, summary)
}

// readUploadedPDF reads the multipart "file" field, requiring a .pdf name.
// It writes the error response itself and returns ok=false on failure.
func readUploadedPDF(c *gin.Context) (data []byte, filename string, ok bool) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file field is required"})
		return nil, "", false
	}
	defer func() { _ = file.Close() }()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".pdf") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only PDF files are supported"})
		return nil, "", false
	}

	data, err = io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uploaded file"})
		return nil, "", false
	}
	return data, header.Filename, true
}

// parseUploadedStatement extracts the PDF text, detects the bank parser and
// parses the statement. It writes the error response itself on failure.
func parseUploadedStatement(c *gin.Context, data []byte) (*models.Statement, string, bool) {
	text, err := pdf.ExtractTextFromBytes(data)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "could not extract text from PDF"})
		return nil, "", false
	}

	p, err := parser.Detect(text)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "unsupported bank statement format — only BAC statements are supported right now"})
		return nil, "", false
	}

	// This format's PDF content stream places table cells with no whitespace
	// glyph between them (e.g. two adjacent amount columns come out as
	// "11233.000.00"), which the default extraction can't unambiguously
	// re-split. Re-extract with cell boundaries preserved for this parser only.
	if p.Name() == "bac/creditcard" {
		if cellText, err := pdf.ExtractCellsFromBytes(data); err == nil {
			text = cellText
		}
	}

	stmt, err := p.Parse(text)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "failed to parse statement: " + err.Error()})
		return nil, "", false
	}
	return stmt, p.Name(), true
}

// dryRunPreview builds the no-write preview: parsed rows, how many already
// exist, and the categories import would auto-assign.
func (h *UploadHandler) dryRunPreview(ctx context.Context, stmt *models.Statement, parserName string) *models.ImportPreview {
	preview := buildPreview(stmt, parserName)
	if count, err := h.importer.CheckOverlap(ctx, stmt); err == nil {
		preview.ExistingCount = count
	}
	for idx, catID := range h.importer.PreviewCategories(ctx, stmt) {
		if idx >= 0 && idx < len(preview.Transactions) {
			preview.Transactions[idx].Categories = []*models.Category{{ID: catID}}
		}
	}
	return preview
}

// applyFormOverrides applies the optional "overrides" JSON field (per-row
// edits from the preview screen) to stmt and returns the category choices
// keyed by row index. Malformed JSON is ignored, as before.
func applyFormOverrides(stmt *models.Statement, raw string) map[int][]string {
	catOverrides := map[int][]string{}
	if raw == "" {
		return catOverrides
	}
	var overrides []models.TransactionOverride
	if err := json.Unmarshal([]byte(raw), &overrides); err != nil {
		return catOverrides
	}
	applyOverrides(stmt, overrides)
	for _, o := range overrides {
		if len(o.CategoryIDs) > 0 {
			catOverrides[o.Index] = o.CategoryIDs
		}
	}
	return catOverrides
}

func applyOverrides(stmt *models.Statement, overrides []models.TransactionOverride) {
	for _, o := range overrides {
		if o.Index < 0 || o.Index >= len(stmt.Transactions) {
			continue
		}
		tx := &stmt.Transactions[o.Index]
		if o.Date != "" {
			if t, err := time.Parse("2006-01-02", o.Date); err == nil {
				tx.Date = t
			}
		}
		if o.Description != "" {
			tx.Description = o.Description
		}
		switch models.TransactionType(o.Type) {
		case models.TypeExpense, models.TypeIncome, models.TypeTransfer:
			tx.Type = models.TransactionType(o.Type)
		}
	}
}

func buildPreview(stmt *models.Statement, parserName string) *models.ImportPreview {
	bank := parserName
	if idx := strings.Index(parserName, "/"); idx != -1 {
		bank = parserName[:idx]
	}

	currency := "CRC"
	if len(stmt.Transactions) > 0 {
		currency = stmt.Transactions[0].Currency
	}

	var periodStart, periodEnd string
	if len(stmt.Transactions) > 0 {
		periodStart = stmt.Transactions[0].Date.Format("2006-01-02")
		periodEnd = stmt.Transactions[len(stmt.Transactions)-1].Date.Format("2006-01-02")
	}

	txs := make([]models.Transaction, len(stmt.Transactions))
	copy(txs, stmt.Transactions)

	return &models.ImportPreview{
		AccountNumber:    stmt.AccountNumber,
		Bank:             strings.ToUpper(bank),
		Currency:         currency,
		TransactionCount: len(stmt.Transactions),
		PeriodStart:      periodStart,
		PeriodEnd:        periodEnd,
		Transactions:     txs,
	}
}
