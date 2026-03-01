package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCORS_DevMode_AllowsWildcard(t *testing.T) {
	handler := CORS(CORSConfig{
		AllowedOrigins: []string{"*"},
		Environment:    "development",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCORS_DevMode_NoOriginStillWildcard(t *testing.T) {
	handler := CORS(CORSConfig{
		AllowedOrigins: []string{"*"},
		Environment:    "development",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_ProdMode_AllowedOrigin(t *testing.T) {
	handler := CORS(CORSConfig{
		AllowedOrigins: []string{"https://example.com", "https://admin.example.com"},
		Environment:    "production",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "https://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Origin", rr.Header().Get("Vary"))
}

func TestCORS_ProdMode_SecondAllowedOrigin(t *testing.T) {
	handler := CORS(CORSConfig{
		AllowedOrigins: []string{"https://example.com", "https://admin.example.com"},
		Environment:    "production",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://admin.example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "https://admin.example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Origin", rr.Header().Get("Vary"))
}

func TestCORS_ProdMode_RejectedOrigin(t *testing.T) {
	handler := CORS(CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		Environment:    "production",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCORS_ProdMode_NoOrigin(t *testing.T) {
	handler := CORS(CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		Environment:    "production",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No Origin header set.
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_ProdMode_WildcardInList_AllowsAll(t *testing.T) {
	handler := CORS(CORSConfig{
		AllowedOrigins: []string{"https://example.com", "*"},
		Environment:    "production",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://anything.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_PreflightOptions_Returns204(t *testing.T) {
	handler := CORS(CORSConfig{
		AllowedOrigins: []string{"*"},
		Environment:    "development",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This should NOT be reached for OPTIONS.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("should not reach"))
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.Empty(t, rr.Body.String())
}

func TestCORS_AllowedHeaders_AreSet(t *testing.T) {
	handler := CORS(CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedHeaders: []string{"Accept", "Authorization", "X-Custom"},
		Environment:    "development",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "Accept, Authorization, X-Custom", rr.Header().Get("Access-Control-Allow-Headers"))
}

func TestCORS_ExposedHeaders(t *testing.T) {
	handler := CORS(CORSConfig{
		AllowedOrigins: []string{"*"},
		ExposedHeaders: []string{"X-Correlation-ID", "X-User-ID"},
		Environment:    "development",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "X-Correlation-ID, X-User-ID", rr.Header().Get("Access-Control-Expose-Headers"))
}

func TestCORS_MaxAge(t *testing.T) {
	handler := CORS(CORSConfig{
		AllowedOrigins: []string{"*"},
		MaxAge:         7200,
		Environment:    "development",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "7200", rr.Header().Get("Access-Control-Max-Age"))
}

func TestCORS_AllowCredentials(t *testing.T) {
	handler := CORS(CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowCredentials: true,
		Environment:      "production",
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORS_DefaultMethods(t *testing.T) {
	handler := CORS(CORSConfig{
		AllowedOrigins: []string{"*"},
		Environment:    "development",
		// AllowedMethods not set — should use defaults.
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "GET, POST, PUT, PATCH, DELETE, OPTIONS", rr.Header().Get("Access-Control-Allow-Methods"))
}

func TestCORS_DefaultConfig(t *testing.T) {
	cfg := DefaultCORSConfig()
	assert.Equal(t, []string{"*"}, cfg.AllowedOrigins)
	assert.Contains(t, cfg.AllowedMethods, "GET")
	assert.Contains(t, cfg.AllowedMethods, "POST")
	assert.Equal(t, 3600, cfg.MaxAge)
	assert.Equal(t, "development", cfg.Environment)
}
