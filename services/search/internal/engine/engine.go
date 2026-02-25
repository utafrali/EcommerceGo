package engine

import (
	"context"

	"github.com/utafrali/EcommerceGo/services/search/internal/domain"
)

// SearchEngine defines the interface for indexing and searching products.
// Implementations may use Elasticsearch, in-memory storage, or other backends.
type SearchEngine interface {
	// Index adds or updates a single product in the search index.
	Index(ctx context.Context, product *domain.SearchableProduct) error

	// Delete removes a product from the search index by its ID.
	Delete(ctx context.Context, id string) error

	// Search executes a search query and returns matching products.
	Search(ctx context.Context, query *domain.SearchQuery) (*domain.SearchResult, error)

	// BulkIndex adds or updates multiple products in the search index.
	BulkIndex(ctx context.Context, products []domain.SearchableProduct) error
}
