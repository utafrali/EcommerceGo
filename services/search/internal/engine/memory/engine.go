package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/utafrali/EcommerceGo/services/search/internal/domain"
)

// Engine is an in-memory implementation of the SearchEngine interface.
// It provides simple string matching on name and description fields.
// Thread-safe via sync.RWMutex.
type Engine struct {
	mu       sync.RWMutex
	products map[string]domain.SearchableProduct
}

// New creates a new in-memory search engine.
func New() *Engine {
	return &Engine{
		products: make(map[string]domain.SearchableProduct),
	}
}

// Index adds or updates a single product in the in-memory index.
func (e *Engine) Index(_ context.Context, product *domain.SearchableProduct) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.products[product.ID] = *product
	return nil
}

// Delete removes a product from the in-memory index by its ID.
func (e *Engine) Delete(_ context.Context, id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.products, id)
	return nil
}

// Search executes a search query against the in-memory index.
func (e *Engine) Search(_ context.Context, query *domain.SearchQuery) (*domain.SearchResult, error) {
	start := time.Now()

	e.mu.RLock()
	defer e.mu.RUnlock()

	matched := make([]domain.SearchableProduct, 0)

	queryLower := strings.ToLower(query.Query)

	for _, p := range e.products {
		if !e.matches(p, query, queryLower) {
			continue
		}
		matched = append(matched, p)
	}

	// Sort the results.
	e.sortProducts(matched, query.SortBy)

	total := len(matched)

	// Apply pagination.
	page := query.Page
	if page < 1 {
		page = 1
	}
	perPage := query.PerPage
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	offset := (page - 1) * perPage
	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}

	tookMs := time.Since(start).Milliseconds()

	return &domain.SearchResult{
		Products: matched[offset:end],
		Total:    total,
		Page:     page,
		PerPage:  perPage,
		TookMs:   tookMs,
	}, nil
}

// BulkIndex adds or updates multiple products in the in-memory index.
func (e *Engine) BulkIndex(_ context.Context, products []domain.SearchableProduct) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i := range products {
		e.products[products[i].ID] = products[i]
	}
	return nil
}

// matches checks whether a product matches the given search query filters.
func (e *Engine) matches(p domain.SearchableProduct, query *domain.SearchQuery, queryLower string) bool {
	// Full-text match on name and description.
	if queryLower != "" {
		nameLower := strings.ToLower(p.Name)
		descLower := strings.ToLower(p.Description)
		if !strings.Contains(nameLower, queryLower) && !strings.Contains(descLower, queryLower) {
			return false
		}
	}

	// Category filter.
	if query.CategoryID != nil && *query.CategoryID != "" {
		if p.CategoryID != *query.CategoryID {
			return false
		}
	}

	// Brand filter.
	if query.BrandID != nil && *query.BrandID != "" {
		if p.BrandID != *query.BrandID {
			return false
		}
	}

	// Price range filter.
	if query.MinPrice != nil {
		if p.BasePrice < *query.MinPrice {
			return false
		}
	}
	if query.MaxPrice != nil {
		if p.BasePrice > *query.MaxPrice {
			return false
		}
	}

	// Status filter.
	if query.Status != nil && *query.Status != "" {
		if p.Status != *query.Status {
			return false
		}
	}

	return true
}

// sortProducts sorts the matched products based on the sort option.
func (e *Engine) sortProducts(products []domain.SearchableProduct, sortBy string) {
	switch sortBy {
	case domain.SortPriceAsc:
		sort.Slice(products, func(i, j int) bool {
			return products[i].BasePrice < products[j].BasePrice
		})
	case domain.SortPriceDesc:
		sort.Slice(products, func(i, j int) bool {
			return products[i].BasePrice > products[j].BasePrice
		})
	case domain.SortNewest:
		sort.Slice(products, func(i, j int) bool {
			return products[i].CreatedAt.After(products[j].CreatedAt)
		})
	default:
		// SortRelevance or unknown: keep insertion order (no additional sort).
	}
}
