package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig holds configuration for the CORS middleware.
type CORSConfig struct {
	// AllowedOrigins is the list of allowed origins (e.g. "https://example.com").
	// If it contains "*", all origins are allowed (only safe in development).
	AllowedOrigins []string

	// AllowedMethods is the list of allowed HTTP methods.
	// Defaults to GET, POST, PUT, PATCH, DELETE, OPTIONS if empty.
	AllowedMethods []string

	// AllowedHeaders is the list of allowed request headers.
	// Defaults to Accept, Authorization, Content-Type, X-Correlation-ID, X-User-ID if empty.
	AllowedHeaders []string

	// ExposedHeaders is the list of headers the browser may access.
	ExposedHeaders []string

	// MaxAge is how long (in seconds) preflight results can be cached.
	// Defaults to 3600 if 0.
	MaxAge int

	// AllowCredentials indicates whether credentials (cookies, auth headers) are supported.
	AllowCredentials bool

	// Environment controls wildcard behavior. Wildcard origins are only
	// accepted when Environment is "development" or AllowedOrigins explicitly contains "*".
	Environment string
}

// DefaultCORSConfig returns a restrictive default CORS configuration
// suitable for development.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Correlation-ID", "X-User-ID"},
		ExposedHeaders: []string{"X-Correlation-ID", "X-User-ID"},
		MaxAge:         3600,
		Environment:    "development",
	}
}

// CORS returns middleware that handles Cross-Origin Resource Sharing headers
// based on the provided configuration.
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	// Apply defaults for empty fields.
	if len(cfg.AllowedMethods) == 0 {
		cfg.AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if len(cfg.AllowedHeaders) == 0 {
		cfg.AllowedHeaders = []string{"Accept", "Authorization", "Content-Type", "X-Correlation-ID", "X-User-ID"}
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 3600
	}

	// Determine if wildcard is allowed.
	allowWildcard := cfg.Environment == "development"
	originSet := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		if o == "*" {
			allowWildcard = true
		}
		originSet[o] = struct{}{}
	}

	methods := strings.Join(cfg.AllowedMethods, ", ")
	headers := strings.Join(cfg.AllowedHeaders, ", ")
	exposed := strings.Join(cfg.ExposedHeaders, ", ")
	maxAge := strconv.Itoa(cfg.MaxAge)

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

			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", headers)
			if exposed != "" {
				w.Header().Set("Access-Control-Expose-Headers", exposed)
			}
			w.Header().Set("Access-Control-Max-Age", maxAge)

			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
