package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// maxHandlerRetries is the maximum number of times a message handler will be
// attempted before the message is committed and skipped (poison pill protection).
const maxHandlerRetries = 3

// Handler is a function that processes a Kafka event.
type Handler func(ctx context.Context, event *Event) error

// ConsumerConfig holds Kafka consumer configuration.
type ConsumerConfig struct {
	Brokers  []string
	GroupID  string
	Topic    string
	MinBytes int
	MaxBytes int
	// EnableDLQ enables dead-letter queue publishing for poison pill messages.
	// When true, failed messages are sent to ecommerce.dlq.<topic> instead of
	// being silently dropped.
	EnableDLQ bool
}

// Consumer wraps the kafka-go reader for consuming events.
type Consumer struct {
	reader    *kafka.Reader
	logger    *slog.Logger
	handler   Handler
	dlq       *DLQProducer
	closeOnce sync.Once
}

// NewConsumer creates a new Kafka consumer for a specific topic and group.
// If cfg.EnableDLQ is true, a DLQ producer is automatically initialized.
func NewConsumer(cfg ConsumerConfig, handler Handler, logger *slog.Logger) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		GroupID:  cfg.GroupID,
		Topic:    cfg.Topic,
		MinBytes: cfg.MinBytes,
		MaxBytes: cfg.MaxBytes,
	})

	var dlq *DLQProducer
	if cfg.EnableDLQ && len(cfg.Brokers) > 0 {
		dlq = NewDLQProducer(cfg.Brokers, logger)
	}

	return &Consumer{
		reader:  r,
		logger:  logger,
		handler: handler,
		dlq:     dlq,
	}
}

// Start begins consuming messages. It blocks until the context is canceled.
func (c *Consumer) Start(ctx context.Context) error {
	topic := c.reader.Config().Topic
	group := c.reader.Config().GroupID

	c.logger.Info("consumer started",
		slog.String("topic", topic),
		slog.String("group", group),
	)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("consumer stopping", slog.String("topic", topic))
			return c.Close()
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				c.logger.Error("failed to fetch message", slog.String("error", err.Error()))
				continue
			}

			ConsumerMessagesReceived.WithLabelValues(topic, group).Inc()

			// Extract trace context from Kafka message headers.
			msgCtx := otel.GetTextMapPropagator().Extract(ctx, &KafkaHeaderCarrier{&msg.Headers})

			event, err := UnmarshalEvent(msg.Value)
			if err != nil {
				c.logger.Error("failed to unmarshal event",
					slog.String("error", err.Error()),
					slog.String("topic", msg.Topic),
				)
				ConsumerMessagesFailed.WithLabelValues(topic, group).Inc()
				// Send unmarshalable messages to DLQ if available.
				if c.dlq != nil {
					if dlqErr := c.dlq.Publish(ctx, msg, err, group); dlqErr != nil {
						c.logger.Error("failed to send unmarshalable message to DLQ", slog.String("error", dlqErr.Error()))
					} else {
						ConsumerDLQPublished.WithLabelValues(topic, group).Inc()
					}
				}
				if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
					c.logger.Error("failed to commit bad message", slog.String("error", commitErr.Error()))
				}
				continue
			}

			// Create a consumer span linked to the producer trace.
			tracer := otel.Tracer("github.com/utafrali/EcommerceGo/pkg/kafka")
			msgCtx, span := tracer.Start(msgCtx, "kafka.consume "+msg.Topic,
				trace.WithSpanKind(trace.SpanKindConsumer),
				trace.WithAttributes(
					attribute.String("messaging.system", "kafka"),
					attribute.String("messaging.destination.name", msg.Topic),
					attribute.String("messaging.operation", "process"),
					attribute.String("messaging.kafka.consumer_group", group),
					attribute.Int("messaging.kafka.partition", msg.Partition),
					attribute.Int64("messaging.kafka.offset", msg.Offset),
					attribute.String("messaging.kafka.event_type", event.EventType),
				),
			)

			// Retry logic with exponential backoff.
			start := time.Now()
			var lastErr error
			for attempt := 1; attempt <= maxHandlerRetries; attempt++ {
				if err := c.handler(msgCtx, event); err != nil {
					lastErr = err
					c.logger.Warn("handler failed, will retry",
						slog.String("event_type", event.EventType),
						slog.String("aggregate_id", event.AggregateID),
						slog.String("error", err.Error()),
						slog.String("topic", msg.Topic),
						slog.Int("partition", msg.Partition),
						slog.Int64("offset", msg.Offset),
						slog.Int("attempt", attempt),
						slog.Int("max_retries", maxHandlerRetries),
					)

					// If not the last attempt, wait with exponential backoff + jitter.
					if attempt < maxHandlerRetries {
						backoff := 500 * time.Millisecond * time.Duration(1<<uint(attempt-1)) // 500ms, 1s, 2s
						// Apply ±25% jitter to prevent thundering herd.
						jitter := (rand.Float64() - 0.5) * 0.5 // #nosec G404 -- non-cryptographic jitter for retry backoff
						backoff = time.Duration(float64(backoff) * (1.0 + jitter))
						select {
						case <-ctx.Done():
							span.End()
							return nil
						case <-time.After(backoff):
							// Continue to next retry attempt
						}
					}
				} else {
					// Success - break out of retry loop
					lastErr = nil
					break
				}
			}

			elapsed := time.Since(start).Seconds()
			ConsumerProcessingDuration.WithLabelValues(topic, group).Observe(elapsed)

			// If all retries failed, send to DLQ (if enabled) then commit.
			if lastErr != nil {
				span.RecordError(lastErr)
				span.SetStatus(codes.Error, "all retries exhausted")
				span.End()
				ConsumerMessagesFailed.WithLabelValues(topic, group).Inc()
				c.logger.Error("handler failed after all retries, sending to DLQ",
					slog.String("event_type", event.EventType),
					slog.String("aggregate_id", event.AggregateID),
					slog.String("error", lastErr.Error()),
					slog.String("topic", msg.Topic),
					slog.Int("partition", msg.Partition),
					slog.Int64("offset", msg.Offset),
					slog.Int("retries", maxHandlerRetries),
				)
				if c.dlq != nil {
					if dlqErr := c.dlq.Publish(ctx, msg, lastErr, group); dlqErr != nil {
						c.logger.Error("failed to send poison message to DLQ", slog.String("error", dlqErr.Error()))
					} else {
						ConsumerDLQPublished.WithLabelValues(topic, group).Inc()
					}
				}
				if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
					c.logger.Error("failed to commit poison message", slog.String("error", commitErr.Error()))
				}
				continue
			}

			span.End()
			ConsumerMessagesProcessed.WithLabelValues(topic, group).Inc()

			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				c.logger.Error("failed to commit message", slog.String("error", err.Error()))
			}
		}
	}
}

// Close closes the consumer and its DLQ producer. It is safe to call multiple times.
func (c *Consumer) Close() error {
	var err error
	c.closeOnce.Do(func() {
		err = c.reader.Close()
		if c.dlq != nil {
			if dlqErr := c.dlq.Close(); dlqErr != nil {
				c.logger.Error("failed to close DLQ producer", slog.String("error", dlqErr.Error()))
				if err == nil {
					err = dlqErr
				}
			}
		}
	})
	return err
}

// TopicPrefix is the standard prefix for all EcommerceGo Kafka topics.
const TopicPrefix = "ecommerce"

// Topic constructs a fully-qualified topic name.
func Topic(domain, action string) string {
	return fmt.Sprintf("%s.%s.%s", TopicPrefix, domain, action)
}
