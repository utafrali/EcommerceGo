package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- ContentTypeJSON Middleware Tests ---

func TestContentTypeJSON_PostWithoutContentType_Returns415(t *testing.T) {
	handler := ContentTypeJSON(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(`{"key":"value"}`))
	// Intentionally do NOT set Content-Type
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
	assert.Contains(t, rr.Body.String(), "UNSUPPORTED_MEDIA_TYPE")
}

func TestContentTypeJSON_PutWithoutContentType_Returns415(t *testing.T) {
	handler := ContentTypeJSON(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPut, "/api/test", strings.NewReader(`{"key":"value"}`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
	assert.Contains(t, rr.Body.String(), "UNSUPPORTED_MEDIA_TYPE")
}

func TestContentTypeJSON_PatchWithoutContentType_Returns415(t *testing.T) {
	handler := ContentTypeJSON(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPatch, "/api/test", strings.NewReader(`{"key":"value"}`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
	assert.Contains(t, rr.Body.String(), "UNSUPPORTED_MEDIA_TYPE")
}

func TestContentTypeJSON_PostWithValidJSON_Passes(t *testing.T) {
	called := false
	handler := ContentTypeJSON(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(`{"key":"value"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, called, "next handler should have been called")
}

func TestContentTypeJSON_PostWithJSONCharset_Passes(t *testing.T) {
	called := false
	handler := ContentTypeJSON(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(`{"key":"value"}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, called, "next handler should have been called with charset variant")
}

func TestContentTypeJSON_PostWithWrongContentType_Returns415(t *testing.T) {
	handler := ContentTypeJSON(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(`key=value`))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
	assert.Contains(t, rr.Body.String(), "UNSUPPORTED_MEDIA_TYPE")
}

func TestContentTypeJSON_GetWithoutContentType_Passes(t *testing.T) {
	called := false
	handler := ContentTypeJSON(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, called, "GET requests without Content-Type should pass through")
}

func TestContentTypeJSON_DeleteWithoutContentType_Passes(t *testing.T) {
	called := false
	handler := ContentTypeJSON(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodDelete, "/api/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, called, "DELETE requests without body should pass through")
}

func TestContentTypeJSON_ResponseContentType_IsJSON(t *testing.T) {
	handler := ContentTypeJSON(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(`data`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

// --- CORS Middleware Tests ---

func TestCORS_DevMode_AllowsWildcard(t *testing.T) {
	corsMiddleware := CORS(CORSConfig{
		AllowedOrigins: []string{},
		Environment:    "development",
	})

	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCORS_DevMode_NoOrigin_StillWildcard(t *testing.T) {
	corsMiddleware := CORS(CORSConfig{
		AllowedOrigins: []string{},
		Environment:    "development",
	})

	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_ProdMode_AllowedOrigin_SetsHeader(t *testing.T) {
	corsMiddleware := CORS(CORSConfig{
		AllowedOrigins: []string{"https://example.com", "https://www.example.com"},
		Environment:    "production",
	})

	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "https://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Origin", rr.Header().Get("Vary"))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCORS_ProdMode_DisallowedOrigin_NoHeader(t *testing.T) {
	corsMiddleware := CORS(CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		Environment:    "production",
	})

	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCORS_ProdMode_NoOrigin_NoHeader(t *testing.T) {
	corsMiddleware := CORS(CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		Environment:    "production",
	})

	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_ProdMode_WildcardInList_AllowsAll(t *testing.T) {
	corsMiddleware := CORS(CORSConfig{
		AllowedOrigins: []string{"*"},
		Environment:    "production",
	})

	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "https://any-origin.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_PreflightOptions_Returns204(t *testing.T) {
	corsMiddleware := CORS(CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		Environment:    "production",
	})

	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // Should NOT be reached
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.Equal(t, "https://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, rr.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	assert.Equal(t, "3600", rr.Header().Get("Access-Control-Max-Age"))
}

func TestCORS_AllowedHeaders_AreSet(t *testing.T) {
	corsMiddleware := CORS(CORSConfig{
		AllowedOrigins: []string{},
		Environment:    "development",
	})

	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	allowHeaders := rr.Header().Get("Access-Control-Allow-Headers")
	assert.Contains(t, allowHeaders, "Accept")
	assert.Contains(t, allowHeaders, "Authorization")
	assert.Contains(t, allowHeaders, "Content-Type")
	assert.Contains(t, allowHeaders, "X-Correlation-ID")
}

func TestCORS_ProdMode_SecondAllowedOrigin(t *testing.T) {
	corsMiddleware := CORS(CORSConfig{
		AllowedOrigins: []string{"https://example.com", "https://admin.example.com"},
		Environment:    "production",
	})

	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "https://admin.example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "https://admin.example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Origin", rr.Header().Get("Vary"))
}
