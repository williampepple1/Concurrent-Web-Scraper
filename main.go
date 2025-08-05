package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

// Config holds the configuration for the scraper
type Config struct {
	MaxRetries     int
	RetryDelay     time.Duration
	RateLimitDelay time.Duration
	UserAgents     []string
}

// Result represents the result of scraping a URL
type Result struct {
	URL      string
	Content  string
	Err      error
	Duration time.Duration
	Retries  int
}

// fetchURL fetches the content of a URL and returns a Result
func fetchURL(url string, config Config) Result {
	start := time.Now()
	var retries int
	var lastErr error

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

		// Check for non-200 status codes
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
			retries++
			continue
		}

		// Read the response body
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			retries++
			continue
		}

		// Success! Return the result
		return Result{
			URL:      url,
			Content:  string(body),
			Err:      nil,
			Duration: time.Since(start),
			Retries:  retries,
		}
	}

	// If we get here, all retries failed
	return Result{
		URL:      url,
		Content:  "",
		Err:      lastErr,
		Duration: time.Since(start),
		Retries:  retries,
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

func main() {
	fmt.Println("Concurrent Web Scraper Starting...")

	// Configure the scraper
	config := Config{
		MaxRetries:     3,
		RetryDelay:     2 * time.Second,
		RateLimitDelay: 1 * time.Second,
		UserAgents: []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36",
		},
	}

	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Sample URLs to scrape
	urls := []string{
		"https://www.google.com",
		"https://www.github.com",
		"https://www.golang.org",
		"https://www.wikipedia.org",
		"https://www.reddit.com",
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
	numWorkers := 3 // Number of concurrent workers
	for w := 1; w <= numWorkers; w++ {
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

	// Collect and print results
	successCount := 0
	failureCount := 0

	for result := range results {
		if result.Err != nil {
			fmt.Printf("Error fetching %s: %v (after %d retries)\n", result.URL, result.Err, result.Retries)
			failureCount++
			continue
		}

		fmt.Printf("Successfully fetched %s in %v (retries: %d). Content length: %d bytes\n",
			result.URL, result.Duration, result.Retries, len(result.Content))
		successCount++
	}

	fmt.Printf("All URLs have been processed. Success: %d, Failures: %d\n", successCount, failureCount)
}
