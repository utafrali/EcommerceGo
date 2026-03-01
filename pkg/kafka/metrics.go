package kafka

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ConsumerMessagesProcessed counts the total number of successfully processed messages.
	ConsumerMessagesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_consumer_messages_processed_total",
			Help: "Total number of successfully processed Kafka messages",
		},
		[]string{"topic", "consumer_group"},
	)

	// ConsumerMessagesFailed counts the total number of messages that exhausted retries.
	ConsumerMessagesFailed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_consumer_messages_failed_total",
			Help: "Total number of Kafka messages that failed all retries (sent to DLQ or dropped)",
		},
		[]string{"topic", "consumer_group"},
	)

	// ConsumerProcessingDuration observes the duration of message handler execution.
	ConsumerProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kafka_consumer_processing_duration_seconds",
			Help:    "Duration of Kafka message processing in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"topic", "consumer_group"},
	)

	// ConsumerMessagesReceived counts total messages fetched from Kafka (before processing).
	ConsumerMessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_consumer_messages_received_total",
			Help: "Total number of Kafka messages received (fetched from broker)",
		},
		[]string{"topic", "consumer_group"},
	)

	// ConsumerMessagesDuplicate counts messages skipped by idempotency middleware.
	ConsumerMessagesDuplicate = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_consumer_messages_duplicate_total",
			Help: "Total number of duplicate Kafka messages skipped by idempotency guard",
		},
		[]string{"topic", "consumer_group"},
	)

	// ProducerMessagesPublished counts the total number of messages published.
	ProducerMessagesPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_producer_messages_published_total",
			Help: "Total number of Kafka messages published",
		},
		[]string{"topic"},
	)

	// ProducerPublishErrors counts the total number of publish failures.
	ProducerPublishErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_producer_publish_errors_total",
			Help: "Total number of Kafka publish errors",
		},
		[]string{"topic"},
	)

	// ProducerPublishDuration observes the duration of publish operations.
	ProducerPublishDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kafka_producer_publish_duration_seconds",
			Help:    "Duration of Kafka publish operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"topic"},
	)

	// ConsumerDLQPublished counts messages sent to DLQ.
	ConsumerDLQPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_consumer_dlq_published_total",
			Help: "Total number of messages published to dead-letter queue",
		},
		[]string{"topic", "consumer_group"},
	)
)
