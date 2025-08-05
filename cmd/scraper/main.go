package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/williampepple1/concurrent-web-scraper/internal/config"
	"github.com/williampepple1/concurrent-web-scraper/internal/io"
	"github.com/williampepple1/concurrent-web-scraper/internal/worker"
	"github.com/williampepple1/concurrent-web-scraper/pkg/models"
)

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

	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Load configuration
	var appConfig *config.AppConfig
	if *configFile != "" {
		// Load from config file
		var err error
		appConfig, err = config.Load(*configFile)
		if err != nil {
			log.Fatalf("Error loading configuration: %v", err)
		}
		fmt.Printf("Loaded configuration from %s\n", *configFile)
	} else {
		// Create default configuration
		appConfig = config.CreateDefault(
			*numWorkers,
			*rateLimitDelay,
			*retryDelay,
			*maxRetries,
			*inputFile,
			*outputFile,
			*titleSelector,
			*headingSelector,
			*enableProxy,
			*enableBrowser,
		)
		fmt.Println("Using default configuration (no config file provided)")
	}

	// Override config with command-line flags if provided
	if *inputFile != "" {
		appConfig.IO.InputFile = *inputFile
	}
	if *outputFile != "results.json" {
		appConfig.IO.OutputFile = *outputFile
	}

	// Get URLs to scrape
	urlReader := io.NewURLReader(&appConfig.IO)
	urls, err := urlReader.GetURLs()
	if err != nil {
		log.Fatalf("Error reading URLs: %v", err)
	}

	if len(urls) == 0 {
		log.Fatal("No URLs to scrape")
	}

	fmt.Printf("Preparing to scrape %d URLs with %d workers\n", len(urls), appConfig.Scraper.Workers)

	// Create worker pool
	pool := worker.NewPool(appConfig, urls)

	// Start the worker pool
	pool.Start()

	// Add jobs to the pool
	pool.AddJobs(urls)

	// Collect results
	var allResults []models.Result
	successCount := 0
	failureCount := 0

	for result := range pool.Results {
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
	resultWriter := io.NewResultWriter(&appConfig.IO)
	if err := resultWriter.SaveToFile(allResults); err != nil {
		log.Fatalf("Error saving results to file: %v", err)
	}

	fmt.Printf("All URLs have been processed. Success: %d, Failures: %d\n", successCount, failureCount)
	fmt.Printf("Results saved to %s\n", appConfig.IO.OutputFile)
}
