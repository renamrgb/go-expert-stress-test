// Package loadtest orchestrates concurrent execution of HTTP requests
// against a target URL, distributing exactly the requested number of calls
// across a fixed pool of workers.
package loadtest

import (
	"fmt"
	"sync"
	"time"

	"github.com/renamrgb/go-expert-stress-test/internal/httpclient"
	"github.com/renamrgb/go-expert-stress-test/internal/report"
)

// Config holds the parameters that control a load test run.
type Config struct {
	URL         string
	Requests    int
	Concurrency int
	Timeout     time.Duration
}

// Validate checks that the configuration is usable, returning a descriptive
// error otherwise.
func (c Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url must not be empty")
	}
	if c.Requests <= 0 {
		return fmt.Errorf("requests must be greater than zero, got %d", c.Requests)
	}
	if c.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be greater than zero, got %d", c.Concurrency)
	}
	return nil
}

// Run executes the load test described by cfg and returns the resulting
// report. Exactly cfg.Requests GET requests are performed, distributed
// across cfg.Concurrency worker goroutines via a shared job channel.
func Run(cfg Config) (*report.Report, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	client := httpclient.New(cfg.URL, cfg.Timeout)
	rpt := report.New()

	// jobs is pre-loaded with exactly cfg.Requests tokens, one per request
	// to be made, guaranteeing the exact total regardless of concurrency.
	jobs := make(chan struct{}, cfg.Requests)
	for range cfg.Requests {
		jobs <- struct{}{}
	}
	close(jobs)

	// Concurrency is bounded to the number of requests: spinning up more
	// workers than there is work to do would be wasteful.
	workers := min(cfg.Concurrency, cfg.Requests)

	var wg sync.WaitGroup
	wg.Add(workers)

	start := time.Now()
	for range workers {
		go func() {
			defer wg.Done()
			for range jobs {
				statusCode, err := client.Get()
				if err != nil {
					rpt.AddResult(0)
					continue
				}
				rpt.AddResult(statusCode)
			}
		}()
	}

	wg.Wait()
	rpt.SetDuration(time.Since(start))

	return rpt, nil
}
