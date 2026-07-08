// Package httpclient provides a thin wrapper around net/http for issuing
// the GET requests used by the load test.
package httpclient

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client performs HTTP GET requests against a fixed URL.
type Client struct {
	url        string
	httpClient *http.Client
}

// New creates a Client for the given target URL. timeout bounds how long a
// single request is allowed to take before it is considered failed.
func New(url string, timeout time.Duration) *Client {
	return &Client{
		url: url,
		// A nil CheckRedirect keeps net/http's default behavior of
		// following redirects (up to 10 hops), same as curl -L or a
		// browser. This is what makes, e.g., "http://google.com" (which
		// always 301s to https://www.google.com/) resolve to a real 200
		// instead of only ever reporting the first hop's 301.
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Get performs a single GET request and returns the resulting HTTP status
// code. The response body is fully drained and closed so the underlying
// connection can be reused by the client's transport.
func (c *Client) Get() (int, error) {
	resp, err := c.httpClient.Get(c.url)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Drain the body so the connection is returned to the pool.
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return resp.StatusCode, fmt.Errorf("failed reading response body: %w", err)
	}

	return resp.StatusCode, nil
}
