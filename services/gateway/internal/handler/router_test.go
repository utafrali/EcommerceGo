package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/utafrali/EcommerceGo/pkg/health"
	"github.com/utafrali/EcommerceGo/services/gateway/internal/config"
	"github.com/utafrali/EcommerceGo/services/gateway/internal/proxy"
)

const testJWTSecret = "test-jwt-secret-for-router-tests"

// testLogger returns a logger that discards output.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// serviceEchoServer creates a test server that responds with the service name
// and requested path, allowing tests to verify which backend received the request.
func serviceEchoServer(serviceName string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"service": serviceName,
			"path":    r.URL.Path,
		})
	}))
}

// testRouter holds a fully wired gateway router with echo backend servers.
type testRouter struct {
	handler http.Handler
	servers map[string]*httptest.Server
}

func newTestRouter(t *testing.T) *testRouter {
	t.Helper()

	services := []string{
		"product", "cart", "order", "checkout", "payment",
		"user", "inventory", "campaign", "notification", "search", "media",
	}

	servers := make(map[string]*httptest.Server)
	for _, name := range services {
		servers[name] = serviceEchoServer(name)
	}

	cfg := &config.Config{
		Environment:            "development",
		JWTSecret:              testJWTSecret,
		RateLimitRPS:           10000,
		RateLimitBurst:         20000,
		CORSAllowedOrigins:     []string{"*"},
		CORSAllowedMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		CORSAllowedHeaders:     []string{"Accept", "Authorization", "Content-Type", "X-Correlation-ID", "X-User-ID"},
		CORSMaxAge:             3600,
		MetricsAllowedCIDRs:    []string{"127.0.0.0/8", "10.0.0.0/8", "192.168.0.0/16"},
		ProductServiceURL:      servers["product"].URL,
		CartServiceURL:         servers["cart"].URL,
		OrderServiceURL:        servers["order"].URL,
		CheckoutServiceURL:     servers["checkout"].URL,
		PaymentServiceURL:      servers["payment"].URL,
		UserServiceURL:         servers["user"].URL,
		InventoryServiceURL:    servers["inventory"].URL,
		CampaignServiceURL:     servers["campaign"].URL,
		NotificationServiceURL: servers["notification"].URL,
		SearchServiceURL:       servers["search"].URL,
		MediaServiceURL:        servers["media"].URL,
		ProxyDialTimeout:       5 * time.Second,
		ProxyResponseTimeout:   30 * time.Second,
		ProxyIdleTimeout:       90 * time.Second,
		ProxyMaxIdleConns:      100,
	}

	logger := testLogger()
	sp := proxy.NewServiceProxy(cfg, logger)
	healthHandler := health.NewHandler()
	router := NewRouter(cfg, sp, healthHandler, logger)

	t.Cleanup(func() {
		for _, s := range servers {
			s.Close()
		}
	})

	return &testRouter{
		handler: router,
		servers: servers,
	}
}

func generateRouterTestToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)
	return s
}

func validRouterJWT(t *testing.T) string {
	t.Helper()
	return generateRouterTestToken(t, jwt.MapClaims{
		"user_id": "test-user-123",
		"email":   "test@example.com",
		"role":    "user",
		"exp":     jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
	})
}

// --- Health Endpoint Tests ---

func TestRouter_HealthLive_Returns200(t *testing.T) {
	tr := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()

	tr.handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRouter_HealthReady_Returns200(t *testing.T) {
	tr := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()

	tr.handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// --- Public Route Proxy Tests ---

func TestRouter_PublicRoutes_ProxyToCorrectService(t *testing.T) {
	tr := newTestRouter(t)

	tests := []struct {
		name            string
		method          string
		path            string
		expectedService string
	}{
		{"GET products", http.MethodGet, "/api/v1/products", "product"},
		{"GET product by slug", http.MethodGet, "/api/v1/products/my-product", "product"},
		{"GET categories", http.MethodGet, "/api/v1/categories", "product"},
		{"GET category by id", http.MethodGet, "/api/v1/categories/electronics", "product"},
		{"GET brands", http.MethodGet, "/api/v1/brands", "product"},
		{"GET banners", http.MethodGet, "/api/v1/banners", "product"},
		{"GET search", http.MethodGet, "/api/v1/search?q=laptop", "search"},
		{"GET campaigns", http.MethodGet, "/api/v1/campaigns", "campaign"},
		{"GET campaign by slug", http.MethodGet, "/api/v1/campaigns/summer-sale", "campaign"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.RemoteAddr = "127.0.0.1:12345"
			rr := httptest.NewRecorder()

			tr.handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code, "expected 200 for public route %s %s", tt.method, tt.path)

			var body map[string]string
			err := json.Unmarshal(rr.Body.Bytes(), &body)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedService, body["service"],
				"request should be proxied to %s service", tt.expectedService)
		})
	}
}

// --- Auth Public Route Bypass ---

func TestRouter_PostAuth_NoAuthRequired(t *testing.T) {
	tr := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()

	tr.handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var body map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "user", body["service"])
}

// --- Protected Route Tests ---

func TestRouter_ProtectedRoutes_RequireAuth(t *testing.T) {
	tr := newTestRouter(t)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"POST cart", http.MethodPost, "/api/v1/cart"},
		{"POST orders", http.MethodPost, "/api/v1/orders"},
		{"POST checkout", http.MethodPost, "/api/v1/checkout"},
		{"POST payments", http.MethodPost, "/api/v1/payments"},
		{"GET notifications", http.MethodGet, "/api/v1/notifications"},
		{"POST inventory", http.MethodPost, "/api/v1/inventory"},
		{"POST media", http.MethodPost, "/api/v1/media"},
		{"DELETE products", http.MethodDelete, "/api/v1/products/123"},
		{"POST coupons", http.MethodPost, "/api/v1/coupons"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.RemoteAddr = "127.0.0.1:12345"
			rr := httptest.NewRecorder()

			tr.handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusUnauthorized, rr.Code,
				"protected route %s %s should return 401 without auth", tt.method, tt.path)
			assert.Contains(t, rr.Body.String(), "UNAUTHORIZED")
		})
	}
}

func TestRouter_ProtectedRoutes_WithValidJWT_ProxyToCorrectService(t *testing.T) {
	tr := newTestRouter(t)
	token := validRouterJWT(t)

	tests := []struct {
		name            string
		method          string
		path            string
		expectedService string
	}{
		{"POST cart", http.MethodPost, "/api/v1/cart", "cart"},
		{"POST orders", http.MethodPost, "/api/v1/orders", "order"},
		{"POST checkout", http.MethodPost, "/api/v1/checkout", "checkout"},
		{"POST payments", http.MethodPost, "/api/v1/payments", "payment"},
		{"GET users", http.MethodGet, "/api/v1/users", "user"},
		{"POST inventory", http.MethodPost, "/api/v1/inventory", "inventory"},
		{"GET notifications", http.MethodGet, "/api/v1/notifications", "notification"},
		{"POST media", http.MethodPost, "/api/v1/media", "media"},
		{"POST coupons", http.MethodPost, "/api/v1/coupons", "campaign"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.RemoteAddr = "127.0.0.1:12345"
			rr := httptest.NewRecorder()

			tr.handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code,
				"expected 200 for authenticated %s %s", tt.method, tt.path)

			var body map[string]string
			err := json.Unmarshal(rr.Body.Bytes(), &body)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedService, body["service"],
				"request should be proxied to %s service", tt.expectedService)
		})
	}
}

// --- All 11 Services Routing Verification ---

func TestRouter_AllServicePaths_RouteCorrectly(t *testing.T) {
	tr := newTestRouter(t)
	token := validRouterJWT(t)

	// Verify each of the 11 backend services is reachable.
	tests := []struct {
		path    string
		service string
	}{
		{"/api/v1/products", "product"},
		{"/api/v1/cart", "cart"},
		{"/api/v1/orders", "order"},
		{"/api/v1/checkout", "checkout"},
		{"/api/v1/payments", "payment"},
		{"/api/v1/users", "user"},
		{"/api/v1/inventory", "inventory"},
		{"/api/v1/campaigns", "campaign"},
		{"/api/v1/notifications", "notification"},
		{"/api/v1/search", "search"},
		{"/api/v1/media", "media"},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.RemoteAddr = "127.0.0.1:12345"
			rr := httptest.NewRecorder()

			tr.handler.ServeHTTP(rr, req)

			require.Equal(t, http.StatusOK, rr.Code, "service %s should be reachable", tt.service)

			var body map[string]string
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
			assert.Equal(t, tt.service, body["service"])
		})
	}
}

// --- 404 Handling ---

func TestRouter_UnknownPath_Returns404(t *testing.T) {
	tr := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()

	tr.handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestRouter_UnknownAPIPath_WithAuth_Returns404(t *testing.T) {
	tr := newTestRouter(t)
	token := validRouterJWT(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()

	tr.handler.ServeHTTP(rr, req)

	// chi returns 404 for paths that don't match any route within the /api group.
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// --- JWT User Context Forwarding ---

func TestRouter_JWT_ForwardsUserContextHeaders(t *testing.T) {
	// Create a backend that captures headers.
	headerCapture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"X-User-ID":    r.Header.Get("X-User-ID"),
			"X-User-Email": r.Header.Get("X-User-Email"),
			"X-User-Role":  r.Header.Get("X-User-Role"),
		})
	}))
	defer headerCapture.Close()

	cfg := &config.Config{
		Environment:            "development",
		JWTSecret:              testJWTSecret,
		RateLimitRPS:           10000,
		RateLimitBurst:         20000,
		CORSAllowedOrigins:     []string{"*"},
		CORSAllowedMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		CORSAllowedHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		CORSMaxAge:             3600,
		MetricsAllowedCIDRs:    []string{"127.0.0.0/8"},
		ProductServiceURL:      headerCapture.URL,
		CartServiceURL:         headerCapture.URL,
		OrderServiceURL:        headerCapture.URL,
		CheckoutServiceURL:     headerCapture.URL,
		PaymentServiceURL:      headerCapture.URL,
		UserServiceURL:         headerCapture.URL,
		InventoryServiceURL:    headerCapture.URL,
		CampaignServiceURL:     headerCapture.URL,
		NotificationServiceURL: headerCapture.URL,
		SearchServiceURL:       headerCapture.URL,
		MediaServiceURL:        headerCapture.URL,
		ProxyDialTimeout:       5 * time.Second,
		ProxyResponseTimeout:   30 * time.Second,
		ProxyIdleTimeout:       90 * time.Second,
		ProxyMaxIdleConns:      100,
	}

	logger := testLogger()
	sp := proxy.NewServiceProxy(cfg, logger)
	healthHandler := health.NewHandler()
	router := NewRouter(cfg, sp, healthHandler, logger)

	token := generateRouterTestToken(t, jwt.MapClaims{
		"user_id": "user-42",
		"email":   "alice@example.com",
		"role":    "admin",
		"exp":     jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var headers map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &headers))
	assert.Equal(t, "user-42", headers["X-User-ID"])
	assert.Equal(t, "alice@example.com", headers["X-User-Email"])
	assert.Equal(t, "admin", headers["X-User-Role"])
}

// --- Expired JWT ---

func TestRouter_ExpiredJWT_Returns401(t *testing.T) {
	tr := newTestRouter(t)

	token := generateRouterTestToken(t, jwt.MapClaims{
		"user_id": "user-42",
		"exp":     jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()

	tr.handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "UNAUTHORIZED")
}

// --- Metrics Endpoint (via router) ---

func TestRouter_MetricsEndpoint_AllowedIP_Returns200(t *testing.T) {
	tr := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()

	tr.handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRouter_MetricsEndpoint_BlockedIP_Returns403(t *testing.T) {
	tr := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = fmt.Sprintf("203.0.113.50:%d", 12345)
	rr := httptest.NewRecorder()

	tr.handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Contains(t, rr.Body.String(), "FORBIDDEN")
}
