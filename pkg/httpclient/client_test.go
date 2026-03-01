package httpclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, time.Second, cfg.RetryWaitMin)
	assert.Equal(t, 5*time.Second, cfg.RetryWaitMax)
	assert.Equal(t, 100, cfg.MaxConnsPerHost)
}

func TestNew_ReturnsClient(t *testing.T) {
	client := New(DefaultConfig())
	assert.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
}

func TestGet_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := New(Config{
		Timeout:         5 * time.Second,
		MaxRetries:      0,
		MaxConnsPerHost: 10,
	})

	resp, err := client.Get(context.Background(), server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "ok")
}

func TestPost_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, `{"name":"test"}`, string(body))
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := New(Config{
		Timeout:         5 * time.Second,
		MaxRetries:      0,
		MaxConnsPerHost: 10,
	})

	resp, err := client.Post(context.Background(), server.URL, "application/json",
		strings.NewReader(`{"name":"test"}`))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestDo_Retries5xx(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(Config{
		Timeout:         5 * time.Second,
		MaxRetries:      3,
		RetryWaitMin:    1 * time.Millisecond,
		RetryWaitMax:    5 * time.Millisecond,
		MaxConnsPerHost: 10,
	})

	resp, err := client.Get(context.Background(), server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))
}

func TestDo_DoesNotRetry501(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusNotImplemented) // 501
	}))
	defer server.Close()

	client := New(Config{
		Timeout:         5 * time.Second,
		MaxRetries:      3,
		RetryWaitMin:    1 * time.Millisecond,
		RetryWaitMax:    5 * time.Millisecond,
		MaxConnsPerHost: 10,
	})

	resp, err := client.Get(context.Background(), server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotImplemented, resp.StatusCode)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))
}

func TestDo_DoesNotRetry4xx(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadRequest) // 400
	}))
	defer server.Close()

	client := New(Config{
		Timeout:         5 * time.Second,
		MaxRetries:      3,
		RetryWaitMin:    1 * time.Millisecond,
		RetryWaitMax:    5 * time.Millisecond,
		MaxConnsPerHost: 10,
	})

	resp, err := client.Get(context.Background(), server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))
}

func TestDo_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := New(Config{
		Timeout:         5 * time.Second,
		MaxRetries:      10,
		RetryWaitMin:    100 * time.Millisecond,
		RetryWaitMax:    500 * time.Millisecond,
		MaxConnsPerHost: 10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.Get(ctx, server.URL)
	require.Error(t, err)
}

func TestGet_InvalidURL(t *testing.T) {
	client := New(Config{
		Timeout:         5 * time.Second,
		MaxRetries:      0,
		MaxConnsPerHost: 10,
	})

	_, err := client.Get(context.Background(), "://invalid")
	require.Error(t, err)
}

func TestIsRetryableError(t *testing.T) {
	assert.False(t, isRetryableError(nil))
	assert.False(t, isRetryableError(context.Canceled))
	// context.DeadlineExceeded implements net.Error so it is considered retryable
	// by the net.Error check (which runs before the context check).
	assert.True(t, isRetryableError(context.DeadlineExceeded))
}

func TestAddJitter_Distribution(t *testing.T) {
	// Verify that jitter produces values in the expected ±25% range
	// and that the distribution is not degenerate (all identical).
	const base = 1 * time.Second
	const samples = 200

	var minVal, maxVal time.Duration
	var sum time.Duration

	for i := 0; i < samples; i++ {
		d := addJitter(base)
		if i == 0 || d < minVal {
			minVal = d
		}
		if i == 0 || d > maxVal {
			maxVal = d
		}
		sum += d

		// Each sample must be within ±25% of the base.
		assert.GreaterOrEqual(t, d, time.Duration(float64(base)*0.75),
			"jitter value %v is below 75%% of base", d)
		assert.LessOrEqual(t, d, time.Duration(float64(base)*1.25),
			"jitter value %v is above 125%% of base", d)
	}

	// The spread (max - min) should be non-trivial, verifying actual randomness.
	spread := maxVal - minVal
	assert.Greater(t, spread, 50*time.Millisecond,
		"jitter spread %v is too narrow; expected meaningful variation", spread)

	// The mean should be close to the base (within 10%).
	mean := sum / time.Duration(samples)
	assert.InDelta(t, float64(base), float64(mean), float64(base)*0.1,
		"mean jitter %v deviates too much from base %v", mean, base)
}

func TestAddJitter_ZeroDuration(t *testing.T) {
	d := addJitter(0)
	assert.Equal(t, time.Duration(0), d)
}

func TestAddJitter_SmallDuration(t *testing.T) {
	d := addJitter(1 * time.Millisecond)
	assert.GreaterOrEqual(t, d, time.Duration(0))
	assert.LessOrEqual(t, d, 2*time.Millisecond)
}
