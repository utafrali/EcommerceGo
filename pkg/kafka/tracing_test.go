package kafka

import (
	"testing"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func TestKafkaHeaderCarrier_SetAndGet(t *testing.T) {
	headers := []kafka.Header{
		{Key: "existing", Value: []byte("value1")},
	}
	carrier := &KafkaHeaderCarrier{headers: &headers}

	// Get existing header.
	if got := carrier.Get("existing"); got != "value1" {
		t.Errorf("Get(existing) = %q, want %q", got, "value1")
	}

	// Get non-existing header.
	if got := carrier.Get("missing"); got != "" {
		t.Errorf("Get(missing) = %q, want empty", got)
	}

	// Set a new header.
	carrier.Set("new-key", "new-value")
	if got := carrier.Get("new-key"); got != "new-value" {
		t.Errorf("Get(new-key) = %q, want %q", got, "new-value")
	}

	// Overwrite existing header.
	carrier.Set("existing", "updated")
	if got := carrier.Get("existing"); got != "updated" {
		t.Errorf("Get(existing) after update = %q, want %q", got, "updated")
	}
}

func TestKafkaHeaderCarrier_Keys(t *testing.T) {
	headers := []kafka.Header{
		{Key: "a", Value: []byte("1")},
		{Key: "b", Value: []byte("2")},
		{Key: "c", Value: []byte("3")},
	}
	carrier := &KafkaHeaderCarrier{headers: &headers}

	keys := carrier.Keys()
	if len(keys) != 3 {
		t.Fatalf("Keys() returned %d keys, want 3", len(keys))
	}

	expected := map[string]bool{"a": true, "b": true, "c": true}
	for _, k := range keys {
		if !expected[k] {
			t.Errorf("unexpected key: %q", k)
		}
	}
}

func TestKafkaHeaderCarrier_PropagationRoundTrip(t *testing.T) {
	// Set up W3C trace context propagator.
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(propagator)

	headers := []kafka.Header{}
	carrier := &KafkaHeaderCarrier{headers: &headers}

	// Inject a known traceparent.
	carrier.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	// Verify we can read it back.
	got := carrier.Get("traceparent")
	if got != "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" {
		t.Errorf("traceparent = %q, want full W3C trace context", got)
	}
}

func TestKafkaHeaderCarrier_EmptyHeaders(t *testing.T) {
	headers := []kafka.Header{}
	carrier := &KafkaHeaderCarrier{headers: &headers}

	keys := carrier.Keys()
	if len(keys) != 0 {
		t.Errorf("Keys() on empty headers = %d, want 0", len(keys))
	}

	if got := carrier.Get("anything"); got != "" {
		t.Errorf("Get on empty headers = %q, want empty", got)
	}
}
