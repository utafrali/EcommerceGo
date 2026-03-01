package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- metricsIPAllowlist Unit Tests ---

func TestMetricsIPAllowlist_AllowedCIDR_PassesThrough(t *testing.T) {
	handler := metricsIPAllowlist([]string{"10.0.0.0/8"}, testLogger())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("metrics"))
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "metrics", rr.Body.String())
}

func TestMetricsIPAllowlist_BlockedIP_Returns403(t *testing.T) {
	handler := metricsIPAllowlist([]string{"10.0.0.0/8"}, testLogger())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = "203.0.113.50:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Contains(t, rr.Body.String(), "FORBIDDEN")
	assert.Contains(t, rr.Body.String(), "metrics endpoint is restricted")
}

func TestMetricsIPAllowlist_DefaultPrivateRanges(t *testing.T) {
	cidrs := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8"}
	handler := metricsIPAllowlist(cidrs, testLogger())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	tests := []struct {
		name string
		ip   string
		want int
	}{
		{"loopback", "127.0.0.1:12345", http.StatusOK},
		{"10.x.x.x", "10.20.30.40:12345", http.StatusOK},
		{"172.16.x.x", "172.16.5.10:12345", http.StatusOK},
		{"192.168.x.x", "192.168.1.100:12345", http.StatusOK},
		{"public_ip_blocked", "8.8.8.8:12345", http.StatusForbidden},
		{"another_public_ip", "203.0.113.1:12345", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			req.RemoteAddr = tt.ip
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.want, rr.Code)
		})
	}
}

func TestMetricsIPAllowlist_InvalidCIDR_Skipped(t *testing.T) {
	// One invalid CIDR and one valid. The invalid is skipped; valid still works.
	handler := metricsIPAllowlist([]string{"invalid-cidr", "10.0.0.0/8"}, testLogger())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMetricsIPAllowlist_EmptyCIDRs_BlocksAll(t *testing.T) {
	handler := metricsIPAllowlist([]string{}, testLogger())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestMetricsIPAllowlist_IPv6Loopback(t *testing.T) {
	handler := metricsIPAllowlist([]string{"::1/128"}, testLogger())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = "[::1]:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMetricsIPAllowlist_RemoteAddrWithoutPort(t *testing.T) {
	handler := metricsIPAllowlist([]string{"10.0.0.0/8"}, testLogger())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = "10.0.0.1" // no port
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMetricsIPAllowlist_ResponseBody_HasJSONStructure(t *testing.T) {
	handler := metricsIPAllowlist([]string{"10.0.0.0/8"}, testLogger())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = "203.0.113.50:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestMetricsIPAllowlist_MultipleCIDRs_AnyMatch(t *testing.T) {
	cidrs := []string{"10.0.0.0/8", "172.16.0.0/12"}
	handler := metricsIPAllowlist(cidrs, testLogger())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	// IP from second CIDR should also be allowed.
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = "172.16.0.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
