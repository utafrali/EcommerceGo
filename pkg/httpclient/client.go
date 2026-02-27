package httpclient

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// Config holds HTTP client configuration
type Config struct {
	Timeout         time.Duration
	MaxRetries      int
	RetryWaitMin    time.Duration
	RetryWaitMax    time.Duration
	MaxConnsPerHost int
}

// DefaultConfig returns sensible defaults for HTTP client
func DefaultConfig() Config {
	return Config{
		Timeout:         30 * time.Second,
		MaxRetries:      3,
		RetryWaitMin:    time.Second,
		RetryWaitMax:    5 * time.Second,
		MaxConnsPerHost: 100,
	}
}

// Client wraps http.Client with retry logic and better defaults
type Client struct {
	httpClient *http.Client
	config     Config
}

// New creates a new HTTP client with retry and connection pooling
func New(cfg Config) *Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   cfg.MaxConnsPerHost,
		MaxConnsPerHost:       cfg.MaxConnsPerHost,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   cfg.Timeout,
		},
		config: cfg,
	}
}

// Do executes HTTP request with retry logic
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			wait := c.config.RetryWaitMin * time.Duration(1<<uint(attempt-1))
			if wait > c.config.RetryWaitMax {
				wait = c.config.RetryWaitMax
			}

			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, err = c.httpClient.Do(req)
		if err != nil {
			// Retry on network errors
			if isRetryableError(err) && attempt < c.config.MaxRetries {
				continue
			}
			return nil, fmt.Errorf("http request failed after %d attempts: %w", attempt+1, err)
		}

		// Retry on 5xx errors (except 501 Not Implemented)
		if resp.StatusCode >= 500 && resp.StatusCode != 501 && attempt < c.config.MaxRetries {
			resp.Body.Close()
			continue
		}

		return resp, nil
	}

	return resp, err
}

// Get performs HTTP GET request with retry
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create GET request: %w", err)
	}
	return c.Do(ctx, req)
}

// Post performs HTTP POST request with retry
func (c *Client) Post(ctx context.Context, url string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("create POST request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(ctx, req)
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Network errors are retryable
	if _, ok := err.(net.Error); ok {
		return true
	}

	// Context errors are not retryable
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	return false
}
