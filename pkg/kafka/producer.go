package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

// ProducerConfig holds Kafka producer configuration.
type ProducerConfig struct {
	Brokers      []string
	BatchSize    int
	BatchTimeout time.Duration
	Async        bool
}

// DefaultProducerConfig returns sensible defaults for the Kafka producer.
func DefaultProducerConfig(brokers []string) ProducerConfig {
	return ProducerConfig{
		Brokers:      brokers,
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		Async:        false,
	}
}

// Producer wraps the kafka-go writer for publishing events.
type Producer struct {
	writer *kafka.Writer
	logger *slog.Logger
}

// NewProducer creates a new Kafka producer.
func NewProducer(cfg ProducerConfig, logger *slog.Logger) *Producer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		Async:        cfg.Async,
		RequiredAcks: kafka.RequireAll,
	}

	return &Producer{
		writer: w,
		logger: logger,
	}
}

// Publish sends an event to the specified Kafka topic.
func (p *Producer) Publish(ctx context.Context, topic string, event *Event) error {
	data, err := event.Marshal()
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(event.AggregateID),
		Value: data,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(event.EventType)},
			{Key: "source", Value: []byte(event.Source)},
		},
	}

	if event.CorrelationID != "" {
		msg.Headers = append(msg.Headers, kafka.Header{
			Key: "correlation_id", Value: []byte(event.CorrelationID),
		})
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		p.logger.ErrorContext(ctx, "failed to publish event",
			slog.String("topic", topic),
			slog.String("event_type", event.EventType),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("publish event to %s: %w", topic, err)
	}

	p.logger.DebugContext(ctx, "event published",
		slog.String("topic", topic),
		slog.String("event_type", event.EventType),
		slog.String("aggregate_id", event.AggregateID),
	)

	return nil
}

// Close closes the producer and flushes pending messages.
func (p *Producer) Close() error {
	return p.writer.Close()
}
