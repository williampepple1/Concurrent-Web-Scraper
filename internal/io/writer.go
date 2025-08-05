package io

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/williampepple1/concurrent-web-scraper/internal/config"
	"github.com/williampepple1/concurrent-web-scraper/pkg/models"
)

// ResultWriter writes results to various outputs
type ResultWriter struct {
	Config *config.IOConfig
}

// NewResultWriter creates a new result writer
func NewResultWriter(config *config.IOConfig) *ResultWriter {
	return &ResultWriter{
		Config: config,
	}
}

// SaveToFile saves the results to a file in the specified format
func (w *ResultWriter) SaveToFile(results []models.Result) error {
	switch w.Config.OutputFormat {
	case "json":
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(w.Config.OutputFile, data, 0644)

	case "csv":
		// Implement CSV output if needed
		return fmt.Errorf("CSV output not implemented yet")

	default:
		return fmt.Errorf("unsupported output format: %s", w.Config.OutputFormat)
	}
}
