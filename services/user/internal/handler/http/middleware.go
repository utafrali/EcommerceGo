package http

import (
	"net/http"
	"strings"
)

// ContentTypeJSON enforces that requests with a body have Content-Type: application/json.
func ContentTypeJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > 0 || r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			ct := r.Header.Get("Content-Type")
			if !strings.HasPrefix(ct, "application/json") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnsupportedMediaType)
				_, _ = w.Write([]byte(`{"error":{"code":"UNSUPPORTED_MEDIA_TYPE","message":"Content-Type must be application/json"}}`))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// CORSConfig holds configuration for the CORS middleware.
type CORSConfig struct {
	AllowedOrigins []string
	Environment    string
}

// CORS returns a middleware that sets Cross-Origin Resource Sharing headers.
// In development mode (or when AllowedOrigins contains "*"), a wildcard origin is used.
// In non-development modes, only the explicitly listed origins are allowed and the
// request Origin header is validated against the list.
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	// Build a lookup set for fast origin matching.
	allowWildcard := cfg.Environment == "development"
	originSet := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		if o == "*" {
			allowWildcard = true
		}
		originSet[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if allowWildcard {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				if _, ok := originSet[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Correlation-ID")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
