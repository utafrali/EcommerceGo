package http

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is an unexported type for context keys to prevent collisions.
type contextKey string

// userIDKey is the context key for the authenticated user ID.
const userIDKey contextKey = "user_id"

// UserIDFromHeader is middleware that reads the X-User-ID header (injected by
// the API gateway after JWT validation) and stores it in the request context.
// If the header is absent the request is rejected with 401 Unauthorized.
func UserIDFromHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := r.Header.Get("X-User-ID")
		if uid == "" {
			writeJSON(w, http.StatusUnauthorized, response{
				Error: &errorResponse{Code: "UNAUTHORIZED", Message: "authentication required"},
			})
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// userIDFromContext extracts the authenticated user ID from the request context.
// Returns the user ID and true if present, or empty string and false otherwise.
func userIDFromContext(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(userIDKey).(string)
	return uid, ok && uid != ""
}

// ContentTypeJSON enforces that requests with a body have Content-Type: application/json.
func ContentTypeJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > 0 || r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			ct := r.Header.Get("Content-Type")
			if ct != "" && !strings.HasPrefix(ct, "application/json") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnsupportedMediaType)
				_, _ = w.Write([]byte(`{"error":{"code":"UNSUPPORTED_MEDIA_TYPE","message":"Content-Type must be application/json"}}`))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// CORS adds permissive Cross-Origin Resource Sharing headers for development.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Correlation-ID, X-User-ID")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
