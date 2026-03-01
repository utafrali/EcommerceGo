package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// setupTestTracer installs an in-memory span exporter and returns it along
// with a cleanup function that restores the previous global tracer provider.
func setupTestTracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	t.Cleanup(func() {
		tp.Shutdown(context.Background()) //nolint:errcheck
		otel.SetTracerProvider(prev)
	})

	return exporter
}

func TestTracing_CreatesSpan(t *testing.T) {
	exporter := setupTestTracer(t)

	r := chi.NewRouter()
	r.Use(Tracing("test-service"))
	r.Get("/api/v1/products", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span, got none")
	}

	span := spans[0]
	if span.Name != "GET /api/v1/products" {
		t.Errorf("span name = %q, want %q", span.Name, "GET /api/v1/products")
	}
}

func TestTracing_RecordsStatusCode(t *testing.T) {
	exporter := setupTestTracer(t)

	r := chi.NewRouter()
	r.Use(Tracing("test-service"))
	r.Get("/not-found", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}

	// Verify the span has an http.status_code attribute.
	found := false
	for _, attr := range spans[0].Attributes {
		if string(attr.Key) == "http.status_code" {
			if attr.Value.AsInt64() != 404 {
				t.Errorf("http.status_code = %d, want 404", attr.Value.AsInt64())
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("http.status_code attribute not found on span")
	}
}

func TestTracing_ServerError_SetsSpanError(t *testing.T) {
	exporter := setupTestTracer(t)

	r := chi.NewRouter()
	r.Use(Tracing("test-service"))
	r.Get("/error", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}

	span := spans[0]
	if span.Status.Code != 1 { // codes.Error = 1 in Go SDK
		t.Errorf("span status code = %d, want 2 (Error)", span.Status.Code)
	}
}

func TestTracing_PropagatesTraceContext(t *testing.T) {
	exporter := setupTestTracer(t)

	r := chi.NewRouter()
	r.Use(Tracing("test-service"))
	r.Get("/traced", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Send a request with an existing trace context.
	req := httptest.NewRequest(http.MethodGet, "/traced", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}

	// The span should have the parent trace ID from the incoming header.
	traceID := spans[0].SpanContext.TraceID().String()
	if traceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("trace ID = %s, want 4bf92f3577b34da6a3ce929d0e0e4736", traceID)
	}

	// The response should include a traceparent header.
	if tp := rec.Header().Get("traceparent"); tp == "" {
		t.Error("response missing traceparent header")
	}
}

func TestTracing_InjectsResponseHeaders(t *testing.T) {
	setupTestTracer(t)

	r := chi.NewRouter()
	r.Use(Tracing("test-service"))
	r.Get("/inject", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/inject", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// The middleware should inject traceparent into response headers.
	if tp := rec.Header().Get("traceparent"); tp == "" {
		t.Error("response missing traceparent header")
	}
}
