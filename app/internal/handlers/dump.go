package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"life-base-api/app/internal/pdf"
)

func NewDumpHandler() *DumpHandler { return &DumpHandler{} }

func (h *DumpHandler) Handle(inputDir, outputDir string, verbose bool) error {
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	files, err := os.ReadDir(inputDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() || strings.ToLower(filepath.Ext(file.Name())) != ".pdf" {
			continue
		}

		inputPath := filepath.Join(inputDir, file.Name())
		outputPath := filepath.Join(outputDir, file.Name()+".txt")

		if err := pdf.ExtractText(inputPath, outputPath, verbose); err != nil {
			return fmt.Errorf("extract %s: %w", file.Name(), err)
		}

		fmt.Printf("Dumped: %s → %s\n", file.Name(), outputPath)
	}

	return nil
}
