package event

import (
	"context"
	"encoding/json"
	"fmt"
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

// NotificationSender is the interface the consumer uses to create notifications.
// It is satisfied by *service.NotificationService without importing the service package,
// which avoids the event <-> service import cycle.
type NotificationSender interface {
	SendNotification(ctx context.Context, input *SendInput) error
}

// SendInput mirrors service.SendNotificationInput so the event package does not
// import the service package. The app layer is responsible for the adapter.
type SendInput struct {
	UserID   string
	Type     string
	Channel  string
	Subject  string
	Body     string
	Priority string
	Metadata map[string]any
}

// ConsumerHandler routes incoming Kafka events to the appropriate handler.
type ConsumerHandler struct {
	logger *slog.Logger
	sender NotificationSender
}

// NewConsumerHandler creates a new event consumer handler.
func NewConsumerHandler(sender NotificationSender, logger *slog.Logger) *ConsumerHandler {
	return &ConsumerHandler{
		logger: logger,
		sender: sender,
	}
}

// orderCreatedPayload is the JSON structure in the Data field of an order.created event.
type orderCreatedPayload struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	TotalAmount int64  `json:"total_amount"`
	Currency    string `json:"currency"`
}

// paymentPayload is the JSON structure in the Data field of payment events.
type paymentPayload struct {
	ID            string `json:"id"`
	OrderID       string `json:"order_id"`
	UserID        string `json:"user_id"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	FailureReason string `json:"failure_reason,omitempty"`
}

// checkoutCompletedPayload is the JSON structure in the Data field of a checkout.completed event.
type checkoutCompletedPayload struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	OrderID     string `json:"order_id"`
	TotalAmount int64  `json:"total_amount"`
	Currency    string `json:"currency"`
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
func (h *ConsumerHandler) handleOrderCreated(ctx context.Context, event *pkgkafka.Event) error {
	h.logger.InfoContext(ctx, "received order.created event",
		slog.String("event_id", event.EventID),
		slog.String("aggregate_id", event.AggregateID),
		slog.String("source", event.Source),
	)

	var payload orderCreatedPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("unmarshal order.created payload: %w", err)
	}

	if payload.UserID == "" {
		h.logger.WarnContext(ctx, "order.created event missing user_id, skipping",
			slog.String("event_id", event.EventID),
		)
		return nil
	}

	input := &SendInput{
		UserID:  payload.UserID,
		Type:    "email",
		Channel: "email",
		Subject: "Order Confirmed",
		Body:    fmt.Sprintf("Your order %s has been created successfully. Total: %d %s.", payload.ID, payload.TotalAmount, payload.Currency),
		Metadata: map[string]any{
			"order_id": payload.ID,
			"event_id": event.EventID,
		},
	}

	if err := h.sender.SendNotification(ctx, input); err != nil {
		return fmt.Errorf("send order_created notification: %w", err)
	}

	return nil
}

// handlePaymentSucceeded processes payment.succeeded events.
func (h *ConsumerHandler) handlePaymentSucceeded(ctx context.Context, event *pkgkafka.Event) error {
	h.logger.InfoContext(ctx, "received payment.succeeded event",
		slog.String("event_id", event.EventID),
		slog.String("aggregate_id", event.AggregateID),
		slog.String("source", event.Source),
	)

	var payload paymentPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("unmarshal payment.succeeded payload: %w", err)
	}

	if payload.UserID == "" {
		h.logger.WarnContext(ctx, "payment.succeeded event missing user_id, skipping",
			slog.String("event_id", event.EventID),
		)
		return nil
	}

	input := &SendInput{
		UserID:  payload.UserID,
		Type:    "email",
		Channel: "email",
		Subject: "Payment Successful",
		Body:    fmt.Sprintf("Your payment of %d %s for order %s was successful.", payload.Amount, payload.Currency, payload.OrderID),
		Metadata: map[string]any{
			"payment_id": payload.ID,
			"order_id":   payload.OrderID,
			"event_id":   event.EventID,
		},
	}

	if err := h.sender.SendNotification(ctx, input); err != nil {
		return fmt.Errorf("send payment_succeeded notification: %w", err)
	}

	return nil
}

// handlePaymentFailed processes payment.failed events.
func (h *ConsumerHandler) handlePaymentFailed(ctx context.Context, event *pkgkafka.Event) error {
	h.logger.InfoContext(ctx, "received payment.failed event",
		slog.String("event_id", event.EventID),
		slog.String("aggregate_id", event.AggregateID),
		slog.String("source", event.Source),
	)

	var payload paymentPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("unmarshal payment.failed payload: %w", err)
	}

	if payload.UserID == "" {
		h.logger.WarnContext(ctx, "payment.failed event missing user_id, skipping",
			slog.String("event_id", event.EventID),
		)
		return nil
	}

	input := &SendInput{
		UserID:   payload.UserID,
		Type:     "email",
		Channel:  "email",
		Subject:  "Payment Failed",
		Body:     fmt.Sprintf("Your payment for order %s has failed. Reason: %s. Please try again.", payload.OrderID, payload.FailureReason),
		Priority: "high",
		Metadata: map[string]any{
			"payment_id":     payload.ID,
			"order_id":       payload.OrderID,
			"failure_reason": payload.FailureReason,
			"event_id":       event.EventID,
		},
	}

	if err := h.sender.SendNotification(ctx, input); err != nil {
		return fmt.Errorf("send payment_failed notification: %w", err)
	}

	return nil
}

// handleCheckoutCompleted processes checkout.completed events.
func (h *ConsumerHandler) handleCheckoutCompleted(ctx context.Context, event *pkgkafka.Event) error {
	h.logger.InfoContext(ctx, "received checkout.completed event",
		slog.String("event_id", event.EventID),
		slog.String("aggregate_id", event.AggregateID),
		slog.String("source", event.Source),
	)

	var payload checkoutCompletedPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("unmarshal checkout.completed payload: %w", err)
	}

	if payload.UserID == "" {
		h.logger.WarnContext(ctx, "checkout.completed event missing user_id, skipping",
			slog.String("event_id", event.EventID),
		)
		return nil
	}

	input := &SendInput{
		UserID:  payload.UserID,
		Type:    "email",
		Channel: "email",
		Subject: "Checkout Complete",
		Body:    fmt.Sprintf("Your checkout %s is complete. Order %s totaling %d %s is being processed.", payload.ID, payload.OrderID, payload.TotalAmount, payload.Currency),
		Metadata: map[string]any{
			"checkout_id": payload.ID,
			"order_id":    payload.OrderID,
			"event_id":    event.EventID,
		},
	}

	if err := h.sender.SendNotification(ctx, input); err != nil {
		return fmt.Errorf("send checkout_completed notification: %w", err)
	}

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
