package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
)

type contextKey string

const (
	correlationIDKey contextKey = "correlation_id"
	serviceKey       contextKey = "service"
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

// WithContext returns a logger with context-derived fields (correlation_id, etc.).
func WithContext(ctx context.Context, l *slog.Logger) *slog.Logger {
	if id := CorrelationIDFromContext(ctx); id != "" {
		l = l.With(slog.String("correlation_id", id))
	}
	return l
}
