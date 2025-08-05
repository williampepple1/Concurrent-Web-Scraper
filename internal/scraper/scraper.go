package scraper

import (
	"github.com/williampepple1/concurrent-web-scraper/internal/config"
	"github.com/williampepple1/concurrent-web-scraper/pkg/models"
)

// Scraper defines the interface for a web scraper
type Scraper interface {
	Fetch(url string) models.Result
}

// New creates a new scraper based on the configuration
func New(config *config.AppConfig) Scraper {
	if config.Browser.Enabled {
		return NewBrowserScraper(config)
	}
	return NewHTTPScraper(config)
}
