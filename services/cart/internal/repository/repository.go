package repository

import (
	"context"

	"github.com/utafrali/EcommerceGo/services/cart/internal/domain"
)

// CartRepository defines the interface for cart persistence operations.
type CartRepository interface {
	// Get retrieves a cart by its user ID.
	Get(ctx context.Context, userID string) (*domain.Cart, error)

	// Save persists a cart to the store, overwriting any existing cart for the user.
	Save(ctx context.Context, cart *domain.Cart) error

	// Delete removes a cart from the store by the user ID.
	Delete(ctx context.Context, userID string) error
}
