package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Config holds the configuration for the scraper
type Config struct {
	MaxRetries     int
	RetryDelay     time.Duration
	RateLimitDelay time.Duration
	UserAgents     []string
	InputFile      string
	OutputFile     string
	Selectors      map[string]string
	NumWorkers     int
}

// Result represents the result of scraping a URL
type Result struct {
	URL        string            `json:"url"`
	Content    string            `json:"content,omitempty"`
	Extracted  map[string]string `json:"extracted,omitempty"`
	Err        string            `json:"error,omitempty"`
	Duration   time.Duration     `json:"duration"`
	Retries    int               `json:"retries"`
	StatusCode int               `json:"status_code,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
}

// fetchURL fetches the content of a URL and returns a Result
func fetchURL(url string, config Config) Result {
	start := time.Now()
	var retries int
	var lastErr error
	var statusCode int

	// Create a client with a timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for retries <= config.MaxRetries {
		if retries > 0 {
			// Wait before retrying
			retryWait := config.RetryDelay * time.Duration(retries)
			fmt.Printf("Retrying %s after %v (attempt %d/%d)\n", url, retryWait, retries, config.MaxRetries)
			time.Sleep(retryWait)
		}

		// Create a new request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			lastErr = err
			retries++
			continue
		}

		// Set a random user agent if available
		if len(config.UserAgents) > 0 {
			userAgent := config.UserAgents[rand.Intn(len(config.UserAgents))]
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

		// Extract data using selectors
		extracted := make(map[string]string)
		for name, selector := range config.Selectors {
			doc.Find(selector).Each(func(i int, s *goquery.Selection) {
				if i == 0 { // Just take the first match for simplicity
					extracted[name] = strings.TrimSpace(s.Text())
				}
			})
		}

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
	}
}

// worker processes URLs from the jobs channel and sends results to the results channel
func worker(id int, jobs <-chan string, results chan<- Result, config Config, rateLimiter *time.Ticker, wg *sync.WaitGroup) {
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

// saveResultsToFile saves the results to a JSON file
func saveResultsToFile(results []Result, filename string) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0644)
}

func main() {
	// Define command-line flags
	inputFile := flag.String("input", "", "File containing URLs to scrape (one per line)")
	outputFile := flag.String("output", "results.json", "File to save results to (JSON format)")
	numWorkers := flag.Int("workers", 3, "Number of concurrent workers")
	rateLimitDelay := flag.Duration("rate-limit", 1*time.Second, "Delay between requests")
	maxRetries := flag.Int("retries", 3, "Maximum number of retries per URL")
	retryDelay := flag.Duration("retry-delay", 2*time.Second, "Base delay between retries")
	titleSelector := flag.String("title-selector", "title", "CSS selector for title extraction")
	headingSelector := flag.String("heading-selector", "h1", "CSS selector for heading extraction")
	flag.Parse()

	fmt.Println("Concurrent Web Scraper Starting...")

	// Configure the scraper
	config := Config{
		MaxRetries:     *maxRetries,
		RetryDelay:     *retryDelay,
		RateLimitDelay: *rateLimitDelay,
		InputFile:      *inputFile,
		OutputFile:     *outputFile,
		NumWorkers:     *numWorkers,
		Selectors: map[string]string{
			"title":   *titleSelector,
			"heading": *headingSelector,
		},
		UserAgents: []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36",
		},
	}

	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Get URLs to scrape
	var urls []string
	if config.InputFile != "" {
		// Read URLs from file
		var err error
		urls, err = readURLsFromFile(config.InputFile)
		if err != nil {
			log.Fatalf("Error reading URLs from file: %v", err)
		}
		fmt.Printf("Read %d URLs from %s\n", len(urls), config.InputFile)
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
	rateLimiter := time.NewTicker(config.RateLimitDelay)
	defer rateLimiter.Stop()

	// Start workers
	for w := 1; w <= config.NumWorkers; w++ {
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
				fmt.Printf("  %s: %s\n", name, value)
			}
		}
		successCount++
	}

	// Save results to file
	if err := saveResultsToFile(allResults, config.OutputFile); err != nil {
		log.Fatalf("Error saving results to file: %v", err)
	}

	fmt.Printf("All URLs have been processed. Success: %d, Failures: %d\n", successCount, failureCount)
	fmt.Printf("Results saved to %s\n", config.OutputFile)
}
