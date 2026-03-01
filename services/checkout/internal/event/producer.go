package event

import (
	"context"
	"fmt"
	"log/slog"

	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/checkout/internal/domain"
)

// Kafka topic constants for checkout domain events.
const (
	TopicCheckoutInitiated = "ecommerce.checkout.initiated"
	TopicCheckoutCompleted = "ecommerce.checkout.completed"
	TopicCheckoutFailed    = "ecommerce.checkout.failed"
)

// Aggregate type constant.
const AggregateTypeCheckout = "checkout"

// Source identifier for events originating from the checkout service.
const SourceCheckoutService = "checkout-service"

// CheckoutInitiatedData is the payload for a checkout.initiated event.
type CheckoutInitiatedData struct {
	ID             string                `json:"id"`
	UserID         string                `json:"user_id"`
	Items          []domain.CheckoutItem `json:"items"`
	SubtotalAmount int64                 `json:"subtotal_amount"`
	TotalAmount    int64                 `json:"total_amount"`
	Currency       string                `json:"currency"`
}

// CheckoutCompletedData is the payload for a checkout.completed event.
type CheckoutCompletedData struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	OrderID     string `json:"order_id"`
	PaymentID   string `json:"payment_id"`
	TotalAmount int64  `json:"total_amount"`
	Currency    string `json:"currency"`
}

// CheckoutFailedData is the payload for a checkout.failed event.
type CheckoutFailedData struct {
	ID            string `json:"id"`
	UserID        string `json:"user_id"`
	FailureReason string `json:"failure_reason"`
}

// Producer publishes checkout domain events to Kafka.
type Producer struct {
	kafka  *pkgkafka.Producer
	logger *slog.Logger
}

// NewProducer creates a new event producer for the checkout service.
func NewProducer(kafka *pkgkafka.Producer, logger *slog.Logger) *Producer {
	return &Producer{
		kafka:  kafka,
		logger: logger,
	}
}

// PublishCheckoutInitiated publishes a checkout.initiated event.
func (p *Producer) PublishCheckoutInitiated(ctx context.Context, session *domain.CheckoutSession) error {
	data := CheckoutInitiatedData{
		ID:             session.ID,
		UserID:         session.UserID,
		Items:          session.Items,
		SubtotalAmount: session.SubtotalAmount,
		TotalAmount:    session.TotalAmount,
		Currency:       session.Currency,
	}

	event, err := pkgkafka.NewEvent(TopicCheckoutInitiated, session.ID, AggregateTypeCheckout, SourceCheckoutService, data)
	if err != nil {
		return fmt.Errorf("create checkout.initiated event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicCheckoutInitiated, event); err != nil {
		return fmt.Errorf("publish checkout.initiated event: %w", err)
	}

	p.logger.DebugContext(ctx, "published checkout.initiated event",
		slog.String("checkout_id", session.ID),
		slog.String("user_id", session.UserID),
	)

	return nil
}

// PublishCheckoutCompleted publishes a checkout.completed event.
func (p *Producer) PublishCheckoutCompleted(ctx context.Context, session *domain.CheckoutSession) error {
	data := CheckoutCompletedData{
		ID:          session.ID,
		UserID:      session.UserID,
		OrderID:     session.OrderID,
		PaymentID:   session.PaymentID,
		TotalAmount: session.TotalAmount,
		Currency:    session.Currency,
	}

	event, err := pkgkafka.NewEvent(TopicCheckoutCompleted, session.ID, AggregateTypeCheckout, SourceCheckoutService, data)
	if err != nil {
		return fmt.Errorf("create checkout.completed event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicCheckoutCompleted, event); err != nil {
		return fmt.Errorf("publish checkout.completed event: %w", err)
	}

	p.logger.DebugContext(ctx, "published checkout.completed event",
		slog.String("checkout_id", session.ID),
		slog.String("order_id", session.OrderID),
	)

	return nil
}

// PublishCheckoutFailed publishes a checkout.failed event.
func (p *Producer) PublishCheckoutFailed(ctx context.Context, session *domain.CheckoutSession) error {
	data := CheckoutFailedData{
		ID:            session.ID,
		UserID:        session.UserID,
		FailureReason: session.FailureReason,
	}

	event, err := pkgkafka.NewEvent(TopicCheckoutFailed, session.ID, AggregateTypeCheckout, SourceCheckoutService, data)
	if err != nil {
		return fmt.Errorf("create checkout.failed event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicCheckoutFailed, event); err != nil {
		return fmt.Errorf("publish checkout.failed event: %w", err)
	}

	p.logger.DebugContext(ctx, "published checkout.failed event",
		slog.String("checkout_id", session.ID),
		slog.String("failure_reason", session.FailureReason),
	)

	return nil
}
