package logger

import (
	"context"
	"io"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const (
	correlationIDKey contextKey = "correlation_id"
	serviceKey       contextKey = "service"
	userIDKey        contextKey = "user_id"
	loggerKey        contextKey = "logger"
)

// New creates a new structured logger with the given service name and level.
func New(serviceName, level string) *slog.Logger {
	return NewWithWriter(serviceName, level, os.Stdout)
}

// NewWithWriter creates a new structured logger writing to the given writer.
func NewWithWriter(serviceName, level string, w io.Writer) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:     lvl,
		AddSource: lvl == slog.LevelDebug,
	})

	return slog.New(handler).With(
		slog.String("service", serviceName),
	)
}

// WithCorrelationID returns a new context with the correlation ID set.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

// CorrelationIDFromContext extracts the correlation ID from the context.
func CorrelationIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return id
	}
	return ""
}

// WithUserID returns a new context with the user ID set for logging.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// UserIDFromContext extracts the user ID stored by the logger package from context.
func UserIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey).(string); ok {
		return id
	}
	return ""
}

// NewContext returns a new context with the given logger stored in it.
func NewContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// FromContext returns the request-scoped logger stored in context.
// Returns slog.Default() if no logger is stored.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// WithContext returns a logger with context-derived fields (correlation_id, user_id, trace_id, span_id).
func WithContext(ctx context.Context, l *slog.Logger) *slog.Logger {
	if id := CorrelationIDFromContext(ctx); id != "" {
		l = l.With(slog.String("correlation_id", id))
	}

	if id := UserIDFromContext(ctx); id != "" {
		l = l.With(slog.String("user_id", id))
	}

	// Inject OpenTelemetry trace context if a valid span is present.
	if spanCtx := trace.SpanFromContext(ctx).SpanContext(); spanCtx.IsValid() {
		l = l.With(
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	return l
}
