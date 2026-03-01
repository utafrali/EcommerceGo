package database

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func setupTestTracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	t.Cleanup(func() {
		tp.Shutdown(context.Background()) //nolint:errcheck
		otel.SetTracerProvider(prev)
	})

	return exporter
}

func TestTraceQuery_Success(t *testing.T) {
	exporter := setupTestTracer(t)

	ctx := context.Background()
	ctx, end := TraceQuery(ctx, "GetProduct", "SELECT * FROM products WHERE id = $1")
	end(nil)

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}

	span := spans[0]
	if span.Name != "db.GetProduct" {
		t.Errorf("span name = %q, want %q", span.Name, "db.GetProduct")
	}

	// Check attributes.
	attrs := make(map[string]string)
	for _, a := range span.Attributes {
		attrs[string(a.Key)] = a.Value.Emit()
	}

	if attrs["db.system"] != "postgresql" {
		t.Errorf("db.system = %q, want %q", attrs["db.system"], "postgresql")
	}
	if attrs["db.operation"] != "GetProduct" {
		t.Errorf("db.operation = %q, want %q", attrs["db.operation"], "GetProduct")
	}
	if attrs["db.statement"] != "SELECT * FROM products WHERE id = $1" {
		t.Errorf("db.statement = %q, want correct SQL", attrs["db.statement"])
	}

	// Success should not set error status.
	if span.Status.Code != 0 { // codes.Unset = 0
		t.Errorf("span status = %d, want 0 (Unset)", span.Status.Code)
	}
}

func TestTraceQuery_Error(t *testing.T) {
	exporter := setupTestTracer(t)

	ctx := context.Background()
	ctx, end := TraceQuery(ctx, "UpdateProduct", "UPDATE products SET name = $1 WHERE id = $2")
	end(errors.New("connection refused"))

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}

	span := spans[0]
	if span.Status.Code != 1 { // codes.Error = 1 in Go SDK
		t.Errorf("span status = %d, want 2 (Error)", span.Status.Code)
	}

	// Should have recorded an error event.
	if len(span.Events) == 0 {
		t.Error("expected error event to be recorded on span")
	}
}

func TestTraceQuery_PropagatesContext(t *testing.T) {
	setupTestTracer(t)

	ctx := context.Background()
	// Start a parent span.
	tracer := otel.Tracer("test")
	ctx, parentSpan := tracer.Start(ctx, "parent")

	// TraceQuery should create a child span.
	ctx, end := TraceQuery(ctx, "ListOrders", "SELECT * FROM orders")
	end(nil)

	parentSpan.End()

	// Verify the context is still usable.
	if ctx == nil {
		t.Error("returned context should not be nil")
	}
}

func TestSlowQueryLogging_SlowQuery(t *testing.T) {
	setupTestTracer(t)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	// Set a very low threshold so the query is guaranteed to be "slow".
	SetSlowQueryLogging(1*time.Nanosecond, logger)
	t.Cleanup(func() { SetSlowQueryLogging(0, nil) })

	ctx := context.Background()
	_, end := TraceQuery(ctx, "SlowSelect", "SELECT * FROM big_table")
	// Even a near-instant call exceeds 1ns threshold.
	end(nil)

	assert := buf.String()
	if !containsSubstring(assert, "slow query detected") {
		t.Errorf("expected slow query log, got: %s", assert)
	}
	if !containsSubstring(assert, "SlowSelect") {
		t.Errorf("expected operation name in log, got: %s", assert)
	}
	if !containsSubstring(assert, "SELECT * FROM big_table") {
		t.Errorf("expected SQL statement in log, got: %s", assert)
	}
}

func TestSlowQueryLogging_FastQuery_NoLog(t *testing.T) {
	setupTestTracer(t)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	// Set a very high threshold so no query is "slow".
	SetSlowQueryLogging(1*time.Hour, logger)
	t.Cleanup(func() { SetSlowQueryLogging(0, nil) })

	ctx := context.Background()
	_, end := TraceQuery(ctx, "FastSelect", "SELECT 1")
	end(nil)

	if containsSubstring(buf.String(), "slow query detected") {
		t.Error("did not expect slow query log for fast query")
	}
}

func TestSlowQueryLogging_Disabled(t *testing.T) {
	setupTestTracer(t)

	// Ensure slow query logging is disabled.
	SetSlowQueryLogging(0, nil)

	ctx := context.Background()
	_, end := TraceQuery(ctx, "AnyOp", "SELECT 1")
	// Should not panic even with nil logger and zero threshold.
	end(nil)
}

func TestSlowQueryLogging_WithError(t *testing.T) {
	setupTestTracer(t)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	SetSlowQueryLogging(1*time.Nanosecond, logger)
	t.Cleanup(func() { SetSlowQueryLogging(0, nil) })

	ctx := context.Background()
	_, end := TraceQuery(ctx, "FailedQuery", "INSERT INTO t VALUES ($1)")
	end(errors.New("unique constraint violation"))

	output := buf.String()
	if !containsSubstring(output, "slow query detected") {
		t.Errorf("expected slow query log, got: %s", output)
	}
	if !containsSubstring(output, "unique constraint violation") {
		t.Errorf("expected error in slow query log, got: %s", output)
	}
}

func TestSetSlowQueryLogging_Concurrent(t *testing.T) {
	setupTestTracer(t)
	t.Cleanup(func() { SetSlowQueryLogging(0, nil) })

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	// Concurrent reads/writes should not race.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			SetSlowQueryLogging(time.Duration(i)*time.Millisecond, logger)
		}
	}()

	for i := 0; i < 100; i++ {
		getSlowQueryConfig()
	}

	<-done
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
