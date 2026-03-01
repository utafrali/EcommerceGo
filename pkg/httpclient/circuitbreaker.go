package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sony/gobreaker/v2"
)

// CircuitBreakerConfig holds configuration for the circuit breaker.
type CircuitBreakerConfig struct {
	// Name identifies this breaker (used in metrics and logs).
	Name string

	// MaxRequests is the maximum number of requests allowed in the half-open state.
	// 0 means 1 request is allowed.
	MaxRequests uint32

	// Interval is the cyclic period of the closed state for clearing internal counts.
	// 0 means internal counts are never cleared during the closed state.
	Interval time.Duration

	// Timeout is how long the breaker stays open before moving to half-open.
	Timeout time.Duration

	// FailureRatio is the ratio of failures to total requests that trips the breaker.
	// For example, 0.5 means trip when 50% of requests fail.
	FailureRatio float64

	// MinRequests is the minimum number of requests needed before the failure ratio is evaluated.
	MinRequests uint32
}

// DefaultCircuitBreakerConfig returns sensible defaults for a circuit breaker.
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:         name,
		MaxRequests:  1,
		Interval:     60 * time.Second,
		Timeout:      30 * time.Second,
		FailureRatio: 0.5,
		MinRequests:  5,
	}
}

// FallbackFunc is invoked when the circuit breaker is open and a fallback is configured.
// It receives the original error (typically ErrCircuitOpen) and returns a substitute response.
type FallbackFunc func(ctx context.Context, err error) (*http.Response, error)

var (
	circuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Current state of the circuit breaker (0=closed, 1=half-open, 2=open)",
		},
		[]string{"name"},
	)

	circuitBreakerFallbackTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_fallback_invoked_total",
			Help: "Total number of times the circuit breaker fallback was invoked",
		},
		[]string{"name"},
	)
)

func init() {
	prometheus.MustRegister(circuitBreakerState)
	prometheus.MustRegister(circuitBreakerFallbackTotal)
}

// stateToFloat maps gobreaker states to prometheus gauge values.
func stateToFloat(state gobreaker.State) float64 {
	switch state {
	case gobreaker.StateClosed:
		return 0
	case gobreaker.StateHalfOpen:
		return 1
	case gobreaker.StateOpen:
		return 2
	default:
		return -1
	}
}

// CircuitBreakerClient wraps a Client with circuit breaker protection.
type CircuitBreakerClient struct {
	client   *Client
	breaker  *gobreaker.CircuitBreaker[*http.Response]
	logger   *slog.Logger
	fallback FallbackFunc
	name     string
}

// NewCircuitBreakerClient wraps an existing HTTP client with a circuit breaker.
func NewCircuitBreakerClient(client *Client, cbCfg CircuitBreakerConfig, logger *slog.Logger) *CircuitBreakerClient {
	settings := gobreaker.Settings{
		Name:        cbCfg.Name,
		MaxRequests: cbCfg.MaxRequests,
		Interval:    cbCfg.Interval,
		Timeout:     cbCfg.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < cbCfg.MinRequests {
				return false
			}
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return failureRatio >= cbCfg.FailureRatio
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Warn("circuit breaker state change",
				slog.String("breaker", name),
				slog.String("from", from.String()),
				slog.String("to", to.String()),
			)
			circuitBreakerState.WithLabelValues(name).Set(stateToFloat(to))
		},
	}

	cb := gobreaker.NewCircuitBreaker[*http.Response](settings)

	// Set initial state metric.
	circuitBreakerState.WithLabelValues(cbCfg.Name).Set(0)

	return &CircuitBreakerClient{
		client:  client,
		breaker: cb,
		logger:  logger,
		name:    cbCfg.Name,
	}
}

// WithFallback returns a copy of the CircuitBreakerClient with a fallback function
// that is invoked when the circuit breaker is open, instead of returning ErrCircuitOpen.
func (c *CircuitBreakerClient) WithFallback(fn FallbackFunc) *CircuitBreakerClient {
	cpy := *c
	cpy.fallback = fn
	return &cpy
}

// ErrCircuitOpen is returned when the circuit breaker is open and rejects the request.
var ErrCircuitOpen = gobreaker.ErrOpenState

// Do executes an HTTP request through the circuit breaker.
// If the circuit is open and a fallback is configured, the fallback is invoked instead
// of returning ErrCircuitOpen directly.
func (c *CircuitBreakerClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	resp, err := c.breaker.Execute(func() (*http.Response, error) {
		resp, err := c.client.Do(ctx, req)
		if err != nil {
			return nil, err
		}
		// Treat 5xx responses as failures for the circuit breaker.
		if resp.StatusCode >= 500 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				body = []byte{}
			}
			_ = resp.Body.Close()
			return nil, fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
		}
		return resp, nil
	})
	if err != nil && c.fallback != nil && errors.Is(err, ErrCircuitOpen) {
		circuitBreakerFallbackTotal.WithLabelValues(c.name).Inc()
		c.logger.WarnContext(ctx, "circuit breaker open, invoking fallback",
			slog.String("breaker", c.name),
		)
		return c.fallback(ctx, err)
	}
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Get performs an HTTP GET request through the circuit breaker.
func (c *CircuitBreakerClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create GET request: %w", err)
	}
	return c.Do(ctx, req)
}

// Post performs an HTTP POST request through the circuit breaker.
func (c *CircuitBreakerClient) Post(ctx context.Context, url string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("create POST request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(ctx, req)
}

// State returns the current state of the circuit breaker.
func (c *CircuitBreakerClient) State() gobreaker.State {
	return c.breaker.State()
}
