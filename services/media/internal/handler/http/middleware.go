package http

import (
	"net/http"
	"strings"
)

// ContentTypeJSON enforces that requests with a body have Content-Type: application/json.
// Excludes multipart/form-data requests (used for file uploads).
func ContentTypeJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > 0 || r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			ct := r.Header.Get("Content-Type")
			if ct != "" && !strings.HasPrefix(ct, "application/json") && !strings.HasPrefix(ct, "multipart/form-data") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnsupportedMediaType)
				_, _ = w.Write([]byte(`{"error":{"code":"UNSUPPORTED_MEDIA_TYPE","message":"Content-Type must be application/json or multipart/form-data"}}`))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
