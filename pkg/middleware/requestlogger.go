package middleware

import (
	"log/slog"
	"net/http"

	"github.com/utafrali/EcommerceGo/pkg/logger"
)

// RequestLogger returns middleware that builds a request-scoped logger enriched
// with correlation_id, user_id, trace_id, and span_id, then stores it in
// context via logger.NewContext. Downstream handlers retrieve it with
// logger.FromContext(ctx).
//
// This middleware should be mounted AFTER RequestLogging (which sets
// correlation_id) and Tracing (which sets the OpenTelemetry span context).
func RequestLogger(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Pick up user_id from the auth middleware context key or the
			// X-User-ID header (used by services that don't run Auth middleware).
			userID := UserIDFromContext(ctx)
			if userID == "" {
				userID = r.Header.Get("X-User-ID")
			}
			if userID != "" {
				ctx = logger.WithUserID(ctx, userID)
			}

			// Build enriched logger with all available context fields
			// (correlation_id, user_id, trace_id, span_id).
			enriched := logger.WithContext(ctx, base)

			// Store the enriched logger in context for downstream handlers.
			ctx = logger.NewContext(ctx, enriched)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
