package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// Result represents the result of scraping a URL
type Result struct {
	URL      string
	Content  string
	Err      error
	Duration time.Duration
}

// fetchURL fetches the content of a URL and returns a Result
func fetchURL(url string) Result {
	start := time.Now()

	resp, err := http.Get(url)
	if err != nil {
		return Result{URL: url, Err: err, Duration: time.Since(start)}
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Result{URL: url, Err: err, Duration: time.Since(start)}
	}

	return Result{
		URL:      url,
		Content:  string(body),
		Err:      nil,
		Duration: time.Since(start),
	}
}

// worker processes URLs from the jobs channel and sends results to the results channel
func worker(id int, jobs <-chan string, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	for url := range jobs {
		fmt.Printf("Worker %d processing URL: %s\n", id, url)
		result := fetchURL(url)
		results <- result
	}
}

func main() {
	fmt.Println("Concurrent Web Scraper Starting...")

	// Sample URLs to scrape
	urls := []string{
		"https://www.google.com",
		"https://www.github.com",
		"https://www.golang.org",
	}

	// Create channels for jobs and results
	jobs := make(chan string, len(urls))
	results := make(chan Result, len(urls))

	// Create a wait group to wait for all workers to finish
	var wg sync.WaitGroup

	// Start workers
	numWorkers := 3 // Number of concurrent workers
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go worker(w, jobs, results, &wg)
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
	for result := range results {
		if result.Err != nil {
			fmt.Printf("Error fetching %s: %v\n", result.URL, result.Err)
			continue
		}

		fmt.Printf("Successfully fetched %s in %v. Content length: %d bytes\n",
			result.URL, result.Duration, len(result.Content))
	}

	fmt.Println("All URLs have been processed.")
}
