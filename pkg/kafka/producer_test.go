package kafka

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Event tests ---

func TestNewEvent_Fields(t *testing.T) {
	type OrderData struct {
		OrderID string `json:"order_id"`
		Amount  int64  `json:"amount"`
	}

	data := OrderData{OrderID: "ord-123", Amount: 4999}
	event, err := NewEvent("order.created", "ord-123", "order", "order-service", data)
	require.NoError(t, err)

	assert.NotEmpty(t, event.EventID, "EventID should be a non-empty UUID")
	assert.Equal(t, "order.created", event.EventType)
	assert.Equal(t, "ord-123", event.AggregateID)
	assert.Equal(t, "order", event.AggregateType)
	assert.Equal(t, "order-service", event.Source)
	assert.Equal(t, 1, event.Version)
	assert.WithinDuration(t, time.Now().UTC(), event.Timestamp, 2*time.Second)
	assert.NotNil(t, event.Metadata)
	assert.NotNil(t, event.Data)

	// Verify the data was marshaled correctly.
	var roundTripped OrderData
	require.NoError(t, json.Unmarshal(event.Data, &roundTripped))
	assert.Equal(t, data, roundTripped)
}

func TestNewEvent_InvalidData(t *testing.T) {
	// Channels are not serializable to JSON.
	_, err := NewEvent("test.event", "agg-1", "test", "test-service", make(chan int))
	require.Error(t, err)
}

func TestEvent_Marshal_Unmarshal(t *testing.T) {
	original, err := NewEvent("product.updated", "prod-456", "product", "product-service", map[string]string{"name": "Widget"})
	require.NoError(t, err)
	original.CorrelationID = "corr-abc"
	original.Metadata["user"] = "admin"

	bytes, err := original.Marshal()
	require.NoError(t, err)
	assert.NotEmpty(t, bytes)

	restored, err := UnmarshalEvent(bytes)
	require.NoError(t, err)

	assert.Equal(t, original.EventID, restored.EventID)
	assert.Equal(t, original.EventType, restored.EventType)
	assert.Equal(t, original.AggregateID, restored.AggregateID)
	assert.Equal(t, original.AggregateType, restored.AggregateType)
	assert.Equal(t, original.Version, restored.Version)
	assert.Equal(t, original.Source, restored.Source)
	assert.Equal(t, original.CorrelationID, restored.CorrelationID)
	assert.Equal(t, original.Metadata, restored.Metadata)
	assert.JSONEq(t, string(original.Data), string(restored.Data))
	assert.WithinDuration(t, original.Timestamp, restored.Timestamp, time.Millisecond)
}

func TestEvent_WithCorrelationID(t *testing.T) {
	event, err := NewEvent("test.event", "agg-1", "test", "svc", nil)
	require.NoError(t, err)

	// Verify chaining returns the same pointer.
	result := event.WithCorrelationID("corr-xyz")
	assert.Same(t, event, result, "WithCorrelationID should return the same event for chaining")
	assert.Equal(t, "corr-xyz", event.CorrelationID)
}

func TestEvent_WithMetadata(t *testing.T) {
	event, err := NewEvent("test.event", "agg-1", "test", "svc", nil)
	require.NoError(t, err)

	result := event.WithMetadata("key1", "value1").WithMetadata("key2", "value2")
	assert.Same(t, event, result, "WithMetadata should return the same event for chaining")
	assert.Equal(t, "value1", event.Metadata["key1"])
	assert.Equal(t, "value2", event.Metadata["key2"])
}

func TestEvent_WithMetadata_NilMetadataMap(t *testing.T) {
	// Manually create an event with nil Metadata to test initialization.
	event := &Event{
		EventID:   "test-id",
		EventType: "test",
		Metadata:  nil,
	}
	event.WithMetadata("key", "value")
	assert.NotNil(t, event.Metadata)
	assert.Equal(t, "value", event.Metadata["key"])
}

func TestEvent_UnmarshalData(t *testing.T) {
	type ProductPayload struct {
		Name  string `json:"name"`
		Price int64  `json:"price"`
	}

	payload := ProductPayload{Name: "Sneakers", Price: 7999}
	event, err := NewEvent("product.created", "prod-1", "product", "product-service", payload)
	require.NoError(t, err)

	var target ProductPayload
	require.NoError(t, event.UnmarshalData(&target))
	assert.Equal(t, payload.Name, target.Name)
	assert.Equal(t, payload.Price, target.Price)
}

func TestEvent_UnmarshalData_Invalid(t *testing.T) {
	event := &Event{
		Data: json.RawMessage(`not valid json`),
	}
	var target map[string]string
	err := event.UnmarshalData(&target)
	require.Error(t, err)
}

func TestUnmarshalEvent_InvalidJSON(t *testing.T) {
	_, err := UnmarshalEvent([]byte(`{broken json`))
	require.Error(t, err)
}

func TestUnmarshalEvent_EmptyBytes(t *testing.T) {
	_, err := UnmarshalEvent([]byte{})
	require.Error(t, err)
}

// --- ProducerConfig tests ---

func TestDefaultProducerConfig(t *testing.T) {
	brokers := []string{"broker1:9092", "broker2:9092"}
	cfg := DefaultProducerConfig(brokers)

	assert.Equal(t, brokers, cfg.Brokers)
	assert.Equal(t, 100, cfg.BatchSize)
	assert.Equal(t, 10*time.Millisecond, cfg.BatchTimeout)
	assert.False(t, cfg.Async)
}

func TestDefaultProducerConfig_SingleBroker(t *testing.T) {
	cfg := DefaultProducerConfig([]string{"localhost:9092"})
	assert.Len(t, cfg.Brokers, 1)
	assert.Equal(t, "localhost:9092", cfg.Brokers[0])
}

// --- Topic tests ---

func TestTopic_Format(t *testing.T) {
	got := Topic("order", "confirmed")
	assert.Equal(t, "ecommerce.order.confirmed", got)
}

func TestTopic_Prefix(t *testing.T) {
	assert.Equal(t, "ecommerce", TopicPrefix)
}

func TestTopic_VariousCombinations(t *testing.T) {
	tests := []struct {
		domain string
		action string
		want   string
	}{
		{"order", "created", "ecommerce.order.created"},
		{"payment", "completed", "ecommerce.payment.completed"},
		{"inventory", "reserved", "ecommerce.inventory.reserved"},
		{"cart", "item-added", "ecommerce.cart.item-added"},
	}

	for _, tt := range tests {
		t.Run(tt.domain+"."+tt.action, func(t *testing.T) {
			assert.Equal(t, tt.want, Topic(tt.domain, tt.action))
		})
	}
}

// --- KafkaHeaderCarrier additional tests ---

func TestNewProducer_CreatesInstance(t *testing.T) {
	// NewProducer requires broker addresses but does not connect immediately.
	// We verify the returned producer is non-nil and can be closed.
	cfg := DefaultProducerConfig([]string{"localhost:19092"})
	p := NewProducer(cfg, nil)
	require.NotNil(t, p)
	assert.Equal(t, []string{"localhost:19092"}, p.brokers)

	// Close should succeed even without a real broker.
	err := p.Close()
	assert.NoError(t, err)
}

func TestPingBrokers_NoBrokers(t *testing.T) {
	err := PingBrokers(t.Context(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no brokers configured")
}

func TestPingBrokers_EmptySlice(t *testing.T) {
	err := PingBrokers(t.Context(), []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no brokers configured")
}
