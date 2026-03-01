package tracing

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestInitTracer_Disabled(t *testing.T) {
	cfg := DefaultConfig("test-service")
	cfg.Enabled = false

	shutdown, err := InitTracer(context.Background(), cfg)
	if err != nil {
		t.Fatalf("InitTracer(disabled) returned error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown function should not be nil even when disabled")
	}

	// Calling shutdown should be a no-op and return nil.
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown(disabled) returned error: %v", err)
	}
}

func TestInitTracer_Enabled(t *testing.T) {
	// Use a non-routable endpoint so the exporter doesn't actually connect,
	// but the SDK initializes successfully (batched export is async).
	cfg := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "127.0.0.1:0",
		SampleRate:     1.0,
		Enabled:        true,
	}

	shutdown, err := InitTracer(context.Background(), cfg)
	if err != nil {
		t.Fatalf("InitTracer(enabled) returned error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown function should not be nil")
	}

	// Verify the global tracer provider was set to an SDK provider.
	tp := otel.GetTracerProvider()
	if _, ok := tp.(*sdktrace.TracerProvider); !ok {
		t.Errorf("expected *sdktrace.TracerProvider, got %T", tp)
	}

	// Clean up.
	if err := shutdown(context.Background()); err != nil {
		t.Logf("shutdown returned (expected due to unreachable endpoint): %v", err)
	}
}

func TestInitTracer_SampleRateZero(t *testing.T) {
	cfg := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "127.0.0.1:0",
		SampleRate:     0.0,
		Enabled:        true,
	}

	shutdown, err := InitTracer(context.Background(), cfg)
	if err != nil {
		t.Fatalf("InitTracer(sample=0) returned error: %v", err)
	}
	defer shutdown(context.Background()) //nolint:errcheck
}

func TestInitTracer_SampleRatePartial(t *testing.T) {
	cfg := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "127.0.0.1:0",
		SampleRate:     0.5,
		Enabled:        true,
	}

	shutdown, err := InitTracer(context.Background(), cfg)
	if err != nil {
		t.Fatalf("InitTracer(sample=0.5) returned error: %v", err)
	}
	defer shutdown(context.Background()) //nolint:errcheck
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("my-service")

	if cfg.ServiceName != "my-service" {
		t.Errorf("ServiceName = %q, want %q", cfg.ServiceName, "my-service")
	}
	if cfg.Enabled {
		t.Error("default config should have Enabled = false")
	}
	if cfg.SampleRate != 1.0 {
		t.Errorf("SampleRate = %f, want 1.0", cfg.SampleRate)
	}
	if cfg.OTLPEndpoint != "localhost:4318" {
		t.Errorf("OTLPEndpoint = %q, want %q", cfg.OTLPEndpoint, "localhost:4318")
	}
}

func TestTracer(t *testing.T) {
	tracer := Tracer("test-component")
	if tracer == nil {
		t.Fatal("Tracer should not return nil")
	}

	// Start a span to verify the tracer works.
	_, span := tracer.Start(context.Background(), "test-op")
	defer span.End()

	if !span.SpanContext().IsValid() || !span.IsRecording() {
		// When no SDK is configured, the span may be a no-op span.
		// This is acceptable - we just verify it doesn't panic.
		t.Log("span is no-op (expected when no SDK provider is set)")
	}
}
