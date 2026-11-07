package handlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"life-base-api/app/internal/parser"
	_ "life-base-api/app/internal/parser/parsers/bac" // registers the BAC statement parsers
	"life-base-api/app/internal/pdf"
)

func NewExtractHandler(importer Importer) *ExtractHandler {
	return &ExtractHandler{importer: importer}
}

func (h *ExtractHandler) Handle(ctx context.Context, inputDir, outputDir string, dryRun, verbose bool) error {
	tempDir, err := os.MkdirTemp("", "pdf-extract-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	if verbose {
		fmt.Printf("Registered parsers: %v\n", parser.List())
	}

	if err := pdf.ProcessPDFs(inputDir, tempDir, verbose); err != nil {
		return fmt.Errorf("extract PDFs: %w", err)
	}

	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	files, err := os.ReadDir(tempDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".txt" {
			continue
		}
		if err := h.processText(ctx, tempDir, outputDir, file.Name(), dryRun, verbose); err != nil {
			return err
		}
	}

	return nil
}

// processText parses one extracted statement text file, then either writes
// its transactions to outputDir (dry run) or imports them.
func (h *ExtractHandler) processText(ctx context.Context, tempDir, outputDir, name string, dryRun, verbose bool) error {
	baseName := strings.TrimSuffix(name, ".txt")
	raw, err := os.ReadFile(filepath.Join(tempDir, name)) //nolint:gosec // tempDir is process-owned (os.MkdirTemp), name enumerated from it; CLI-only, not HTTP-reachable
	if err != nil {
		return err
	}
	text := string(raw)

	p, err := parser.Detect(text)
	if err != nil {
		return fmt.Errorf("%s: %w", baseName, err)
	}
	if verbose {
		fmt.Printf("Detected parser %q for %s\n", p.Name(), baseName)
	}

	stmt, err := p.Parse(text)
	if err != nil {
		return fmt.Errorf("parse %s: %w", baseName, err)
	}
	stmt.SourceFile = name
	if verbose {
		fmt.Printf("  Account: %s (short: %s)\n", stmt.AccountNumber, stmt.ShortNumber)
	}

	if dryRun {
		outputPath := filepath.Join(outputDir, baseName+".transactions")
		if err := parser.WriteTransactions(outputPath, stmt.Transactions); err != nil {
			return fmt.Errorf("write %s: %w", baseName, err)
		}
	} else if err := h.importer.Import(ctx, stmt, p.Name()); err != nil {
		return fmt.Errorf("import %s: %w", baseName, err)
	}

	fmt.Printf("%s: %d transactions\n", baseName, len(stmt.Transactions))
	return nil
}
