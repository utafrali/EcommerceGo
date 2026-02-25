package event

import (
	"context"
	"fmt"
	"log/slog"

	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/payment/internal/domain"
)

// Kafka topic constants for payment domain events.
const (
	TopicPaymentSucceeded = "ecommerce.payment.succeeded"
	TopicPaymentFailed    = "ecommerce.payment.failed"
	TopicPaymentRefunded  = "ecommerce.payment.refunded"
)

// Aggregate type constant.
const AggregateTypePayment = "payment"

// Source identifier for events originating from the payment service.
const SourcePaymentService = "payment-service"

// PaymentSucceededData is the payload for a payment.succeeded event.
type PaymentSucceededData struct {
	ID            string `json:"id"`
	CheckoutID    string `json:"checkout_id"`
	OrderID       string `json:"order_id"`
	UserID        string `json:"user_id"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	Method        string `json:"method"`
	ProviderName  string `json:"provider_name"`
	ProviderPayID string `json:"provider_payment_id"`
}

// PaymentFailedData is the payload for a payment.failed event.
type PaymentFailedData struct {
	ID            string `json:"id"`
	CheckoutID    string `json:"checkout_id"`
	OrderID       string `json:"order_id"`
	UserID        string `json:"user_id"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	FailureReason string `json:"failure_reason"`
}

// PaymentRefundedData is the payload for a payment.refunded event.
type PaymentRefundedData struct {
	PaymentID    string `json:"payment_id"`
	RefundID     string `json:"refund_id"`
	RefundAmount int64  `json:"refund_amount"`
	Currency     string `json:"currency"`
	Reason       string `json:"reason"`
	Status       string `json:"status"`
}

// Producer publishes payment domain events to Kafka.
type Producer struct {
	kafka  *pkgkafka.Producer
	logger *slog.Logger
}

// NewProducer creates a new event producer for the payment service.
func NewProducer(kafka *pkgkafka.Producer, logger *slog.Logger) *Producer {
	return &Producer{
		kafka:  kafka,
		logger: logger,
	}
}

// PublishPaymentSucceeded publishes a payment.succeeded event.
func (p *Producer) PublishPaymentSucceeded(ctx context.Context, payment *domain.Payment) error {
	data := PaymentSucceededData{
		ID:            payment.ID,
		CheckoutID:    payment.CheckoutID,
		OrderID:       payment.OrderID,
		UserID:        payment.UserID,
		Amount:        payment.Amount,
		Currency:      payment.Currency,
		Method:        payment.Method,
		ProviderName:  payment.ProviderName,
		ProviderPayID: payment.ProviderPayID,
	}

	event, err := pkgkafka.NewEvent(TopicPaymentSucceeded, payment.ID, AggregateTypePayment, SourcePaymentService, data)
	if err != nil {
		return fmt.Errorf("create payment.succeeded event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicPaymentSucceeded, event); err != nil {
		return fmt.Errorf("publish payment.succeeded event: %w", err)
	}

	p.logger.DebugContext(ctx, "published payment.succeeded event",
		slog.String("payment_id", payment.ID),
		slog.String("order_id", payment.OrderID),
	)

	return nil
}

// PublishPaymentFailed publishes a payment.failed event.
func (p *Producer) PublishPaymentFailed(ctx context.Context, payment *domain.Payment) error {
	data := PaymentFailedData{
		ID:            payment.ID,
		CheckoutID:    payment.CheckoutID,
		OrderID:       payment.OrderID,
		UserID:        payment.UserID,
		Amount:        payment.Amount,
		Currency:      payment.Currency,
		FailureReason: payment.FailureReason,
	}

	event, err := pkgkafka.NewEvent(TopicPaymentFailed, payment.ID, AggregateTypePayment, SourcePaymentService, data)
	if err != nil {
		return fmt.Errorf("create payment.failed event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicPaymentFailed, event); err != nil {
		return fmt.Errorf("publish payment.failed event: %w", err)
	}

	p.logger.DebugContext(ctx, "published payment.failed event",
		slog.String("payment_id", payment.ID),
		slog.String("failure_reason", payment.FailureReason),
	)

	return nil
}

// PublishPaymentRefunded publishes a payment.refunded event.
func (p *Producer) PublishPaymentRefunded(ctx context.Context, payment *domain.Payment, refund *domain.Refund) error {
	data := PaymentRefundedData{
		PaymentID:    payment.ID,
		RefundID:     refund.ID,
		RefundAmount: refund.Amount,
		Currency:     refund.Currency,
		Reason:       refund.Reason,
		Status:       payment.Status,
	}

	event, err := pkgkafka.NewEvent(TopicPaymentRefunded, payment.ID, AggregateTypePayment, SourcePaymentService, data)
	if err != nil {
		return fmt.Errorf("create payment.refunded event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicPaymentRefunded, event); err != nil {
		return fmt.Errorf("publish payment.refunded event: %w", err)
	}

	p.logger.DebugContext(ctx, "published payment.refunded event",
		slog.String("payment_id", payment.ID),
		slog.String("refund_id", refund.ID),
		slog.Int64("refund_amount", refund.Amount),
	)

	return nil
}
