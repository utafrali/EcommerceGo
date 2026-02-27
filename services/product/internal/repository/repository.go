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
	SortBy     string
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

	// GetImages returns all images for a product ordered by sort_order.
	GetImages(ctx context.Context, productID string) ([]domain.ProductImage, error)

	// GetVariants returns all active variants for a product ordered by name.
	GetVariants(ctx context.Context, productID string) ([]domain.ProductVariant, error)

	// GetCategory retrieves a single category by its ID.
	GetCategory(ctx context.Context, categoryID string) (*domain.Category, error)

	// GetBrand retrieves a single brand by its ID.
	GetBrand(ctx context.Context, brandID string) (*domain.Brand, error)

	// GetPrimaryImages returns the primary image for each of the given product IDs.
	// The returned map is keyed by product ID.
	GetPrimaryImages(ctx context.Context, productIDs []string) (map[string]domain.ProductImage, error)
}

// ReviewRepository defines the interface for review persistence operations.
type ReviewRepository interface {
	// Create inserts a new product review into the store.
	Create(ctx context.Context, review *domain.Review) error

	// ListByProductID returns paginated reviews for a given product along with the total count.
	ListByProductID(ctx context.Context, productID string, page, perPage int) ([]domain.Review, int, error)

	// GetSummary returns the average rating and total count of reviews for a product.
	GetSummary(ctx context.Context, productID string) (*domain.ReviewSummary, error)
}
