package middleware

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// publicRoutes defines path prefixes and methods that do not require authentication.
var publicRoutes = []struct {
	method string
	prefix string
}{
	{method: http.MethodGet, prefix: "/api/v1/products"},
	{method: http.MethodGet, prefix: "/api/v1/categories"},
	{method: http.MethodGet, prefix: "/api/v1/brands"},
	{method: http.MethodGet, prefix: "/api/v1/banners"},
	{method: http.MethodGet, prefix: "/api/v1/search"},
	{method: http.MethodGet, prefix: "/api/v1/campaigns"},
	{method: http.MethodPost, prefix: "/api/v1/auth"},
	{method: http.MethodGet, prefix: "/health"},
}

// trustedHeaders are headers injected by the gateway from JWT claims.
// They are stripped from all incoming requests to prevent spoofing,
// then set by the middleware after successful JWT validation.
var trustedHeaders = []string{
	"X-User-ID",
	"X-User-Email",
	"X-User-Role",
}

// isPublicRoute checks whether a given method + path combination is public.
func isPublicRoute(method, path string) bool {
	for _, route := range publicRoutes {
		if method == route.method && strings.HasPrefix(path, route.prefix) {
			return true
		}
	}
	// OPTIONS requests are always allowed (for CORS preflight).
	if method == http.MethodOptions {
		return true
	}
	return false
}

// JWTAuth returns middleware that validates JWT tokens from the Authorization header.
// For public routes, the request is passed through without authentication.
// For protected routes, the token is validated and user context headers
// (X-User-ID, X-User-Email, X-User-Role) are injected into the proxied request.
//
// Security: trusted headers are always stripped from incoming requests to prevent
// clients from spoofing user context. They are only set from validated JWT claims.
func JWTAuth(secret string, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Always strip trusted headers from incoming requests to prevent
			// spoofing, regardless of whether the route is public or protected.
			for _, h := range trustedHeaders {
				r.Header.Del(h)
			}

			// Allow public routes through without authentication.
			if isPublicRoute(r.Method, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract Bearer token from Authorization header.
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization header format")
				return
			}

			tokenString := parts[1]

			// Parse and validate the JWT token.
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
				// Ensure the signing method is HMAC.
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				logger.Warn("invalid JWT token",
					slog.String("path", r.URL.Path),
					slog.String("error", errString(err)),
				)
				writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
				return
			}

			// Extract claims and inject user context headers.
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token claims")
				return
			}

			// Extract user ID from "user_id" claim, falling back to "sub".
			userID := claimString(claims, "user_id")
			if userID == "" {
				userID = claimString(claims, "sub")
			}

			if userID != "" {
				r.Header.Set("X-User-ID", userID)
			}

			// Forward email and role if present in the token.
			if email := claimString(claims, "email"); email != "" {
				r.Header.Set("X-User-Email", email)
			}
			if role := claimString(claims, "role"); role != "" {
				r.Header.Set("X-User-Role", role)
			}

			logger.Debug("JWT authenticated request",
				slog.String("user_id", userID),
				slog.String("path", r.URL.Path),
				slog.String("method", r.Method),
			)

			next.ServeHTTP(w, r)
		})
	}
}

// claimString extracts a claim value as a string.
// It handles both string and numeric (float64) claim values,
// since JWT JSON decoding may represent numeric IDs as float64.
func claimString(claims jwt.MapClaims, key string) string {
	val, exists := claims[key]
	if !exists || val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case float64:
		// Handle numeric IDs (JSON numbers decode as float64).
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    code,
		"message": message,
	})
}

func errString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
