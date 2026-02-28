package repository

import (
	"context"

	"github.com/utafrali/EcommerceGo/services/payment/internal/domain"
)

// PaymentRepository defines the interface for payment persistence operations.
type PaymentRepository interface {
	// Create inserts a new payment into the store.
	Create(ctx context.Context, payment *domain.Payment) error

	// GetByID retrieves a payment by its unique identifier.
	GetByID(ctx context.Context, id string) (*domain.Payment, error)

	// GetByCheckoutID retrieves a payment by the associated checkout ID.
	GetByCheckoutID(ctx context.Context, checkoutID string) (*domain.Payment, error)

	// GetByIdempotencyKey retrieves a payment by its idempotency key.
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Payment, error)

	// Update modifies an existing payment in the store.
	Update(ctx context.Context, payment *domain.Payment) error

	// ListByUserID returns payments for a given user with pagination support.
	// Returns the payment slice, the total count, and any error.
	ListByUserID(ctx context.Context, userID string, offset, limit int) ([]domain.Payment, int, error)

	// CreateRefund inserts a new refund into the store.
	CreateRefund(ctx context.Context, refund *domain.Refund) error

	// GetRefundByID retrieves a refund by its unique identifier.
	GetRefundByID(ctx context.Context, id string) (*domain.Refund, error)

	// ListRefundsByPaymentID returns all refunds for a given payment.
	ListRefundsByPaymentID(ctx context.Context, paymentID string) ([]domain.Refund, error)

	// UpdateRefund modifies an existing refund in the store.
	UpdateRefund(ctx context.Context, refund *domain.Refund) error
}
