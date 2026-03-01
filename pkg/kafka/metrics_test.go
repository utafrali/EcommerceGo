package kafka

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gatherMetricNames collects all metric names from the default registry.
func gatherMetricNames(t *testing.T) map[string]bool {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	names := make(map[string]bool, len(families))
	for _, fam := range families {
		names[fam.GetName()] = true
	}
	return names
}

func TestConsumerMetrics_Registered(t *testing.T) {
	expectedMetrics := []string{
		"kafka_consumer_messages_processed_total",
		"kafka_consumer_messages_failed_total",
		"kafka_consumer_processing_duration_seconds",
		"kafka_consumer_messages_received_total",
		"kafka_consumer_dlq_published_total",
	}

	// promauto registers with default registry, but counters with no observations
	// may not appear in Gather() until they receive at least one observation.
	// Touch each metric so it appears in the gathered output.
	ConsumerMessagesProcessed.WithLabelValues("test-topic", "test-group")
	ConsumerMessagesFailed.WithLabelValues("test-topic", "test-group")
	ConsumerProcessingDuration.WithLabelValues("test-topic", "test-group")
	ConsumerMessagesReceived.WithLabelValues("test-topic", "test-group")
	ConsumerDLQPublished.WithLabelValues("test-topic", "test-group")

	names := gatherMetricNames(t)

	for _, name := range expectedMetrics {
		assert.True(t, names[name], "expected metric %q to be registered", name)
	}
}

func TestProducerMetrics_Registered(t *testing.T) {
	// Touch the producer metrics so they appear in Gather().
	ProducerMessagesPublished.WithLabelValues("test-topic")
	ProducerPublishErrors.WithLabelValues("test-topic")
	ProducerPublishDuration.WithLabelValues("test-topic")

	names := gatherMetricNames(t)

	expectedMetrics := []string{
		"kafka_producer_messages_published_total",
		"kafka_producer_publish_errors_total",
		"kafka_producer_publish_duration_seconds",
	}

	for _, name := range expectedMetrics {
		assert.True(t, names[name], "expected metric %q to be registered", name)
	}
}

func TestConsumerMetrics_IncrementAndCollect(t *testing.T) {
	// Use a unique label combination to avoid interference from other tests.
	topic := "metrics-test-consumer-topic"
	group := "metrics-test-consumer-group"

	// Record the current value (may be non-zero if other tests ran first).
	initialProcessed := getCounterValue(t, "kafka_consumer_messages_processed_total", topic, group)
	initialFailed := getCounterValue(t, "kafka_consumer_messages_failed_total", topic, group)
	initialReceived := getCounterValue(t, "kafka_consumer_messages_received_total", topic, group)

	// Increment counters.
	ConsumerMessagesProcessed.WithLabelValues(topic, group).Inc()
	ConsumerMessagesProcessed.WithLabelValues(topic, group).Inc()
	ConsumerMessagesProcessed.WithLabelValues(topic, group).Inc()
	ConsumerMessagesFailed.WithLabelValues(topic, group).Inc()
	ConsumerMessagesReceived.WithLabelValues(topic, group).Add(5)
	ConsumerProcessingDuration.WithLabelValues(topic, group).Observe(0.123)

	// Verify the incremented values.
	assert.InDelta(t, initialProcessed+3, getCounterValue(t, "kafka_consumer_messages_processed_total", topic, group), 0.001)
	assert.InDelta(t, initialFailed+1, getCounterValue(t, "kafka_consumer_messages_failed_total", topic, group), 0.001)
	assert.InDelta(t, initialReceived+5, getCounterValue(t, "kafka_consumer_messages_received_total", topic, group), 0.001)

	// Verify histogram has at least one observation.
	histogramCount := getHistogramCount(t, "kafka_consumer_processing_duration_seconds", topic, group)
	assert.GreaterOrEqual(t, histogramCount, uint64(1))
}

func TestProducerMetrics_IncrementAndCollect(t *testing.T) {
	topic := "metrics-test-producer-topic"

	initialPublished := getCounterValue(t, "kafka_producer_messages_published_total", topic, "")
	initialErrors := getCounterValue(t, "kafka_producer_publish_errors_total", topic, "")

	ProducerMessagesPublished.WithLabelValues(topic).Inc()
	ProducerMessagesPublished.WithLabelValues(topic).Inc()
	ProducerPublishErrors.WithLabelValues(topic).Inc()
	ProducerPublishDuration.WithLabelValues(topic).Observe(0.05)

	assert.InDelta(t, initialPublished+2, getCounterValue(t, "kafka_producer_messages_published_total", topic, ""), 0.001)
	assert.InDelta(t, initialErrors+1, getCounterValue(t, "kafka_producer_publish_errors_total", topic, ""), 0.001)

	histogramCount := getHistogramCount(t, "kafka_producer_publish_duration_seconds", topic, "")
	assert.GreaterOrEqual(t, histogramCount, uint64(1))
}

func TestConsumerMessagesDuplicate_Registered(t *testing.T) {
	ConsumerMessagesDuplicate.WithLabelValues("dup-topic", "dup-group").Inc()

	names := gatherMetricNames(t)
	assert.True(t, names["kafka_consumer_messages_duplicate_total"],
		"expected kafka_consumer_messages_duplicate_total to be registered")
}

// getCounterValue retrieves the current value of a counter metric with the given labels.
// For producer metrics (single "topic" label), pass group as "".
func getCounterValue(t *testing.T, metricName, topic, group string) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	for _, fam := range families {
		if fam.GetName() != metricName {
			continue
		}
		for _, m := range fam.GetMetric() {
			labels := make(map[string]string)
			for _, lp := range m.GetLabel() {
				labels[lp.GetName()] = lp.GetValue()
			}
			if labels["topic"] == topic && (group == "" || labels["consumer_group"] == group) {
				if m.GetCounter() != nil {
					return m.GetCounter().GetValue()
				}
			}
		}
	}
	return 0
}

// getHistogramCount retrieves the sample count for a histogram metric.
func getHistogramCount(t *testing.T, metricName, topic, group string) uint64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	for _, fam := range families {
		if fam.GetName() != metricName {
			continue
		}
		for _, m := range fam.GetMetric() {
			labels := make(map[string]string)
			for _, lp := range m.GetLabel() {
				labels[lp.GetName()] = lp.GetValue()
			}
			if labels["topic"] == topic && (group == "" || labels["consumer_group"] == group) {
				if m.GetHistogram() != nil {
					return m.GetHistogram().GetSampleCount()
				}
			}
		}
	}
	return 0
}

func TestMetrics_DescriptionsNonEmpty(t *testing.T) {
	// Verify that each metric has a non-empty help string by checking the
	// gathered MetricFamily descriptions.
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	kafkaMetrics := []string{
		"kafka_consumer_messages_processed_total",
		"kafka_consumer_messages_failed_total",
		"kafka_consumer_processing_duration_seconds",
		"kafka_consumer_messages_received_total",
		"kafka_consumer_messages_duplicate_total",
		"kafka_consumer_dlq_published_total",
		"kafka_producer_messages_published_total",
		"kafka_producer_publish_errors_total",
		"kafka_producer_publish_duration_seconds",
	}

	helpByName := make(map[string]string)
	for _, fam := range families {
		helpByName[fam.GetName()] = fam.GetHelp()
	}

	for _, name := range kafkaMetrics {
		help, exists := helpByName[name]
		assert.True(t, exists, "metric %q not found in gathered families", name)
		assert.NotEmpty(t, help, "metric %q should have a non-empty help string", name)
		lowerHelp := strings.ToLower(help)
		mentionsKafka := strings.Contains(lowerHelp, "kafka") ||
			strings.Contains(lowerHelp, "dead-letter") ||
			strings.Contains(lowerHelp, "dlq")
		assert.True(t, mentionsKafka,
			"metric %q help %q should mention kafka, dead-letter, or dlq", name, help)
	}
}
