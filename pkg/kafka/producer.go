package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
	writer  *kafka.Writer
	brokers []string
	logger  *slog.Logger
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
		writer:  w,
		brokers: cfg.Brokers,
		logger:  logger,
	}
}

// Publish sends an event to the specified Kafka topic. It injects the current
// trace context into the Kafka message headers so consumers can continue the
// trace.
func (p *Producer) Publish(ctx context.Context, topic string, event *Event) error {
	tracer := otel.Tracer("github.com/utafrali/EcommerceGo/pkg/kafka")
	ctx, span := tracer.Start(ctx, "kafka.produce "+topic,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination.name", topic),
			attribute.String("messaging.operation", "publish"),
			attribute.String("messaging.kafka.event_type", event.EventType),
		),
	)
	defer span.End()

	data, err := event.Marshal()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "marshal event failed")
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

	// Inject trace context into Kafka message headers.
	otel.GetTextMapPropagator().Inject(ctx, &KafkaHeaderCarrier{&msg.Headers})

	publishStart := time.Now()
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		ProducerPublishDuration.WithLabelValues(topic).Observe(time.Since(publishStart).Seconds())
		ProducerPublishErrors.WithLabelValues(topic).Inc()
		span.RecordError(err)
		span.SetStatus(codes.Error, "publish failed")
		p.logger.ErrorContext(ctx, "failed to publish event",
			slog.String("topic", topic),
			slog.String("event_type", event.EventType),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("publish event to %s: %w", topic, err)
	}
	ProducerPublishDuration.WithLabelValues(topic).Observe(time.Since(publishStart).Seconds())
	ProducerMessagesPublished.WithLabelValues(topic).Inc()

	p.logger.DebugContext(ctx, "event published",
		slog.String("topic", topic),
		slog.String("event_type", event.EventType),
		slog.String("aggregate_id", event.AggregateID),
	)

	return nil
}

// KafkaHeaderCarrier adapts Kafka message headers for OpenTelemetry
// propagation. It implements propagation.TextMapCarrier.
type KafkaHeaderCarrier struct {
	headers *[]kafka.Header
}

// Get returns the value for a key from Kafka headers.
func (c *KafkaHeaderCarrier) Get(key string) string {
	for _, h := range *c.headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

// Set adds or replaces a key-value pair in Kafka headers.
func (c *KafkaHeaderCarrier) Set(key, value string) {
	for i, h := range *c.headers {
		if h.Key == key {
			(*c.headers)[i].Value = []byte(value)
			return
		}
	}
	*c.headers = append(*c.headers, kafka.Header{Key: key, Value: []byte(value)})
}

// Keys returns all header keys.
func (c *KafkaHeaderCarrier) Keys() []string {
	keys := make([]string, len(*c.headers))
	for i, h := range *c.headers {
		keys[i] = h.Key
	}
	return keys
}

// Ping checks Kafka broker connectivity by dialing the first reachable broker.
func (p *Producer) Ping(ctx context.Context) error {
	return PingBrokers(ctx, p.brokers)
}

// PingBrokers dials the given Kafka brokers and returns nil if at least one
// broker is reachable. This is useful as a standalone health check when only
// consumers (no producer) are used.
func PingBrokers(ctx context.Context, brokers []string) error {
	if len(brokers) == 0 {
		return fmt.Errorf("kafka: no brokers configured")
	}

	var lastErr error
	for _, addr := range brokers {
		conn, err := kafka.DialContext(ctx, "tcp", addr)
		if err != nil {
			lastErr = err
			continue
		}
		// Successfully connected; request broker list as a lightweight health probe.
		_, err = conn.Brokers()
		_ = conn.Close()
		if err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("kafka ping: all brokers unreachable: %w", lastErr)
}

// Close closes the producer and flushes pending messages.
func (p *Producer) Close() error {
	return p.writer.Close()
}
