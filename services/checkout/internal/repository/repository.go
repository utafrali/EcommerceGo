package repository

import (
	"context"
	"time"

	"github.com/utafrali/EcommerceGo/services/checkout/internal/domain"
)

// CheckoutRepository defines the interface for checkout session persistence operations.
type CheckoutRepository interface {
	// Create inserts a new checkout session into the store.
	Create(ctx context.Context, session *domain.CheckoutSession) error

	// GetByID retrieves a checkout session by its unique identifier.
	GetByID(ctx context.Context, id string) (*domain.CheckoutSession, error)

	// Update modifies an existing checkout session in the store.
	Update(ctx context.Context, session *domain.CheckoutSession) error

	// GetActiveByUserID retrieves the active (non-terminal) checkout session for a user.
	GetActiveByUserID(ctx context.Context, userID string) (*domain.CheckoutSession, error)

	// ListExpired returns all checkout sessions that have expired before the given time.
	ListExpired(ctx context.Context, before time.Time) ([]domain.CheckoutSession, error)
}
