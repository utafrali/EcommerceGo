package kafka

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

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
	reader  *kafka.Reader
	logger  *slog.Logger
	handler Handler
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
			return c.reader.Close()
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

			if err := c.handler(ctx, event); err != nil {
				c.logger.Error("failed to handle event",
					slog.String("event_type", event.EventType),
					slog.String("aggregate_id", event.AggregateID),
					slog.String("error", err.Error()),
				)
				// TODO: implement DLQ publishing for failed events
				continue
			}

			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				c.logger.Error("failed to commit message", slog.String("error", err.Error()))
			}
		}
	}
}

// Close closes the consumer.
func (c *Consumer) Close() error {
	return c.reader.Close()
}

// TopicPrefix is the standard prefix for all EcommerceGo Kafka topics.
const TopicPrefix = "ecommerce"

// Topic constructs a fully-qualified topic name.
func Topic(domain, action string) string {
	return fmt.Sprintf("%s.%s.%s", TopicPrefix, domain, action)
}
