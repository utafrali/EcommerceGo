package proxy

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/utafrali/EcommerceGo/services/gateway/internal/config"
)

func proxyTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func proxyTestConfig(serviceURL string) *config.Config {
	return &config.Config{
		ProductServiceURL:      serviceURL,
		CartServiceURL:         serviceURL,
		OrderServiceURL:        serviceURL,
		CheckoutServiceURL:     serviceURL,
		PaymentServiceURL:      serviceURL,
		UserServiceURL:         serviceURL,
		InventoryServiceURL:    serviceURL,
		CampaignServiceURL:     serviceURL,
		NotificationServiceURL: serviceURL,
		SearchServiceURL:       serviceURL,
		MediaServiceURL:        serviceURL,
		ProxyDialTimeout:       5 * time.Second,
		ProxyResponseTimeout:   30 * time.Second,
		ProxyIdleTimeout:       90 * time.Second,
		ProxyMaxIdleConns:      100,
	}
}

// --- Handler Registration Tests ---

func TestServiceProxy_Handler_KnownService_ProxiesRequest(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"proxied": "true"})
	}))
	defer backend.Close()

	cfg := proxyTestConfig(backend.URL)
	sp := NewServiceProxy(cfg, proxyTestLogger())

	handler := sp.Handler("product")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, "true", body["proxied"])
}

func TestServiceProxy_Handler_UnknownService_Returns502(t *testing.T) {
	cfg := proxyTestConfig("http://localhost:1")
	sp := NewServiceProxy(cfg, proxyTestLogger())

	handler := sp.Handler("nonexistent")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadGateway, rr.Code)
	assert.Contains(t, rr.Body.String(), "SERVICE_UNAVAILABLE")
	assert.Contains(t, rr.Body.String(), "service not configured")
}

func TestServiceProxy_AllServices_Registered(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	cfg := proxyTestConfig(backend.URL)
	sp := NewServiceProxy(cfg, proxyTestLogger())

	services := []string{
		"product", "cart", "order", "checkout", "payment",
		"user", "inventory", "campaign", "notification", "search", "media",
	}

	for _, name := range services {
		t.Run(name, func(t *testing.T) {
			handler := sp.Handler(name)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// Should NOT return 502 with SERVICE_UNAVAILABLE (that's the unknown service handler).
			assert.NotContains(t, rr.Body.String(), "SERVICE_UNAVAILABLE",
				"service %s should be registered", name)
		})
	}
}

// --- Error Handler Tests ---

func TestServiceProxy_UpstreamUnavailable_Returns502(t *testing.T) {
	// Create and immediately close a server to get an unreachable URL.
	closedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedServer.Close()

	cfg := proxyTestConfig(closedServer.URL)
	sp := NewServiceProxy(cfg, proxyTestLogger())

	handler := sp.Handler("product")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadGateway, rr.Code)
	assert.Contains(t, rr.Body.String(), "BAD_GATEWAY")
	assert.Contains(t, rr.Body.String(), "upstream service unavailable")
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestServiceProxy_UpstreamTimeout_Returns502(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than the proxy response timeout.
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	cfg := proxyTestConfig(backend.URL)
	cfg.ProxyResponseTimeout = 100 * time.Millisecond // Very short timeout.
	sp := NewServiceProxy(cfg, proxyTestLogger())

	handler := sp.Handler("product")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadGateway, rr.Code)
	assert.Contains(t, rr.Body.String(), "BAD_GATEWAY")
}

func TestServiceProxy_Upstream5xx_Passthrough(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer backend.Close()

	cfg := proxyTestConfig(backend.URL)
	sp := NewServiceProxy(cfg, proxyTestLogger())

	handler := sp.Handler("product")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// The reverse proxy should pass through the upstream's 500 status as-is.
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "internal server error")
}

func TestServiceProxy_Upstream503_Passthrough(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"service temporarily unavailable"}`))
	}))
	defer backend.Close()

	cfg := proxyTestConfig(backend.URL)
	sp := NewServiceProxy(cfg, proxyTestLogger())

	handler := sp.Handler("order")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Contains(t, rr.Body.String(), "service temporarily unavailable")
}

// --- Proxy Header Forwarding ---

func TestServiceProxy_ForwardsHeaders(t *testing.T) {
	var capturedHeaders http.Header

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	cfg := proxyTestConfig(backend.URL)
	sp := NewServiceProxy(cfg, proxyTestLogger())

	handler := sp.Handler("product")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	req.Header.Set("X-User-ID", "user-123")
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "user-123", capturedHeaders.Get("X-User-ID"))
	assert.Equal(t, "Bearer test-token", capturedHeaders.Get("Authorization"))
	assert.NotEmpty(t, capturedHeaders.Get("X-Forwarded-Proto"))
}

func TestServiceProxy_SetsXForwardedProto(t *testing.T) {
	var capturedProto string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedProto = r.Header.Get("X-Forwarded-Proto")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	cfg := proxyTestConfig(backend.URL)
	sp := NewServiceProxy(cfg, proxyTestLogger())

	handler := sp.Handler("product")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "http", capturedProto)
}
