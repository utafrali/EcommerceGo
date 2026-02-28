package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
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
}

// Consumer wraps the kafka-go reader for consuming events.
type Consumer struct {
	reader    *kafka.Reader
	logger    *slog.Logger
	handler   Handler
	closeOnce sync.Once
}

// NewConsumer creates a new Kafka consumer for a specific topic and group.
func NewConsumer(cfg ConsumerConfig, handler Handler, logger *slog.Logger) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		GroupID:  cfg.GroupID,
		Topic:    cfg.Topic,
		MinBytes: cfg.MinBytes,
		MaxBytes: cfg.MaxBytes,
	})

	return &Consumer{
		reader:  r,
		logger:  logger,
		handler: handler,
	}
}

// Start begins consuming messages. It blocks until the context is canceled.
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("consumer started",
		slog.String("topic", c.reader.Config().Topic),
		slog.String("group", c.reader.Config().GroupID),
	)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("consumer stopping", slog.String("topic", c.reader.Config().Topic))
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

			event, err := UnmarshalEvent(msg.Value)
			if err != nil {
				c.logger.Error("failed to unmarshal event",
					slog.String("error", err.Error()),
					slog.String("topic", msg.Topic),
				)
				if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
					c.logger.Error("failed to commit bad message", slog.String("error", commitErr.Error()))
				}
				continue
			}

			// Retry logic with exponential backoff.
			var lastErr error
			for attempt := 1; attempt <= maxHandlerRetries; attempt++ {
				if err := c.handler(ctx, event); err != nil {
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

					// If not the last attempt, wait with exponential backoff.
					if attempt < maxHandlerRetries {
						backoff := time.Duration(attempt) * 100 * time.Millisecond
						select {
						case <-ctx.Done():
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

			// If all retries failed, commit the message as poison pill.
			if lastErr != nil {
				c.logger.Error("handler failed after all retries, skipping poison message",
					slog.String("event_type", event.EventType),
					slog.String("aggregate_id", event.AggregateID),
					slog.String("error", lastErr.Error()),
					slog.String("topic", msg.Topic),
					slog.Int("partition", msg.Partition),
					slog.Int64("offset", msg.Offset),
					slog.Int("retries", maxHandlerRetries),
				)
				if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
					c.logger.Error("failed to commit poison message", slog.String("error", commitErr.Error()))
				}
				continue
			}

			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				c.logger.Error("failed to commit message", slog.String("error", err.Error()))
			}
		}
	}
}

// Close closes the consumer. It is safe to call multiple times.
func (c *Consumer) Close() error {
	var err error
	c.closeOnce.Do(func() {
		err = c.reader.Close()
	})
	return err
}

// TopicPrefix is the standard prefix for all EcommerceGo Kafka topics.
const TopicPrefix = "ecommerce"

// Topic constructs a fully-qualified topic name.
func Topic(domain, action string) string {
	return fmt.Sprintf("%s.%s.%s", TopicPrefix, domain, action)
}
