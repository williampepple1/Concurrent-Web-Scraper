package scraper

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/williampepple1/concurrent-web-scraper/internal/config"
	"github.com/williampepple1/concurrent-web-scraper/internal/extraction"
	"github.com/williampepple1/concurrent-web-scraper/internal/proxy"
	"github.com/williampepple1/concurrent-web-scraper/pkg/models"
)

// HTTPScraper implements HTTP-based scraping
type HTTPScraper struct {
	Config    *config.AppConfig
	Extractor *extraction.Extractor
	Proxy     *proxy.Manager
}

// NewHTTPScraper creates a new HTTP scraper
func NewHTTPScraper(config *config.AppConfig) *HTTPScraper {
	return &HTTPScraper{
		Config:    config,
		Extractor: extraction.NewExtractor(&config.Extraction),
		Proxy:     proxy.NewManager(&config.Proxies),
	}
}

// Fetch fetches the content of a URL and returns a Result
func (s *HTTPScraper) Fetch(url string) models.Result {
	start := time.Now()
	var retries int
	var lastErr error
	var statusCode int
	var proxyUsed string

	// Create a transport with proxy support
	transport := &http.Transport{}

	// Add proxy if enabled
	if s.Config.Proxies.Enabled && len(s.Config.Proxies.List) > 0 {
		var err error
		proxyUsed, err = s.Proxy.ApplyToTransport(transport)
		if err != nil {
			return models.Result{
				URL:       url,
				Err:       fmt.Sprintf("Error applying proxy: %v", err),
				Duration:  time.Since(start),
				Timestamp: time.Now(),
			}
		}
	}

	// Create a client with the transport and timeout
	client := &http.Client{
		Transport: transport,
		Timeout:   s.Config.Scraper.Timeout,
	}

	for retries <= s.Config.Scraper.MaxRetries {
		if retries > 0 {
			// Wait before retrying
			retryWait := s.Config.Scraper.RetryDelay * time.Duration(retries)
			fmt.Printf("Retrying %s after %v (attempt %d/%d)\n", url, retryWait, retries, s.Config.Scraper.MaxRetries)
			time.Sleep(retryWait)

			// Rotate proxy if enabled
			if s.Config.Proxies.Enabled && s.Config.Proxies.Rotate && len(s.Config.Proxies.List) > 1 {
				proxyUsed, _ = s.Proxy.ApplyToTransport(transport)
			}
		}

		// Create a new request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			lastErr = err
			retries++
			continue
		}

		// Set a random user agent if available
		if len(s.Config.Scraper.UserAgents) > 0 {
			userAgent := s.Config.Scraper.UserAgents[rand.Intn(len(s.Config.Scraper.UserAgents))]
			req.Header.Set("User-Agent", userAgent)
		}

		// Make the request
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			retries++
			continue
		}

		// Ensure the response body is closed
		defer resp.Body.Close()
		statusCode = resp.StatusCode

		// Check for non-200 status codes
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
			retries++
			continue
		}

		// Create a goquery document for HTML parsing
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			lastErr = err
			retries++
			continue
		}

		// Extract data using CSS selectors, XPath, and regex
		extracted := s.Extractor.Extract(doc)

		// Get the HTML content
		html, err := doc.Html()
		if err != nil {
			lastErr = err
			retries++
			continue
		}

		// Success! Return the result
		return models.Result{
			URL:        url,
			Content:    html,
			Extracted:  extracted,
			Err:        "",
			Duration:   time.Since(start),
			Retries:    retries,
			StatusCode: statusCode,
			Timestamp:  time.Now(),
			ProxyUsed:  proxyUsed,
			JSRendered: false,
		}
	}

	// If we get here, all retries failed
	return models.Result{
		URL:        url,
		Content:    "",
		Extracted:  nil,
		Err:        lastErr.Error(),
		Duration:   time.Since(start),
		Retries:    retries,
		StatusCode: statusCode,
		Timestamp:  time.Now(),
		ProxyUsed:  proxyUsed,
		JSRendered: false,
	}
}
