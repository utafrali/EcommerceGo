package kafka

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Event represents the standard event envelope for all Kafka messages.
type Event struct {
	EventID       string            `json:"event_id"`
	EventType     string            `json:"event_type"`
	AggregateID   string            `json:"aggregate_id"`
	AggregateType string            `json:"aggregate_type"`
	Version       int               `json:"version"`
	Timestamp     time.Time         `json:"timestamp"`
	Source        string            `json:"source"`
	CorrelationID string            `json:"correlation_id,omitempty"`
	Data          json.RawMessage   `json:"data"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// NewEvent creates a new event with a generated ID and current timestamp.
func NewEvent(eventType, aggregateID, aggregateType, source string, data any) (*Event, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &Event{
		EventID:       uuid.New().String(),
		EventType:     eventType,
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		Version:       1,
		Timestamp:     time.Now().UTC(),
		Source:        source,
		Data:          dataBytes,
		Metadata:      make(map[string]string),
	}, nil
}

// WithCorrelationID sets the correlation ID on the event.
func (e *Event) WithCorrelationID(id string) *Event {
	e.CorrelationID = id
	return e
}

// WithMetadata adds a key-value pair to the event metadata.
func (e *Event) WithMetadata(key, value string) *Event {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}

// Marshal serializes the event to JSON bytes.
func (e *Event) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalEvent deserializes an event from JSON bytes.
func UnmarshalEvent(data []byte) (*Event, error) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// UnmarshalData deserializes the event data payload into the given target.
func (e *Event) UnmarshalData(target any) error {
	return json.Unmarshal(e.Data, target)
}
