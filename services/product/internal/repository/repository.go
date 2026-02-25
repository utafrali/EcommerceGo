package repository

import (
	"context"

	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
)

// ProductFilter defines filter criteria for listing products.
type ProductFilter struct {
	CategoryID *string
	BrandID    *string
	Status     *string
	Search     *string
	MinPrice   *int64
	MaxPrice   *int64
	Page       int
	PerPage    int
}

// ProductRepository defines the interface for product persistence operations.
type ProductRepository interface {
	// Create inserts a new product into the store.
	Create(ctx context.Context, product *domain.Product) error

	// GetByID retrieves a product by its unique identifier.
	GetByID(ctx context.Context, id string) (*domain.Product, error)

	// GetBySlug retrieves a product by its URL-friendly slug.
	GetBySlug(ctx context.Context, slug string) (*domain.Product, error)

	// List returns products matching the given filter along with the total count.
	List(ctx context.Context, filter ProductFilter) ([]domain.Product, int, error)

	// Update modifies an existing product in the store.
	Update(ctx context.Context, product *domain.Product) error

	// Delete removes a product from the store by its identifier.
	Delete(ctx context.Context, id string) error
}
