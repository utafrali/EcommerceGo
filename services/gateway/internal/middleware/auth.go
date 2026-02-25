package middleware

import (
	"encoding/json"
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
	{method: http.MethodGet, prefix: "/api/v1/search"},
	{method: http.MethodPost, prefix: "/api/v1/auth"},
	{method: http.MethodGet, prefix: "/health"},
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
// For protected routes, the token is validated and the X-User-ID header is injected
// into the proxied request.
func JWTAuth(secret string, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

			// Extract claims and inject X-User-ID header.
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token claims")
				return
			}

			userID, _ := claims["user_id"].(string)
			if userID == "" {
				// Fallback: try "sub" claim.
				userID, _ = claims["sub"].(string)
			}

			if userID != "" {
				r.Header.Set("X-User-ID", userID)
			}

			next.ServeHTTP(w, r)
		})
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
