package database

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/utafrali/EcommerceGo/pkg/database"

// slowQueryConfig holds the configurable slow query logging settings.
var slowQueryCfg struct {
	mu        sync.RWMutex
	threshold time.Duration
	logger    *slog.Logger
}

// SetSlowQueryLogging configures slow query detection. Queries exceeding the
// threshold are logged as warnings with operation name, SQL statement, and
// duration. A zero threshold disables slow query logging.
func SetSlowQueryLogging(threshold time.Duration, logger *slog.Logger) {
	slowQueryCfg.mu.Lock()
	defer slowQueryCfg.mu.Unlock()
	slowQueryCfg.threshold = threshold
	slowQueryCfg.logger = logger
}

// getSlowQueryConfig returns the current slow query threshold and logger.
func getSlowQueryConfig() (time.Duration, *slog.Logger) {
	slowQueryCfg.mu.RLock()
	defer slowQueryCfg.mu.RUnlock()
	return slowQueryCfg.threshold, slowQueryCfg.logger
}

// TraceQuery starts a span for a database operation. The returned function
// must be called when the operation completes (typically via defer):
//
//	ctx, end := database.TraceQuery(ctx, "GetProduct", "SELECT * FROM products WHERE id = $1")
//	defer func() { end(err) }()
//
// If slow query logging is enabled via SetSlowQueryLogging, queries exceeding
// the configured threshold are logged as warnings.
func TraceQuery(ctx context.Context, operation, statement string) (context.Context, func(error)) {
	start := time.Now()
	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, "db."+operation,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation", operation),
			attribute.String("db.statement", statement),
		),
	)

	return ctx, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()

		// Slow query logging.
		if threshold, logger := getSlowQueryConfig(); threshold > 0 && logger != nil {
			if elapsed := time.Since(start); elapsed >= threshold {
				attrs := []any{
					slog.String("operation", operation),
					slog.String("statement", statement),
					slog.Duration("duration", elapsed),
				}
				if err != nil {
					attrs = append(attrs, slog.String("error", err.Error()))
				}
				logger.WarnContext(ctx, "slow query detected", attrs...)
			}
		}
	}
}
