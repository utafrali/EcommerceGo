package httpclient

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func testCBConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:         name,
		MaxRequests:  1,
		Interval:     60 * time.Second,
		Timeout:      1 * time.Second, // Short for tests.
		FailureRatio: 0.5,
		MinRequests:  3,
	}
}

func TestCircuitBreaker_ClosedState_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := New(Config{Timeout: 5 * time.Second, MaxRetries: 0, MaxConnsPerHost: 10})
	cb := NewCircuitBreakerClient(client, testCBConfig("test-closed"), testLogger())

	resp, err := cb.Get(context.Background(), server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, gobreaker.StateClosed, cb.State())
}

func TestCircuitBreaker_TripsOnFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`error`))
	}))
	defer server.Close()

	cfg := testCBConfig("test-trip")
	cfg.MinRequests = 3
	cfg.FailureRatio = 0.5

	client := New(Config{Timeout: 5 * time.Second, MaxRetries: 0, MaxConnsPerHost: 10})
	cb := NewCircuitBreakerClient(client, cfg, testLogger())

	// Produce enough failures to trip the breaker.
	for i := 0; i < 3; i++ {
		_, err := cb.Get(context.Background(), server.URL)
		require.Error(t, err) // 500s are treated as errors by the CB.
	}

	// The breaker should now be open.
	assert.Equal(t, gobreaker.StateOpen, cb.State())

	// Subsequent requests should fail immediately with ErrCircuitOpen.
	_, err := cb.Get(context.Background(), server.URL)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCircuitOpen)
}

func TestCircuitBreaker_HalfOpenToClosedRecovery(t *testing.T) {
	var failing atomic.Bool
	failing.Store(true)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failing.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`error`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cfg := testCBConfig("test-recovery")
	cfg.MinRequests = 3
	cfg.FailureRatio = 0.5
	cfg.Timeout = 100 * time.Millisecond // Very short for test.

	client := New(Config{Timeout: 5 * time.Second, MaxRetries: 0, MaxConnsPerHost: 10})
	cb := NewCircuitBreakerClient(client, cfg, testLogger())

	// Trip the breaker.
	for i := 0; i < 3; i++ {
		_, _ = cb.Get(context.Background(), server.URL)
	}
	assert.Equal(t, gobreaker.StateOpen, cb.State())

	// Wait for the timeout to elapse so the breaker transitions to half-open.
	time.Sleep(150 * time.Millisecond)

	// Now make the server healthy.
	failing.Store(false)

	// The next request should succeed and transition to closed.
	resp, err := cb.Get(context.Background(), server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, gobreaker.StateClosed, cb.State())
}

func TestCircuitBreaker_4xxNotCountedAsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest) // 400
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer server.Close()

	cfg := testCBConfig("test-4xx")
	cfg.MinRequests = 3
	cfg.FailureRatio = 0.5

	client := New(Config{Timeout: 5 * time.Second, MaxRetries: 0, MaxConnsPerHost: 10})
	cb := NewCircuitBreakerClient(client, cfg, testLogger())

	// 4xx responses should NOT trip the breaker.
	for i := 0; i < 5; i++ {
		resp, err := cb.Get(context.Background(), server.URL)
		require.NoError(t, err)
		resp.Body.Close()
	}

	// Breaker should still be closed.
	assert.Equal(t, gobreaker.StateClosed, cb.State())
}

func TestCircuitBreaker_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"123"}`))
	}))
	defer server.Close()

	client := New(Config{Timeout: 5 * time.Second, MaxRetries: 0, MaxConnsPerHost: 10})
	cb := NewCircuitBreakerClient(client, testCBConfig("test-post"), testLogger())

	resp, err := cb.Post(context.Background(), server.URL, "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestCircuitBreaker_DefaultConfig(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig("test-defaults")
	assert.Equal(t, "test-defaults", cfg.Name)
	assert.Equal(t, uint32(1), cfg.MaxRequests)
	assert.Equal(t, 60*time.Second, cfg.Interval)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.Equal(t, 0.5, cfg.FailureRatio)
	assert.Equal(t, uint32(5), cfg.MinRequests)
}

func TestCircuitBreaker_OpenStateRejectsRequests(t *testing.T) {
	var reqCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := testCBConfig("test-open-reject")
	cfg.MinRequests = 3
	cfg.Timeout = 5 * time.Second // Long so it stays open during test.

	client := New(Config{Timeout: 5 * time.Second, MaxRetries: 0, MaxConnsPerHost: 10})
	cb := NewCircuitBreakerClient(client, cfg, testLogger())

	// Trip the breaker.
	for i := 0; i < 3; i++ {
		_, _ = cb.Get(context.Background(), server.URL)
	}
	assert.Equal(t, gobreaker.StateOpen, cb.State())

	beforeCount := reqCount.Load()

	// These should be rejected without reaching the server.
	for i := 0; i < 5; i++ {
		_, err := cb.Get(context.Background(), server.URL)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCircuitOpen)
	}

	// No new requests should have reached the server.
	assert.Equal(t, beforeCount, reqCount.Load())
}

func TestCircuitBreaker_WithFallback_InvokedWhenOpen(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := testCBConfig("test-fallback")
	cfg.MinRequests = 3
	cfg.Timeout = 5 * time.Second

	client := New(Config{Timeout: 5 * time.Second, MaxRetries: 0, MaxConnsPerHost: 10})
	cb := NewCircuitBreakerClient(client, cfg, testLogger())

	var fallbackCalled atomic.Bool
	cbWithFallback := cb.WithFallback(func(ctx context.Context, err error) (*http.Response, error) {
		fallbackCalled.Store(true)
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Body:       http.NoBody,
		}, nil
	})

	// Trip the breaker.
	for i := 0; i < 3; i++ {
		_, _ = cbWithFallback.Get(context.Background(), server.URL)
	}
	assert.Equal(t, gobreaker.StateOpen, cbWithFallback.State())

	// Now the fallback should be invoked instead of returning ErrCircuitOpen.
	resp, err := cbWithFallback.Get(context.Background(), server.URL)
	require.NoError(t, err)
	assert.True(t, fallbackCalled.Load())
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestCircuitBreaker_WithFallback_NotInvokedWhenClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`ok`))
	}))
	defer server.Close()

	client := New(Config{Timeout: 5 * time.Second, MaxRetries: 0, MaxConnsPerHost: 10})
	cb := NewCircuitBreakerClient(client, testCBConfig("test-fallback-closed"), testLogger())

	var fallbackCalled atomic.Bool
	cbWithFallback := cb.WithFallback(func(ctx context.Context, err error) (*http.Response, error) {
		fallbackCalled.Store(true)
		return nil, fmt.Errorf("fallback error")
	})

	resp, err := cbWithFallback.Get(context.Background(), server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.False(t, fallbackCalled.Load())
}

func TestCircuitBreaker_WithFallback_FallbackErrorPropagated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := testCBConfig("test-fallback-err")
	cfg.MinRequests = 3
	cfg.Timeout = 5 * time.Second

	client := New(Config{Timeout: 5 * time.Second, MaxRetries: 0, MaxConnsPerHost: 10})
	cb := NewCircuitBreakerClient(client, cfg, testLogger())

	cbWithFallback := cb.WithFallback(func(ctx context.Context, err error) (*http.Response, error) {
		return nil, fmt.Errorf("fallback failed: %w", err)
	})

	for i := 0; i < 3; i++ {
		_, _ = cbWithFallback.Get(context.Background(), server.URL)
	}

	_, err := cbWithFallback.Get(context.Background(), server.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fallback failed")
}

func TestCircuitBreaker_WithoutFallback_StillReturnsErrCircuitOpen(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := testCBConfig("test-no-fallback")
	cfg.MinRequests = 3
	cfg.Timeout = 5 * time.Second

	client := New(Config{Timeout: 5 * time.Second, MaxRetries: 0, MaxConnsPerHost: 10})
	cb := NewCircuitBreakerClient(client, cfg, testLogger())

	for i := 0; i < 3; i++ {
		_, _ = cb.Get(context.Background(), server.URL)
	}

	_, err := cb.Get(context.Background(), server.URL)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCircuitOpen)
}

func TestCircuitBreaker_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Slow response.
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(Config{Timeout: 5 * time.Second, MaxRetries: 0, MaxConnsPerHost: 10})
	cb := NewCircuitBreakerClient(client, testCBConfig("test-ctx"), testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := cb.Get(ctx, server.URL)
	require.Error(t, err)
}
