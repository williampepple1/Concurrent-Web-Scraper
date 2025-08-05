package io

import (
	"bufio"
	"os"
	"strings"

	"github.com/williampepple1/concurrent-web-scraper/internal/config"
)

// URLReader reads URLs from various sources
type URLReader struct {
	Config *config.IOConfig
}

// NewURLReader creates a new URL reader
func NewURLReader(config *config.IOConfig) *URLReader {
	return &URLReader{
		Config: config,
	}
}

// ReadFromFile reads URLs from a file, one URL per line
func (r *URLReader) ReadFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url != "" && !strings.HasPrefix(url, "#") {
			urls = append(urls, url)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return urls, nil
}

// GetURLs returns URLs from the configured source or default URLs
func (r *URLReader) GetURLs() ([]string, error) {
	if r.Config.InputFile != "" {
		return r.ReadFromFile(r.Config.InputFile)
	}

	// Return default URLs
	return config.DefaultURLs, nil
}
