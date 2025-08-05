package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"gopkg.in/yaml.v3"
)

// AppConfig holds the complete application configuration
type AppConfig struct {
	Scraper    ScraperConfig    `yaml:"scraper"`
	IO         IOConfig         `yaml:"io"`
	Extraction ExtractionConfig `yaml:"extraction"`
	Proxies    ProxyConfig      `yaml:"proxies"`
	Browser    BrowserConfig    `yaml:"browser"`
}

// ScraperConfig holds the scraper configuration
type ScraperConfig struct {
	Workers     int           `yaml:"workers"`
	RateLimit   time.Duration `yaml:"rate_limit"`
	MaxRetries  int           `yaml:"max_retries"`
	RetryDelay  time.Duration `yaml:"retry_delay"`
	Timeout     time.Duration `yaml:"timeout"`
	UserAgents  []string      `yaml:"user_agents,omitempty"`
}

// IOConfig holds the input/output configuration
type IOConfig struct {
	InputFile    string `yaml:"input_file"`
	OutputFile   string `yaml:"output_file"`
	OutputFormat string `yaml:"output_format"`
}

// ExtractionConfig holds the data extraction configuration
type ExtractionConfig struct {
	Selectors map[string]string `yaml:"selectors"`
	XPath     map[string]string `yaml:"xpath"`
	Regex     map[string]string `yaml:"regex"`
}

// ProxyConfig holds the proxy configuration
type ProxyConfig struct {
	Enabled bool     `yaml:"enabled"`
	Rotate  bool     `yaml:"rotate"`
	List    []string `yaml:"list"`
	Auth    struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"auth"`
}

// BrowserConfig holds the browser configuration for JavaScript rendering
type BrowserConfig struct {
	Enabled       bool          `yaml:"enabled"`
	Headless      bool          `yaml:"headless"`
	UserAgent     string        `yaml:"user_agent"`
	WaitTime      time.Duration `yaml:"wait_time"`
	Screenshot    bool          `yaml:"screenshot"`
	ScreenshotDir string        `yaml:"screenshot_dir"`
}

// Result represents the result of scraping a URL
type Result struct {
	URL         string                 `json:"url"`
	Content     string                 `json:"content,omitempty"`
	Extracted   map[string]interface{} `json:"extracted,omitempty"`
	Err         string                 `json:"error,omitempty"`
	Duration    time.Duration          `json:"duration"`
	Retries     int                    `json:"retries"`
	StatusCode  int                    `json:"status_code,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Screenshot  string                 `json:"screenshot,omitempty"`
	JSRendered  bool                   `json:"js_rendered,omitempty"`
	ProxyUsed   string                 `json:"proxy_used,omitempty"`
}

// loadConfig loads the configuration from a YAML file
func loadConfig(filename string) (*AppConfig, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Set default user agents if none provided
	if len(config.Scraper.UserAgents) == 0 {
		config.Scraper.UserAgents = []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36",
		}
	}

	return &config, nil
}

// getProxyURL returns a proxy URL from the configuration
func getProxyURL(config *AppConfig) (*url.URL, error) {
	if !config.Proxies.Enabled || len(config.Proxies.List) == 0 {
		return nil, nil
	}

	// Select a proxy
	proxyStr := config.Proxies.List[0]
	if config.Proxies.Rotate && len(config.Proxies.List) > 1 {
		proxyStr = config.Proxies.List[rand.Intn(len(config.Proxies.List))]
	}

	// Parse the proxy URL
	proxyURL, err := url.Parse(proxyStr)
	if err != nil {
		return nil, err
	}

	// Add authentication if provided
	if config.Proxies.Auth.Username != "" && config.Proxies.Auth.Password != "" {
		proxyURL.User = url.UserPassword(config.Proxies.Auth.Username, config.Proxies.Auth.Password)
	}

	return proxyURL, nil
}

// fetchURL fetches the content of a URL and returns a Result
func fetchURL(url string, config *AppConfig) Result {
	start := time.Now()
	var retries int
	var lastErr error
	var statusCode int
	var proxyUsed string

	// Check if we should use browser-based scraping
	if config.Browser.Enabled {
		return fetchWithBrowser(url, config)
	}

	// Create a transport with proxy support
	transport := &http.Transport{}

	// Add proxy if enabled
	if config.Proxies.Enabled && len(config.Proxies.List) > 0 {
		proxyURL, err := getProxyURL(config)
		if err != nil {
			return Result{
				URL:       url,
				Err:       fmt.Sprintf("Error parsing proxy URL: %v", err),
				Duration:  time.Since(start),
				Timestamp: time.Now(),
			}
		}

		if proxyURL != nil {
			transport.Proxy = http.ProxyURL(proxyURL)
			proxyUsed = proxyURL.String()
		}
	}

	// Create a client with the transport and timeout
	client := &http.Client{
		Transport: transport,
		Timeout:   config.Scraper.Timeout,
	}

	for retries <= config.Scraper.MaxRetries {
		if retries > 0 {
			// Wait before retrying
			retryWait := config.Scraper.RetryDelay * time.Duration(retries)
			fmt.Printf("Retrying %s after %v (attempt %d/%d)\n", url, retryWait, retries, config.Scraper.MaxRetries)
			time.Sleep(retryWait)

			// Rotate proxy if enabled
			if config.Proxies.Enabled && config.Proxies.Rotate && len(config.Proxies.List) > 1 {
				proxyURL, err := getProxyURL(config)
				if err == nil && proxyURL != nil {
					transport.Proxy = http.ProxyURL(proxyURL)
					proxyUsed = proxyURL.String()
				}
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
		if len(config.Scraper.UserAgents) > 0 {
			userAgent := config.Scraper.UserAgents[rand.Intn(len(config.Scraper.UserAgents))]
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
		extracted := extractData(doc, resp.Body, config)

		// Get the HTML content
		html, err := doc.Html()
		if err != nil {
			lastErr = err
			retries++
			continue
		}

		// Success! Return the result
		return Result{
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
	return Result{
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

// fetchWithBrowser fetches a URL using a headless browser for JavaScript rendering
func fetchWithBrowser(url string, config *AppConfig) Result {
	start := time.Now()

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), config.Scraper.Timeout)
	defer cancel()

	// Configure browser options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", config.Browser.Headless),
		chromedp.UserAgent(config.Browser.UserAgent),
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
			chromedp.Sleep(config.Browser.WaitTime),
			chromedp.OuterHTML("html", &html),
		}

		// Add screenshot task if enabled
		if config.Browser.Screenshot {
			tasks = append(tasks, chromedp.CaptureScreenshot(&screenshot))
		}

		// Run the tasks
		errChan <- chromedp.Run(browserCtx, tasks...)
	}()

	// Wait for completion or timeout
	select {
		case err := <-errChan:
			if err != nil {
				return Result{
					URL:        url,
					Err:        err.Error(),
					Duration:   time.Since(start),
					Timestamp:  time.Now(),
					JSRendered: true,
				}
			}
		case <-ctx.Done():
			return Result{
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
		return Result{
			URL:        url,
			Content:    html,
			Err:        err.Error(),
			Duration:   time.Since(start),
			Timestamp:  time.Now(),
			JSRendered: true,
		}
	}

	// Extract data
	extracted := make(map[string]interface{})
	for name, selector := range config.Extraction.Selectors {
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

	// Save screenshot if enabled
	var screenshotPath string
	if config.Browser.Screenshot && len(screenshot) > 0 {
		// Create screenshot directory if it doesn't exist
		if err := os.MkdirAll(config.Browser.ScreenshotDir, 0755); err == nil {
			// Generate a filename based on the URL
			filename := fmt.Sprintf("%d.png", time.Now().UnixNano())
			screenshotPath = filepath.Join(config.Browser.ScreenshotDir, filename)

			// Save the screenshot
			if err := ioutil.WriteFile(screenshotPath, screenshot, 0644); err != nil {
				fmt.Printf("Error saving screenshot: %v\n", err)
			}
		}
	}

	return Result{
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

// extractData extracts data from HTML using CSS selectors, XPath, and regex
func extractData(doc *goquery.Document, body interface{}, config *AppConfig) map[string]interface{} {
	extracted := make(map[string]interface{})

	// Extract data using CSS selectors
	for name, selector := range config.Extraction.Selectors {
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

	// Extract data using XPath selectors (not directly supported by goquery, but we can simulate it)
	// Note: This is a simplified implementation, as goquery doesn't directly support XPath
	// For full XPath support, consider using github.com/antchfx/htmlquery

	// Extract data using regex
	html, _ := doc.Html()
	for name, pattern := range config.Extraction.Regex {
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

	return extracted
}

// worker processes URLs from the jobs channel and sends results to the results channel
func worker(id int, jobs <-chan string, results chan<- Result, config *AppConfig, rateLimiter *time.Ticker, wg *sync.WaitGroup) {
	defer wg.Done()

	for url := range jobs {
		// Wait for rate limiter
		<-rateLimiter.C

		fmt.Printf("Worker %d processing URL: %s\n", id, url)
		result := fetchURL(url, config)
		results <- result
	}
}

// readURLsFromFile reads URLs from a file, one URL per line
func readURLsFromFile(filename string) ([]string, error) {
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

// saveResultsToFile saves the results to a file in the specified format
func saveResultsToFile(results []Result, filename string, format string) error {
	switch format {
	case "json":
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return err
		}
		return ioutil.WriteFile(filename, data, 0644)

	case "csv":
		// Implement CSV output if needed
		return fmt.Errorf("CSV output not implemented yet")

	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func main() {
	// Define command-line flags
	configFile := flag.String("config", "", "Path to configuration file (YAML)")
	inputFile := flag.String("input", "", "File containing URLs to scrape (one per line)")
	outputFile := flag.String("output", "results.json", "File to save results to")
	numWorkers := flag.Int("workers", 3, "Number of concurrent workers")
	rateLimitDelay := flag.Duration("rate-limit", 1*time.Second, "Delay between requests")
	maxRetries := flag.Int("retries", 3, "Maximum number of retries per URL")
	retryDelay := flag.Duration("retry-delay", 2*time.Second, "Base delay between retries")
	titleSelector := flag.String("title-selector", "title", "CSS selector for title extraction")
	headingSelector := flag.String("heading-selector", "h1", "CSS selector for heading extraction")
	enableProxy := flag.Bool("proxy", false, "Enable proxy support")
	enableBrowser := flag.Bool("browser", false, "Enable browser-based scraping")
	flag.Parse()

	fmt.Println("Concurrent Web Scraper Starting...")

	// Load configuration
	var config *AppConfig
	if *configFile != "" {
		// Load from config file
		var err error
		config, err = loadConfig(*configFile)
		if err != nil {
			log.Fatalf("Error loading configuration: %v", err)
		}
		fmt.Printf("Loaded configuration from %s\n", *configFile)
	} else {
		// Create default configuration
		config = &AppConfig{
			Scraper: ScraperConfig{
				Workers:    *numWorkers,
				RateLimit:  *rateLimitDelay,
				MaxRetries: *maxRetries,
				RetryDelay: *retryDelay,
				Timeout:    30 * time.Second,
				UserAgents: []string{
					"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
					"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
					"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36",
				},
			},
			IO: IOConfig{
				InputFile:    *inputFile,
				OutputFile:   *outputFile,
				OutputFormat: "json",
			},
			Extraction: ExtractionConfig{
				Selectors: map[string]string{
					"title":   *titleSelector,
					"heading": *headingSelector,
				},
				XPath: map[string]string{},
				Regex: map[string]string{},
			},
			Proxies: ProxyConfig{
				Enabled: *enableProxy,
				Rotate:  true,
				List:    []string{},
			},
			Browser: BrowserConfig{
				Enabled:       *enableBrowser,
				Headless:      true,
				UserAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
				WaitTime:      5 * time.Second,
				Screenshot:    false,
				ScreenshotDir: "screenshots",
			},
		}
		fmt.Println("Using default configuration (no config file provided)")
	}

	// Override config with command-line flags if provided
	if *inputFile != "" {
		config.IO.InputFile = *inputFile
	}
	if *outputFile != "results.json" {
		config.IO.OutputFile = *outputFile
	}

	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Get URLs to scrape
	var urls []string
	if config.IO.InputFile != "" {
		// Read URLs from file
		var err error
		urls, err = readURLsFromFile(config.IO.InputFile)
		if err != nil {
			log.Fatalf("Error reading URLs from file: %v", err)
		}
		fmt.Printf("Read %d URLs from %s\n", len(urls), config.IO.InputFile)
	} else {
		// Use default URLs
		urls = []string{
			"https://www.google.com",
			"https://www.github.com",
			"https://www.golang.org",
			"https://www.wikipedia.org",
			"https://www.reddit.com",
		}
		fmt.Println("Using default URLs (no input file provided)")
	}

	if len(urls) == 0 {
		log.Fatal("No URLs to scrape")
	}

	// Create channels for jobs and results
	jobs := make(chan string, len(urls))
	results := make(chan Result, len(urls))

	// Create a wait group to wait for all workers to finish
	var wg sync.WaitGroup

	// Create a rate limiter
	rateLimiter := time.NewTicker(config.Scraper.RateLimit)
	defer rateLimiter.Stop()

	// Start workers
	for w := 1; w <= config.Scraper.Workers; w++ {
		wg.Add(1)
		go worker(w, jobs, results, config, rateLimiter, &wg)
	}

	// Send jobs to workers
	for _, url := range urls {
		jobs <- url
	}
	close(jobs) // Close the jobs channel to signal workers that no more jobs are coming

	// Start a goroutine to close the results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allResults []Result
	successCount := 0
	failureCount := 0

	for result := range results {
		allResults = append(allResults, result)

		if result.Err != "" {
			fmt.Printf("Error fetching %s: %s (after %d retries)\n", result.URL, result.Err, result.Retries)
			failureCount++
			continue
		}

		fmt.Printf("Successfully fetched %s in %v (retries: %d)\n", result.URL, result.Duration, result.Retries)
		if len(result.Extracted) > 0 {
			fmt.Println("Extracted data:")
			for name, value := range result.Extracted {
				fmt.Printf("  %s: %v\n", name, value)
			}
		}

		if result.Screenshot != "" {
			fmt.Printf("  Screenshot saved to: %s\n", result.Screenshot)
		}

		if result.ProxyUsed != "" {
			fmt.Printf("  Proxy used: %s\n", result.ProxyUsed)
		}

		successCount++
	}

	// Save results to file
	if err := saveResultsToFile(allResults, config.IO.OutputFile, config.IO.OutputFormat); err != nil {
		log.Fatalf("Error saving results to file: %v", err)
	}

	fmt.Printf("All URLs have been processed. Success: %d, Failures: %d\n", successCount, failureCount)
	fmt.Printf("Results saved to %s\n", config.IO.OutputFile)
}
