package domain

import (
	"context"
	"time"
)

// WishlistItem represents a product saved in a user's wishlist.
type WishlistItem struct {
	UserID    string    `json:"user_id"`
	ProductID string    `json:"product_id"`
	CreatedAt time.Time `json:"created_at"`
}

// WishlistRepository defines the interface for wishlist persistence operations.
type WishlistRepository interface {
	// Add inserts a product into the user's wishlist (idempotent).
	Add(ctx context.Context, userID, productID string) error

	// Remove deletes a product from the user's wishlist.
	Remove(ctx context.Context, userID, productID string) error

	// List returns a paginated list of wishlist items for the user and the total count.
	List(ctx context.Context, userID string, page, perPage int) ([]*WishlistItem, int, error)

	// Exists checks whether a product is in the user's wishlist.
	Exists(ctx context.Context, userID, productID string) (bool, error)
}
