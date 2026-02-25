package middleware

import (
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
)

const testSecret = "test-secret-key-for-jwt-signing"

// newTestLogger returns a logger that discards output (for test silence).
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// generateToken creates a signed JWT token with the given claims and secret.
func generateToken(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return tokenString
}

// echoHandler is a test handler that writes the X-User-ID header value to the response.
func echoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(userID))
	}
}

func TestJWTAuth_ValidToken_ExtractsUserID(t *testing.T) {
	tokenString := generateToken(t, testSecret, jwt.MapClaims{
		"user_id": "user-123",
		"exp":     jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
	})

	handler := JWTAuth(testSecret, newTestLogger())(echoHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "user-123", rr.Body.String())
}

func TestJWTAuth_ValidToken_SubClaim(t *testing.T) {
	tokenString := generateToken(t, testSecret, jwt.MapClaims{
		"sub": "user-456",
		"exp": jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
	})

	handler := JWTAuth(testSecret, newTestLogger())(echoHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "user-456", rr.Body.String())
}

func TestJWTAuth_InvalidToken_Returns401(t *testing.T) {
	handler := JWTAuth(testSecret, newTestLogger())(echoHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "UNAUTHORIZED")
}

func TestJWTAuth_MissingToken_ProtectedRoute_Returns401(t *testing.T) {
	handler := JWTAuth(testSecret, newTestLogger())(echoHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "missing authorization header")
}

func TestJWTAuth_InvalidHeaderFormat_Returns401(t *testing.T) {
	handler := JWTAuth(testSecret, newTestLogger())(echoHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", nil)
	req.Header.Set("Authorization", "Token some-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid authorization header format")
}

func TestJWTAuth_ExpiredToken_Returns401(t *testing.T) {
	tokenString := generateToken(t, testSecret, jwt.MapClaims{
		"user_id": "user-789",
		"exp":     jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
	})

	handler := JWTAuth(testSecret, newTestLogger())(echoHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "UNAUTHORIZED")
}

func TestJWTAuth_WrongSecret_Returns401(t *testing.T) {
	tokenString := generateToken(t, "wrong-secret", jwt.MapClaims{
		"user_id": "user-123",
		"exp":     jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
	})

	handler := JWTAuth(testSecret, newTestLogger())(echoHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestJWTAuth_PublicRoute_GetProducts_NoAuthRequired(t *testing.T) {
	handler := JWTAuth(testSecret, newTestLogger())(echoHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestJWTAuth_PublicRoute_GetProductBySlug_NoAuthRequired(t *testing.T) {
	handler := JWTAuth(testSecret, newTestLogger())(echoHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/my-product-slug", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestJWTAuth_PublicRoute_GetSearch_NoAuthRequired(t *testing.T) {
	handler := JWTAuth(testSecret, newTestLogger())(echoHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=laptop", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestJWTAuth_PublicRoute_PostAuth_NoAuthRequired(t *testing.T) {
	handler := JWTAuth(testSecret, newTestLogger())(echoHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestJWTAuth_PublicRoute_HealthCheck_NoAuthRequired(t *testing.T) {
	handler := JWTAuth(testSecret, newTestLogger())(echoHandler())
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestJWTAuth_ProtectedRoute_PostProducts_RequiresAuth(t *testing.T) {
	handler := JWTAuth(testSecret, newTestLogger())(echoHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestJWTAuth_OptionsRequest_AlwaysAllowed(t *testing.T) {
	handler := JWTAuth(testSecret, newTestLogger())(echoHandler())
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/cart", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestIsPublicRoute(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   bool
	}{
		{http.MethodGet, "/api/v1/products", true},
		{http.MethodGet, "/api/v1/products/some-slug", true},
		{http.MethodPost, "/api/v1/products", false},
		{http.MethodGet, "/api/v1/search", true},
		{http.MethodGet, "/api/v1/search?q=test", true},
		{http.MethodPost, "/api/v1/auth/login", true},
		{http.MethodPost, "/api/v1/auth/register", true},
		{http.MethodGet, "/health/live", true},
		{http.MethodGet, "/health/ready", true},
		{http.MethodPost, "/api/v1/cart", false},
		{http.MethodPost, "/api/v1/orders", false},
		{http.MethodDelete, "/api/v1/products/123", false},
		{http.MethodOptions, "/api/v1/anything", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.method, tt.path), func(t *testing.T) {
			got := isPublicRoute(tt.method, tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}
