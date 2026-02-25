package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/utafrali/EcommerceGo/pkg/logger"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += n
	return n, err
}

// RequestLogging logs HTTP requests with duration, status, and correlation ID.
func RequestLogging(l *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Extract or generate correlation ID
			correlationID := r.Header.Get("X-Correlation-ID")
			if correlationID == "" {
				correlationID = uuid.New().String()
			}

			ctx := logger.WithCorrelationID(r.Context(), correlationID)
			r = r.WithContext(ctx)

			// Set correlation ID in response header
			w.Header().Set("X-Correlation-ID", correlationID)

			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			l.InfoContext(ctx, "http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", wrapped.statusCode),
				slog.Duration("duration", duration),
				slog.Int("bytes", wrapped.bytes),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
				slog.String("correlation_id", correlationID),
			)
		})
	}
}
