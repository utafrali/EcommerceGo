package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/utafrali/EcommerceGo/pkg/logger"
)

func newTestLogger(w *bytes.Buffer) *slog.Logger {
	return logger.NewWithWriter("test-svc", "info", w)
}

func TestRequestLogger_StoresLoggerInContext(t *testing.T) {
	var buf bytes.Buffer
	base := newTestLogger(&buf)

	var ctxLogger *slog.Logger
	handler := RequestLogger(base)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxLogger = logger.FromContext(r.Context())
		ctxLogger.Info("handler log")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if ctxLogger == nil {
		t.Fatal("expected non-nil logger from context")
	}

	if buf.Len() == 0 {
		t.Fatal("expected log output")
	}
}

func TestRequestLogger_IncludesCorrelationID(t *testing.T) {
	var buf bytes.Buffer
	base := newTestLogger(&buf)

	handler := RequestLogger(base)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.FromContext(r.Context()).Info("test")
		w.WriteHeader(http.StatusOK)
	}))

	// Set correlation_id in context (as RequestLogging middleware would).
	ctx := logger.WithCorrelationID(context.Background(), "corr-test-123")
	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := out["correlation_id"]; got != "corr-test-123" {
		t.Errorf("correlation_id = %v, want %q", got, "corr-test-123")
	}
}

func TestRequestLogger_IncludesUserIDFromAuthContext(t *testing.T) {
	var buf bytes.Buffer
	base := newTestLogger(&buf)

	handler := RequestLogger(base)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.FromContext(r.Context()).Info("test")
		w.WriteHeader(http.StatusOK)
	}))

	// Simulate auth middleware having set user_id in context.
	ctx := context.WithValue(context.Background(), userIDKey, "user-from-auth")
	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := out["user_id"]; got != "user-from-auth" {
		t.Errorf("user_id = %v, want %q", got, "user-from-auth")
	}
}

func TestRequestLogger_IncludesUserIDFromHeader(t *testing.T) {
	var buf bytes.Buffer
	base := newTestLogger(&buf)

	handler := RequestLogger(base)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.FromContext(r.Context()).Info("test")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-User-ID", "user-from-header")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := out["user_id"]; got != "user-from-header" {
		t.Errorf("user_id = %v, want %q", got, "user-from-header")
	}
}

func TestRequestLogger_AuthContextTakesPrecedenceOverHeader(t *testing.T) {
	var buf bytes.Buffer
	base := newTestLogger(&buf)

	handler := RequestLogger(base)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.FromContext(r.Context()).Info("test")
		w.WriteHeader(http.StatusOK)
	}))

	ctx := context.WithValue(context.Background(), userIDKey, "auth-user")
	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	req.Header.Set("X-User-ID", "header-user")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := out["user_id"]; got != "auth-user" {
		t.Errorf("user_id = %v, want %q (auth context should take precedence)", got, "auth-user")
	}
}

func TestRequestLogger_IncludesTraceFields(t *testing.T) {
	var buf bytes.Buffer
	base := newTestLogger(&buf)

	handler := RequestLogger(base)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.FromContext(r.Context()).Info("test")
		w.WriteHeader(http.StatusOK)
	}))

	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := out["trace_id"]; got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("trace_id = %v, want %q", got, "4bf92f3577b34da6a3ce929d0e0e4736")
	}
	if got := out["span_id"]; got != "00f067aa0ba902b7" {
		t.Errorf("span_id = %v, want %q", got, "00f067aa0ba902b7")
	}
}

func TestRequestLogger_NoUserID_OmitsField(t *testing.T) {
	var buf bytes.Buffer
	base := newTestLogger(&buf)

	handler := RequestLogger(base)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.FromContext(r.Context()).Info("test")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := out["user_id"]; ok {
		t.Error("user_id should not be present when not set")
	}
}
