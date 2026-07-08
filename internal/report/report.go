// Package report aggregates the outcome of a load test run and renders a
// human-readable summary to the console.
package report

import (
	"fmt"
	"io"
	"sort"
	"sync"
	"time"
)

// Report collects the results produced during a load test execution.
// It is safe for concurrent use.
type Report struct {
	mu sync.Mutex

	totalRequests int
	statusCounts  map[int]int // HTTP status code -> count
	networkErrors int         // requests that never got an HTTP response
	duration      time.Duration
}

// New creates an empty Report.
func New() *Report {
	return &Report{
		statusCounts: make(map[int]int),
	}
}

// AddResult records the outcome of a single request. statusCode should be 0
// when the request failed before an HTTP response was received (e.g.
// connection refused, timeout, DNS failure).
func (r *Report) AddResult(statusCode int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.totalRequests++
	if statusCode == 0 {
		r.networkErrors++
		return
	}
	r.statusCounts[statusCode]++
}

// SetDuration records the total wall-clock time the load test took to run.
func (r *Report) SetDuration(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.duration = d
}

// Print writes a formatted summary of the report to w.
func (r *Report) Print(w io.Writer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	fmt.Fprintln(w, "==================================================")
	fmt.Fprintln(w, "               STRESS TEST REPORT")
	fmt.Fprintln(w, "==================================================")
	fmt.Fprintf(w, "Total execution time : %s\n", formatDuration(r.duration))
	fmt.Fprintf(w, "Total requests made  : %d\n", r.totalRequests)
	fmt.Fprintf(w, "Requests with status 200 OK : %d\n", r.statusCounts[200])

	fmt.Fprintln(w, "--------------------------------------------------")
	fmt.Fprintln(w, "Distribution of other HTTP status codes:")

	otherCodes := make([]int, 0, len(r.statusCounts))
	for code := range r.statusCounts {
		if code == 200 {
			continue
		}
		otherCodes = append(otherCodes, code)
	}
	sort.Ints(otherCodes)

	if len(otherCodes) == 0 && r.networkErrors == 0 {
		fmt.Fprintln(w, "  (none)")
	}

	for _, code := range otherCodes {
		fmt.Fprintf(w, "  HTTP %d: %d\n", code, r.statusCounts[code])
	}

	if r.networkErrors > 0 {
		fmt.Fprintf(w, "  Network/connection errors (no response): %d\n", r.networkErrors)
	}

	fmt.Fprintln(w, "==================================================")
}

func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%s (%.3fs)", d.Round(time.Millisecond), d.Seconds())
}
