package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/utafrali/EcommerceGo/pkg/httputil"
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
			httputil.WriteJSON(w, http.StatusUnauthorized, httputil.Response{
				Error: &httputil.ErrorResponse{Code: "UNAUTHORIZED", Message: "authentication required"},
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
