package scraper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/williampepple1/concurrent-web-scraper/internal/config"
	"github.com/williampepple1/concurrent-web-scraper/internal/extraction"
	"github.com/williampepple1/concurrent-web-scraper/pkg/models"
)

// BrowserScraper implements browser-based scraping
type BrowserScraper struct {
	Config    *config.AppConfig
	Extractor *extraction.Extractor
}

// NewBrowserScraper creates a new browser scraper
func NewBrowserScraper(config *config.AppConfig) *BrowserScraper {
	return &BrowserScraper{
		Config:    config,
		Extractor: extraction.NewExtractor(&config.Extraction),
	}
}

// Fetch fetches a URL using a headless browser for JavaScript rendering
func (s *BrowserScraper) Fetch(url string) models.Result {
	start := time.Now()

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.Scraper.Timeout)
	defer cancel()

	// Configure browser options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", s.Config.Browser.Headless),
		chromedp.UserAgent(s.Config.Browser.UserAgent),
	)

	// Create a new ExecAllocator
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	// Create a new browser context
	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Create a channel to capture errors
	errChan := make(chan error, 1)
	var html string
	var screenshot []byte
	var statusCode int

	// Run the browser tasks
	go func() {
		tasks := []chromedp.Action{
			chromedp.Navigate(url),
			chromedp.Sleep(s.Config.Browser.WaitTime),
			chromedp.OuterHTML("html", &html),
		}

		// Add screenshot task if enabled
		if s.Config.Browser.Screenshot {
			tasks = append(tasks, chromedp.CaptureScreenshot(&screenshot))
		}

		// Run the tasks
		errChan <- chromedp.Run(browserCtx, tasks...)
	}()

	// Wait for completion or timeout
	select {
	case err := <-errChan:
		if err != nil {
			return models.Result{
				URL:        url,
				Err:        err.Error(),
				Duration:   time.Since(start),
				Timestamp:  time.Now(),
				JSRendered: true,
			}
		}
	case <-ctx.Done():
		return models.Result{
			URL:        url,
			Err:        "browser timeout",
			Duration:   time.Since(start),
			Timestamp:  time.Now(),
			JSRendered: true,
		}
	}

	// Create a goquery document from the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return models.Result{
			URL:        url,
			Content:    html,
			Err:        err.Error(),
			Duration:   time.Since(start),
			Timestamp:  time.Now(),
			JSRendered: true,
		}
	}

	// Extract data
	extracted := s.Extractor.Extract(doc)

	// Save screenshot if enabled
	var screenshotPath string
	if s.Config.Browser.Screenshot && len(screenshot) > 0 {
		// Create screenshot directory if it doesn't exist
		if err := os.MkdirAll(s.Config.Browser.ScreenshotDir, 0755); err == nil {
			// Generate a filename based on the URL
			filename := fmt.Sprintf("%d.png", time.Now().UnixNano())
			screenshotPath = filepath.Join(s.Config.Browser.ScreenshotDir, filename)

			// Save the screenshot
			if err := os.WriteFile(screenshotPath, screenshot, 0644); err != nil {
				fmt.Printf("Error saving screenshot: %v\n", err)
			}
		}
	}

	return models.Result{
		URL:        url,
		Content:    html,
		Extracted:  extracted,
		Err:        "",
		Duration:   time.Since(start),
		StatusCode: statusCode,
		Timestamp:  time.Now(),
		Screenshot: screenshotPath,
		JSRendered: true,
	}
}
