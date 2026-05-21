// Package httpquery is a thin shared helper for adapters that hit an HTTP API
// for search. Each adapter wraps it with its own URL builder and response parser.
package httpquery

import (
	"context"
	"io"
	"net/http"
)

// Client wraps an *http.Client with a convenience Get method.
type Client struct {
	HTTP *http.Client
}

// New returns a Client backed by c. If c is nil, http.DefaultClient is used.
func New(c *http.Client) *Client {
	if c == nil {
		c = http.DefaultClient
	}
	return &Client{HTTP: c}
}

// Get performs a GET request to url with a pkgr User-Agent and returns
// (body, statusCode, error).
func (c *Client) Get(ctx context.Context, url string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "pkgr")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}
