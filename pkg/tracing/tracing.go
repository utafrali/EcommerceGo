package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds OpenTelemetry tracing configuration.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string  // e.g. "localhost:4318" for OTLP/HTTP
	SampleRate     float64 // 0.0 to 1.0; 1.0 = sample everything
	Enabled        bool
}

// DefaultConfig returns sensible defaults for development.
func DefaultConfig(serviceName string) Config {
	return Config{
		ServiceName:    serviceName,
		ServiceVersion: "0.1.0",
		Environment:    "development",
		OTLPEndpoint:   "localhost:4318",
		SampleRate:     1.0,
		Enabled:        false,
	}
}

// InitTracer initializes the OpenTelemetry trace provider with an OTLP/HTTP
// exporter. It sets the global tracer provider and text map propagator.
// The returned shutdown function must be called on application exit to flush
// pending spans.
func InitTracer(ctx context.Context, cfg Config) (shutdown func(context.Context) error, err error) {
	if !cfg.Enabled {
		// Return a no-op shutdown when tracing is disabled.
		return func(context.Context) error { return nil }, nil
	}

	// Build the OTLP/HTTP exporter.
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create OTLP exporter: %w", err)
	}

	// Build the resource with service metadata.
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
		resource.WithProcessRuntimeDescription(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	// Choose sampler based on sample rate.
	var sampler sdktrace.Sampler
	switch cfg.SampleRate {
	case 1.0:
		sampler = sdktrace.AlwaysSample()
	case 0.0:
		sampler = sdktrace.NeverSample()
	default:
		sampler = sdktrace.TraceIDRatioBased(cfg.SampleRate)
	}

	// Create and register the tracer provider.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// Tracer returns a named tracer from the global provider. Use this in
// application code to create spans:
//
//	tracer := tracing.Tracer("myservice")
//	ctx, span := tracer.Start(ctx, "operation")
//	defer span.End()
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
