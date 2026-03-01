package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestWithContext_CorrelationID(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter("test", "info", &buf)

	ctx := WithCorrelationID(context.Background(), "req-123")
	cl := WithContext(ctx, l)
	cl.Info("hello")

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal log line: %v", err)
	}
	if got := out["correlation_id"]; got != "req-123" {
		t.Errorf("correlation_id = %v, want %q", got, "req-123")
	}
}

func TestWithContext_NoSpan(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter("test", "info", &buf)

	cl := WithContext(context.Background(), l)
	cl.Info("no span")

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal log line: %v", err)
	}
	if _, ok := out["trace_id"]; ok {
		t.Error("trace_id should not be present when no span in context")
	}
	if _, ok := out["span_id"]; ok {
		t.Error("span_id should not be present when no span in context")
	}
}

func TestWithContext_WithValidSpan(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter("test", "info", &buf)

	// Create a valid span context with known IDs.
	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	cl := WithContext(ctx, l)
	cl.Info("with span")

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal log line: %v", err)
	}
	if got := out["trace_id"]; got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("trace_id = %v, want %q", got, "4bf92f3577b34da6a3ce929d0e0e4736")
	}
	if got := out["span_id"]; got != "00f067aa0ba902b7" {
		t.Errorf("span_id = %v, want %q", got, "00f067aa0ba902b7")
	}
}

func TestWithContext_CorrelationAndTrace(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter("test", "info", &buf)

	traceID, _ := trace.TraceIDFromHex("abcdef1234567890abcdef1234567890")
	spanID, _ := trace.SpanIDFromHex("1234567890abcdef")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	ctx = WithCorrelationID(ctx, "corr-456")

	cl := WithContext(ctx, l)
	cl.Info("both")

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal log line: %v", err)
	}
	if got := out["correlation_id"]; got != "corr-456" {
		t.Errorf("correlation_id = %v, want %q", got, "corr-456")
	}
	if got := out["trace_id"]; got != "abcdef1234567890abcdef1234567890" {
		t.Errorf("trace_id = %v, want %q", got, "abcdef1234567890abcdef1234567890")
	}
	if got := out["span_id"]; got != "1234567890abcdef" {
		t.Errorf("span_id = %v, want %q", got, "1234567890abcdef")
	}
}

func TestWithContext_UserID(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter("test", "info", &buf)

	ctx := WithUserID(context.Background(), "user-789")
	cl := WithContext(ctx, l)
	cl.Info("with user")

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal log line: %v", err)
	}
	if got := out["user_id"]; got != "user-789" {
		t.Errorf("user_id = %v, want %q", got, "user-789")
	}
}

func TestWithContext_NoUserID(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter("test", "info", &buf)

	cl := WithContext(context.Background(), l)
	cl.Info("no user")

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal log line: %v", err)
	}
	if _, ok := out["user_id"]; ok {
		t.Error("user_id should not be present when not in context")
	}
}

func TestFromContext_WithLogger(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter("test", "info", &buf)

	ctx := NewContext(context.Background(), l)
	got := FromContext(ctx)
	if got != l {
		t.Error("FromContext should return the logger stored via NewContext")
	}
}

func TestFromContext_WithoutLogger(t *testing.T) {
	got := FromContext(context.Background())
	if got == nil {
		t.Error("FromContext should return a non-nil fallback logger")
	}
}

func TestWithContext_AllFields(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter("test", "info", &buf)

	traceID, _ := trace.TraceIDFromHex("abcdef1234567890abcdef1234567890")
	spanID, _ := trace.SpanIDFromHex("1234567890abcdef")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	ctx = WithCorrelationID(ctx, "corr-all")
	ctx = WithUserID(ctx, "user-all")

	cl := WithContext(ctx, l)
	cl.Info("all fields")

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal log line: %v", err)
	}
	if got := out["correlation_id"]; got != "corr-all" {
		t.Errorf("correlation_id = %v, want %q", got, "corr-all")
	}
	if got := out["user_id"]; got != "user-all" {
		t.Errorf("user_id = %v, want %q", got, "user-all")
	}
	if got := out["trace_id"]; got != "abcdef1234567890abcdef1234567890" {
		t.Errorf("trace_id = %v, want %q", got, "abcdef1234567890abcdef1234567890")
	}
	if got := out["span_id"]; got != "1234567890abcdef" {
		t.Errorf("span_id = %v, want %q", got, "1234567890abcdef")
	}
}
