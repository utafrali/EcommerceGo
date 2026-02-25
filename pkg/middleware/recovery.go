package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery recovers from panics and returns a 500 error instead of crashing.
func Recovery(l *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					l.ErrorContext(r.Context(), "panic recovered",
						slog.Any("panic", rec),
						slog.String("stack", string(debug.Stack())),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
					)

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					if err := json.NewEncoder(w).Encode(map[string]string{
						"code":    "INTERNAL_ERROR",
						"message": "an internal error occurred",
					}); err != nil {
						l.Error("failed to encode response", slog.String("error", err.Error()))
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
