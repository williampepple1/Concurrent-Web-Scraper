package extraction

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/williampepple1/concurrent-web-scraper/internal/config"
)

// Extractor handles data extraction from HTML
type Extractor struct {
	Config *config.ExtractionConfig
}

// NewExtractor creates a new data extractor
func NewExtractor(config *config.ExtractionConfig) *Extractor {
	return &Extractor{
		Config: config,
	}
}

// Extract extracts data from HTML using CSS selectors, XPath, and regex
func (e *Extractor) Extract(doc *goquery.Document) map[string]interface{} {
	extracted := make(map[string]interface{})

	// Extract data using CSS selectors
	for name, selector := range e.Config.Selectors {
		values := []string{}
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			values = append(values, strings.TrimSpace(s.Text()))
		})

		if len(values) == 1 {
			extracted[name] = values[0]
		} else if len(values) > 1 {
			extracted[name] = values
		}
	}

	// Extract data using regex
	html, _ := doc.Html()
	for name, pattern := range e.Config.Regex {
		reg, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}

		matches := reg.FindAllString(html, -1)
		if len(matches) == 1 {
			extracted[name] = matches[0]
		} else if len(matches) > 1 {
			extracted[name] = matches
		}
	}

	// Note: XPath implementation would go here
	// For full XPath support, consider using github.com/antchfx/htmlquery

	return extracted
}
