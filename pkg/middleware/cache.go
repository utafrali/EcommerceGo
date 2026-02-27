package middleware

import (
	"fmt"
	"net/http"
)

// CacheControl returns a middleware that sets Cache-Control header
func CacheControl(maxAge int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
			}
			next.ServeHTTP(w, r)
		})
	}
}
