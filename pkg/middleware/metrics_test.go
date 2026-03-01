package middleware

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// collectMetric is a test helper that extracts a specific label-matched metric
// from a Collector (CounterVec, HistogramVec, GaugeVec).
func collectMetric(t *testing.T, c prometheus.Collector, labels map[string]string) *dto.Metric {
	if t != nil {
		t.Helper()
	}
	ch := make(chan prometheus.Metric, 100)
	c.Collect(ch)
	close(ch)

	for m := range ch {
		d := &dto.Metric{}
		if err := m.Write(d); err != nil {
			continue
		}

		match := true
		for k, v := range labels {
			found := false
			for _, lp := range d.GetLabel() {
				if lp.GetName() == k && lp.GetValue() == v {
					found = true
					break
				}
			}
			if !found {
				match = false
				break
			}
		}
		if match {
			return d
		}
	}
	return nil
}

// serveWithChi wraps a handler in a chi router so RouteContext is available.
func serveWithChi(mw func(http.Handler) http.Handler, handler http.Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(mw)
	r.Get("/test", handler.ServeHTTP)
	return r
}

func TestPrometheusMetrics_RequestCounting(t *testing.T) {
	mw := PrometheusMetrics("test-svc")
	handler := serveWithChi(mw, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Send 3 requests.
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// Verify counter.
	labels := map[string]string{"service": "test-svc", "method": "GET", "path": "/test", "status": "200"}
	m := collectMetric(t, httpRequestsTotal, labels)
	require.NotNil(t, m, "counter metric should exist for test-svc GET /test 200")
	assert.GreaterOrEqual(t, m.GetCounter().GetValue(), float64(3))
}

func TestPrometheusMetrics_DurationHistogram(t *testing.T) {
	mw := PrometheusMetrics("hist-svc")
	handler := serveWithChi(mw, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	labels := map[string]string{"service": "hist-svc", "method": "GET", "path": "/test", "status": "201"}
	m := collectMetric(t, httpRequestDuration, labels)
	require.NotNil(t, m, "histogram metric should exist")
	assert.GreaterOrEqual(t, m.GetHistogram().GetSampleCount(), uint64(1))
}

func TestPrometheusMetrics_InFlightGauge(t *testing.T) {
	// We'll verify the gauge increments while handler is running.
	inFlightSeen := float64(-1)
	mw := PrometheusMetrics("inflight-svc")
	handler := serveWithChi(mw, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// While inside the handler, in-flight should be >= 1.
		labels := map[string]string{"service": "inflight-svc"}
		m := collectMetric(nil, httpRequestsInFlight, labels)
		if m != nil {
			inFlightSeen = m.GetGauge().GetValue()
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.GreaterOrEqual(t, inFlightSeen, float64(1), "in-flight gauge should be at least 1 during request")
}

func TestPrometheusMetrics_StatusCodeCapture(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"200 OK", http.StatusOK},
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svcName := "status-" + http.StatusText(tc.statusCode)
			mw := PrometheusMetrics(svcName)
			handler := serveWithChi(mw, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assert.Equal(t, tc.statusCode, rr.Code)
		})
	}
}

func TestPrometheusMetrics_DefaultStatusCode(t *testing.T) {
	// When handler doesn't call WriteHeader, default should be 200.
	mw := PrometheusMetrics("default-status-svc")
	handler := serveWithChi(mw, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	labels := map[string]string{"service": "default-status-svc", "status": "200"}
	m := collectMetric(t, httpRequestsTotal, labels)
	require.NotNil(t, m, "should record status 200 when WriteHeader not called explicitly")
}

// --- Flusher and Hijacker delegation tests ---

// mockFlusherWriter implements both http.ResponseWriter and http.Flusher.
type mockFlusherWriter struct {
	http.ResponseWriter
	flushed bool
}

func (m *mockFlusherWriter) Flush() {
	m.flushed = true
}

func TestMetricsResponseWriter_Flush_DelegatesToUnderlying(t *testing.T) {
	mock := &mockFlusherWriter{ResponseWriter: httptest.NewRecorder()}
	rw := &metricsResponseWriter{ResponseWriter: mock, statusCode: http.StatusOK}

	rw.Flush()
	assert.True(t, mock.flushed, "Flush should delegate to underlying ResponseWriter")
}

func TestMetricsResponseWriter_Flush_NoOpWhenNotSupported(t *testing.T) {
	// Plain httptest.Recorder does not implement Flusher in all cases.
	// Use a minimal ResponseWriter that definitely doesn't implement Flusher.
	rw := &metricsResponseWriter{ResponseWriter: &minimalResponseWriter{}, statusCode: http.StatusOK}

	// Should not panic.
	rw.Flush()
}

// mockHijackerWriter implements both http.ResponseWriter and http.Hijacker.
type mockHijackerWriter struct {
	http.ResponseWriter
	hijacked bool
}

func (m *mockHijackerWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	m.hijacked = true
	// Return nil conn for test purposes.
	return nil, nil, nil
}

func TestMetricsResponseWriter_Hijack_DelegatesToUnderlying(t *testing.T) {
	mock := &mockHijackerWriter{ResponseWriter: httptest.NewRecorder()}
	rw := &metricsResponseWriter{ResponseWriter: mock, statusCode: http.StatusOK}

	_, _, err := rw.Hijack()
	assert.NoError(t, err)
	assert.True(t, mock.hijacked, "Hijack should delegate to underlying ResponseWriter")
}

func TestMetricsResponseWriter_Hijack_ErrorWhenNotSupported(t *testing.T) {
	rw := &metricsResponseWriter{ResponseWriter: &minimalResponseWriter{}, statusCode: http.StatusOK}

	_, _, err := rw.Hijack()
	assert.ErrorIs(t, err, http.ErrNotSupported)
}

// minimalResponseWriter is a bare http.ResponseWriter without Flusher/Hijacker.
type minimalResponseWriter struct {
	header http.Header
}

func (m *minimalResponseWriter) Header() http.Header {
	if m.header == nil {
		m.header = make(http.Header)
	}
	return m.header
}

func (m *minimalResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (m *minimalResponseWriter) WriteHeader(int) {}

func TestMetricsResponseWriter_ImplementsFlusher(t *testing.T) {
	rw := &metricsResponseWriter{ResponseWriter: httptest.NewRecorder()}
	_, ok := interface{}(rw).(http.Flusher)
	assert.True(t, ok, "metricsResponseWriter should implement http.Flusher")
}

func TestMetricsResponseWriter_ImplementsHijacker(t *testing.T) {
	rw := &metricsResponseWriter{ResponseWriter: httptest.NewRecorder()}
	_, ok := interface{}(rw).(http.Hijacker)
	assert.True(t, ok, "metricsResponseWriter should implement http.Hijacker")
}
