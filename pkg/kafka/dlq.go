package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

// DLQTopicPrefix is the default prefix for dead-letter queue topics.
const DLQTopicPrefix = "ecommerce.dlq"

// DLQProducer publishes failed messages to a dead-letter queue topic.
type DLQProducer struct {
	writer *kafka.Writer
	logger *slog.Logger
}

// NewDLQProducer creates a DLQ producer that writes to topics prefixed with
// the given prefix (defaults to DLQTopicPrefix if empty).
func NewDLQProducer(brokers []string, logger *slog.Logger) *DLQProducer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    1,
		BatchTimeout: 100 * time.Millisecond,
		Async:        false,
		RequiredAcks: kafka.RequireAll,
	}

	return &DLQProducer{
		writer: w,
		logger: logger,
	}
}

// DLQTopic constructs the DLQ topic name for a given source topic.
func DLQTopic(originalTopic string) string {
	return fmt.Sprintf("%s.%s", DLQTopicPrefix, originalTopic)
}

// Publish sends a failed message to the corresponding DLQ topic. It includes
// the original topic, partition, offset, error message, and consumer group as
// headers for debugging.
func (d *DLQProducer) Publish(ctx context.Context, originalMsg kafka.Message, lastErr error, consumerGroup string) error {
	dlqTopic := DLQTopic(originalMsg.Topic)

	headers := make([]kafka.Header, 0, len(originalMsg.Headers)+4)
	headers = append(headers, originalMsg.Headers...)
	headers = append(headers,
		kafka.Header{Key: "dlq.original_topic", Value: []byte(originalMsg.Topic)},
		kafka.Header{Key: "dlq.original_partition", Value: []byte(fmt.Sprintf("%d", originalMsg.Partition))},
		kafka.Header{Key: "dlq.original_offset", Value: []byte(fmt.Sprintf("%d", originalMsg.Offset))},
		kafka.Header{Key: "dlq.consumer_group", Value: []byte(consumerGroup)},
	)
	if lastErr != nil {
		headers = append(headers, kafka.Header{Key: "dlq.error", Value: []byte(lastErr.Error())})
	}

	dlqMsg := kafka.Message{
		Topic:   dlqTopic,
		Key:     originalMsg.Key,
		Value:   originalMsg.Value,
		Headers: headers,
	}

	if err := d.writer.WriteMessages(ctx, dlqMsg); err != nil {
		d.logger.Error("failed to publish message to DLQ",
			slog.String("dlq_topic", dlqTopic),
			slog.String("original_topic", originalMsg.Topic),
			slog.Int("partition", originalMsg.Partition),
			slog.Int64("offset", originalMsg.Offset),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("publish to DLQ %s: %w", dlqTopic, err)
	}

	d.logger.Warn("message sent to DLQ",
		slog.String("dlq_topic", dlqTopic),
		slog.String("original_topic", originalMsg.Topic),
		slog.Int("partition", originalMsg.Partition),
		slog.Int64("offset", originalMsg.Offset),
		slog.String("consumer_group", consumerGroup),
	)

	return nil
}

// Close closes the DLQ producer.
func (d *DLQProducer) Close() error {
	return d.writer.Close()
}
