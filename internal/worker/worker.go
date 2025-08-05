package worker

import (
	"fmt"
	"sync"
	"time"

	"github.com/williampepple1/concurrent-web-scraper/internal/config"
	"github.com/williampepple1/concurrent-web-scraper/internal/scraper"
	"github.com/williampepple1/concurrent-web-scraper/pkg/models"
)

// Pool manages a pool of worker goroutines
type Pool struct {
	Config    *config.AppConfig
	Scraper   scraper.Scraper
	Jobs      chan string
	Results   chan models.Result
	WaitGroup *sync.WaitGroup
}

// NewPool creates a new worker pool
func NewPool(config *config.AppConfig, urls []string) *Pool {
	jobs := make(chan string, len(urls))
	results := make(chan models.Result, len(urls))
	wg := &sync.WaitGroup{}

	return &Pool{
		Config:    config,
		Scraper:   scraper.New(config),
		Jobs:      jobs,
		Results:   results,
		WaitGroup: wg,
	}
}

// Start starts the worker pool
func (p *Pool) Start() {
	// Create a rate limiter
	rateLimiter := time.NewTicker(p.Config.Scraper.RateLimit)
	defer rateLimiter.Stop()

	// Start workers
	for w := 1; w <= p.Config.Scraper.Workers; w++ {
		p.WaitGroup.Add(1)
		go p.worker(w, rateLimiter)
	}

	// Start a goroutine to close the results channel when all workers are done
	go func() {
		p.WaitGroup.Wait()
		close(p.Results)
	}()
}

// worker processes URLs from the jobs channel and sends results to the results channel
func (p *Pool) worker(id int, rateLimiter *time.Ticker) {
	defer p.WaitGroup.Done()

	for url := range p.Jobs {
		// Wait for rate limiter
		<-rateLimiter.C

		fmt.Printf("Worker %d processing URL: %s\n", id, url)
		result := p.Scraper.Fetch(url)
		p.Results <- result
	}
}

// AddJobs adds URLs to the jobs channel
func (p *Pool) AddJobs(urls []string) {
	for _, url := range urls {
		p.Jobs <- url
	}
	close(p.Jobs) // Close the jobs channel to signal workers that no more jobs are coming
}
