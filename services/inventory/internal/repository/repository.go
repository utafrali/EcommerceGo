package repository

import (
	"context"

	"github.com/utafrali/EcommerceGo/services/inventory/internal/domain"
)

// StockRepository defines the interface for stock persistence operations.
type StockRepository interface {
	// GetByProductVariant retrieves stock for a specific product variant.
	GetByProductVariant(ctx context.Context, productID, variantID string) (*domain.Stock, error)

	// CreateStock inserts a new stock record or updates it if it already exists (idempotent).
	CreateStock(ctx context.Context, stock *domain.Stock) (*domain.Stock, error)

	// Upsert creates or updates stock for a product variant.
	Upsert(ctx context.Context, stock *domain.Stock) error

	// AdjustQuantity atomically adjusts the stock quantity by delta and records a movement.
	// If the stock record does not exist, it is created with the delta as the initial quantity.
	AdjustQuantity(ctx context.Context, productID, variantID string, delta int, reason string, refID *string) error

	// ListLowStock returns stock items where available quantity is below the threshold.
	ListLowStock(ctx context.Context, page, perPage int) ([]domain.Stock, int, error)

	// BulkCheck checks availability for multiple items at once.
	BulkCheck(ctx context.Context, items []domain.StockCheckItem) ([]domain.StockCheckResult, error)
}

// ReservationRepository defines the interface for reservation persistence operations.
type ReservationRepository interface {
	// Create inserts a new stock reservation.
	Create(ctx context.Context, reservation *domain.StockReservation) error

	// GetByID retrieves a reservation by its unique identifier.
	GetByID(ctx context.Context, id string) (*domain.StockReservation, error)

	// GetByCheckoutID retrieves all reservations for a checkout session.
	GetByCheckoutID(ctx context.Context, checkoutID string) ([]domain.StockReservation, error)

	// UpdateStatus updates the status of a reservation.
	UpdateStatus(ctx context.Context, id, status string) error

	// GetExpired returns all active reservations that have passed their expiration time.
	GetExpired(ctx context.Context) ([]domain.StockReservation, error)
}
