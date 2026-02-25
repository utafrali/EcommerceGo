package event

import (
	"context"
	"log/slog"

	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
)

// Topics consumed from other services.
const (
	TopicOrderCreated      = "ecommerce.order.created"
	TopicPaymentSucceeded  = "ecommerce.payment.succeeded"
	TopicPaymentFailed     = "ecommerce.payment.failed"
	TopicCheckoutCompleted = "ecommerce.checkout.completed"
)

// Consumer group ID for the notification service.
const ConsumerGroupID = "notification-service"

// ConsumerHandler routes incoming Kafka events to the appropriate handler.
type ConsumerHandler struct {
	logger *slog.Logger
}

// NewConsumerHandler creates a new event consumer handler.
func NewConsumerHandler(logger *slog.Logger) *ConsumerHandler {
	return &ConsumerHandler{
		logger: logger,
	}
}

// Handle processes an incoming Kafka event based on its event type.
func (h *ConsumerHandler) Handle(ctx context.Context, event *pkgkafka.Event) error {
	switch event.EventType {
	case TopicOrderCreated:
		return h.handleOrderCreated(ctx, event)
	case TopicPaymentSucceeded:
		return h.handlePaymentSucceeded(ctx, event)
	case TopicPaymentFailed:
		return h.handlePaymentFailed(ctx, event)
	case TopicCheckoutCompleted:
		return h.handleCheckoutCompleted(ctx, event)
	default:
		h.logger.WarnContext(ctx, "unknown event type received",
			slog.String("event_type", event.EventType),
			slog.String("event_id", event.EventID),
		)
		return nil
	}
}

// handleOrderCreated processes order.created events.
// Stub: logs the event for future implementation.
func (h *ConsumerHandler) handleOrderCreated(ctx context.Context, event *pkgkafka.Event) error {
	h.logger.InfoContext(ctx, "received order.created event",
		slog.String("event_id", event.EventID),
		slog.String("aggregate_id", event.AggregateID),
		slog.String("source", event.Source),
	)
	// TODO: Create notification for the user about order creation.
	return nil
}

// handlePaymentSucceeded processes payment.succeeded events.
// Stub: logs the event for future implementation.
func (h *ConsumerHandler) handlePaymentSucceeded(ctx context.Context, event *pkgkafka.Event) error {
	h.logger.InfoContext(ctx, "received payment.succeeded event",
		slog.String("event_id", event.EventID),
		slog.String("aggregate_id", event.AggregateID),
		slog.String("source", event.Source),
	)
	// TODO: Create notification for the user about successful payment.
	return nil
}

// handlePaymentFailed processes payment.failed events.
// Stub: logs the event for future implementation.
func (h *ConsumerHandler) handlePaymentFailed(ctx context.Context, event *pkgkafka.Event) error {
	h.logger.InfoContext(ctx, "received payment.failed event",
		slog.String("event_id", event.EventID),
		slog.String("aggregate_id", event.AggregateID),
		slog.String("source", event.Source),
	)
	// TODO: Create notification for the user about failed payment.
	return nil
}

// handleCheckoutCompleted processes checkout.completed events.
// Stub: logs the event for future implementation.
func (h *ConsumerHandler) handleCheckoutCompleted(ctx context.Context, event *pkgkafka.Event) error {
	h.logger.InfoContext(ctx, "received checkout.completed event",
		slog.String("event_id", event.EventID),
		slog.String("aggregate_id", event.AggregateID),
		slog.String("source", event.Source),
	)
	// TODO: Create notification for the user about completed checkout.
	return nil
}

// NewConsumers creates Kafka consumers for all topics the notification service subscribes to.
func NewConsumers(brokers []string, handler *ConsumerHandler, logger *slog.Logger) []*pkgkafka.Consumer {
	topics := []string{
		TopicOrderCreated,
		TopicPaymentSucceeded,
		TopicPaymentFailed,
		TopicCheckoutCompleted,
	}

	consumers := make([]*pkgkafka.Consumer, 0, len(topics))

	for _, topic := range topics {
		cfg := pkgkafka.ConsumerConfig{
			Brokers:  brokers,
			GroupID:  ConsumerGroupID,
			Topic:    topic,
			MinBytes: 1,
			MaxBytes: 10e6,
		}

		consumer := pkgkafka.NewConsumer(cfg, handler.Handle, logger)
		consumers = append(consumers, consumer)
	}

	return consumers
}
