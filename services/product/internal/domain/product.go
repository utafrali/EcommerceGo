package domain

import (
	"time"
)

// Product status constants.
const (
	ProductStatusDraft     = "draft"
	ProductStatusPublished = "published"
	ProductStatusArchived  = "archived"
)

// Product represents a product in the catalog.
type Product struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	Description string         `json:"description"`
	BrandID     *string        `json:"brand_id,omitempty"`
	CategoryID  *string        `json:"category_id,omitempty"`
	Status      string         `json:"status"`
	BasePrice   int64          `json:"base_price"`
	Currency    string         `json:"currency"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// ProductVariant represents a specific variant of a product (e.g., size, color).
type ProductVariant struct {
	ID          string            `json:"id"`
	ProductID   string            `json:"product_id"`
	SKU         string            `json:"sku"`
	Name        string            `json:"name"`
	Price       *int64            `json:"price,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	WeightGrams *int              `json:"weight_grams,omitempty"`
	IsActive    bool              `json:"is_active"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// ProductImage represents an image associated with a product.
type ProductImage struct {
	ID        string    `json:"id"`
	ProductID string    `json:"product_id"`
	URL       string    `json:"url"`
	AltText   string    `json:"alt_text"`
	SortOrder int       `json:"sort_order"`
	IsPrimary bool      `json:"is_primary"`
	CreatedAt time.Time `json:"created_at"`
}

// ValidStatuses returns the set of valid product statuses.
func ValidStatuses() []string {
	return []string{ProductStatusDraft, ProductStatusPublished, ProductStatusArchived}
}

// IsValidStatus checks whether the given status string is a valid product status.
func IsValidStatus(status string) bool {
	for _, s := range ValidStatuses() {
		if s == status {
			return true
		}
	}
	return false
}
