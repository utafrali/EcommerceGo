package kafka

import (
	"testing"
)

func TestDLQTopicPrefix(t *testing.T) {
	if DLQTopicPrefix != "ecommerce.dlq" {
		t.Errorf("DLQTopicPrefix = %q, want %q", DLQTopicPrefix, "ecommerce.dlq")
	}
}

func TestDLQTopic(t *testing.T) {
	tests := []struct {
		name          string
		originalTopic string
		want          string
	}{
		{
			name:          "standard topic",
			originalTopic: "ecommerce.order.confirmed",
			want:          "ecommerce.dlq.ecommerce.order.confirmed",
		},
		{
			name:          "simple topic name",
			originalTopic: "orders",
			want:          "ecommerce.dlq.orders",
		},
		{
			name:          "deeply nested topic",
			originalTopic: "ecommerce.payment.stripe.webhook",
			want:          "ecommerce.dlq.ecommerce.payment.stripe.webhook",
		},
		{
			name:          "single word topic",
			originalTopic: "notifications",
			want:          "ecommerce.dlq.notifications",
		},
		{
			name:          "topic with hyphens",
			originalTopic: "user-events",
			want:          "ecommerce.dlq.user-events",
		},
		{
			name:          "topic with underscores",
			originalTopic: "inventory_updates",
			want:          "ecommerce.dlq.inventory_updates",
		},
		{
			name:          "empty topic",
			originalTopic: "",
			want:          "ecommerce.dlq.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DLQTopic(tt.originalTopic)
			if got != tt.want {
				t.Errorf("DLQTopic(%q) = %q, want %q", tt.originalTopic, got, tt.want)
			}
		})
	}
}

func TestDLQTopic_ContainsPrefix(t *testing.T) {
	topic := DLQTopic("some.topic")
	if len(topic) <= len(DLQTopicPrefix) {
		t.Fatalf("DLQTopic result %q should be longer than prefix %q", topic, DLQTopicPrefix)
	}
	prefix := topic[:len(DLQTopicPrefix)]
	if prefix != DLQTopicPrefix {
		t.Errorf("DLQTopic(%q) prefix = %q, want %q", "some.topic", prefix, DLQTopicPrefix)
	}
}
