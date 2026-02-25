package repository

import (
	"context"

	"github.com/utafrali/EcommerceGo/services/order/internal/domain"
)

// OrderFilter defines filter criteria for listing orders.
type OrderFilter struct {
	UserID  *string
	Status  *string
	Page    int
	PerPage int
}

// OrderRepository defines the interface for order persistence operations.
type OrderRepository interface {
	// Create inserts a new order and its items into the store atomically.
	Create(ctx context.Context, order *domain.Order) error

	// GetByID retrieves an order by its unique identifier, including items.
	GetByID(ctx context.Context, id string) (*domain.Order, error)

	// List returns orders matching the given filter along with the total count.
	List(ctx context.Context, filter OrderFilter) ([]domain.Order, int, error)

	// UpdateStatus changes the status of an order and optionally sets a cancel reason.
	UpdateStatus(ctx context.Context, id string, status string, reason string) error
}
