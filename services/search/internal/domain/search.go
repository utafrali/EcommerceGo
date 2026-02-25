package domain

import (
	"time"
)

// SearchableProduct represents a product document in the search index.
type SearchableProduct struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Slug         string            `json:"slug"`
	Description  string            `json:"description"`
	CategoryID   string            `json:"category_id"`
	CategoryName string            `json:"category_name"`
	BrandID      string            `json:"brand_id"`
	BrandName    string            `json:"brand_name"`
	BasePrice    int64             `json:"base_price"`
	Currency     string            `json:"currency"`
	Status       string            `json:"status"`
	ImageURL     string            `json:"image_url"`
	Tags         []string          `json:"tags"`
	Attributes   map[string]string `json:"attributes"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// Sort options for search results.
const (
	SortRelevance = "relevance"
	SortPriceAsc  = "price_asc"
	SortPriceDesc = "price_desc"
	SortNewest    = "newest"
)

// ValidSortOptions returns the list of valid sort options.
func ValidSortOptions() []string {
	return []string{SortRelevance, SortPriceAsc, SortPriceDesc, SortNewest}
}

// IsValidSort checks whether the given sort string is a valid sort option.
func IsValidSort(sort string) bool {
	for _, s := range ValidSortOptions() {
		if s == sort {
			return true
		}
	}
	return false
}

// SearchQuery holds all parameters for a search request.
type SearchQuery struct {
	Query      string  `json:"query"`
	CategoryID *string `json:"category_id,omitempty"`
	BrandID    *string `json:"brand_id,omitempty"`
	MinPrice   *int64  `json:"min_price,omitempty"`
	MaxPrice   *int64  `json:"max_price,omitempty"`
	Status     *string `json:"status,omitempty"`
	SortBy     string  `json:"sort_by"`
	Page       int     `json:"page"`
	PerPage    int     `json:"per_page"`
}

// SearchResult holds the paginated search response.
type SearchResult struct {
	Products []SearchableProduct `json:"products"`
	Total    int                 `json:"total"`
	Page     int                 `json:"page"`
	PerPage  int                 `json:"per_page"`
	TookMs   int64               `json:"took_ms"`
}
