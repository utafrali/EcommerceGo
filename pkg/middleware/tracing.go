package middleware

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// tracingResponseWriter wraps http.ResponseWriter to capture the status code.
type tracingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (w *tracingResponseWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *tracingResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.statusCode = http.StatusOK
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}

// Tracing returns middleware that creates OpenTelemetry spans for incoming HTTP
// requests. It extracts W3C trace context from inbound headers and records the
// route pattern, method, status code, and user agent as span attributes.
func Tracing(serviceName string) func(http.Handler) http.Handler {
	tracer := otel.Tracer(fmt.Sprintf("github.com/utafrali/EcommerceGo/services/%s", serviceName))
	propagator := otel.GetTextMapPropagator()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract trace context from incoming headers.
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// Determine route pattern for the span name (chi populates this
			// after routing, so we use the raw path as fallback here and
			// update it after the handler runs).
			spanName := r.Method + " " + r.URL.Path

			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPMethod(r.Method),
					semconv.HTTPTarget(r.URL.RequestURI()),
					semconv.HTTPScheme(scheme(r)),
					semconv.UserAgentOriginal(r.UserAgent()),
					attribute.String("http.client_ip", r.RemoteAddr),
				),
			)
			defer span.End()

			// Wrap the response writer to capture the status code.
			trw := &tracingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Inject trace context into response headers so downstream
			// clients can correlate.
			propagator.Inject(ctx, propagation.HeaderCarrier(w.Header()))

			// Serve the request with the traced context.
			next.ServeHTTP(trw, r.WithContext(ctx))

			// Update span name with the actual chi route pattern.
			if routeCtx := chi.RouteContext(r.Context()); routeCtx != nil {
				if pattern := routeCtx.RoutePattern(); pattern != "" {
					span.SetName(r.Method + " " + pattern)
					span.SetAttributes(attribute.String("http.route", pattern))
				}
			}

			// Record status code.
			span.SetAttributes(semconv.HTTPStatusCode(trw.statusCode))

			if trw.statusCode >= 500 {
				span.SetStatus(codes.Error, http.StatusText(trw.statusCode))
			}
		})
	}
}

// scheme returns "https" if the request uses TLS, otherwise "http".
func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	return "http"
}
