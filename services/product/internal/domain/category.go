package domain

import (
	"context"
	"time"
)

// Category represents a product category with optional hierarchical nesting.
type Category struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Slug         string      `json:"slug"`
	ParentID     *string     `json:"parent_id,omitempty"`
	SortOrder    int         `json:"sort_order"`
	IsActive     bool        `json:"is_active"`
	ImageURL     *string     `json:"image_url,omitempty"`
	IconURL      *string     `json:"icon_url,omitempty"`
	Description  *string     `json:"description,omitempty"`
	Level        int         `json:"level"`
	ProductCount int         `json:"product_count"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	Children     []*Category `json:"children,omitempty"`
}

// CreateCategoryInput holds the parameters for creating a category.
type CreateCategoryInput struct {
	Name        string  `json:"name" validate:"required,min=1,max=255"`
	ParentID    *string `json:"parent_id" validate:"omitempty,uuid"`
	SortOrder   int     `json:"sort_order" validate:"gte=0"`
	IsActive    *bool   `json:"is_active"`
	ImageURL    *string `json:"image_url" validate:"omitempty,url"`
	IconURL     *string `json:"icon_url" validate:"omitempty,url"`
	Description *string `json:"description"`
}

// UpdateCategoryInput holds the parameters for updating a category.
type UpdateCategoryInput struct {
	Name        *string `json:"name" validate:"omitempty,min=1,max=255"`
	ParentID    *string `json:"parent_id" validate:"omitempty,uuid"`
	SortOrder   *int    `json:"sort_order" validate:"omitempty,gte=0"`
	IsActive    *bool   `json:"is_active"`
	ImageURL    *string `json:"image_url" validate:"omitempty"`
	IconURL     *string `json:"icon_url" validate:"omitempty"`
	Description *string `json:"description"`
}

// CategoryRepository defines the interface for category persistence operations.
type CategoryRepository interface {
	// Create inserts a new category into the store.
	Create(ctx context.Context, category *Category) error

	// GetByID retrieves a category by its unique identifier.
	GetByID(ctx context.Context, id string) (*Category, error)

	// GetBySlug retrieves a category by its URL-friendly slug.
	GetBySlug(ctx context.Context, slug string) (*Category, error)

	// Update modifies an existing category in the store.
	Update(ctx context.Context, category *Category) error

	// Delete removes a category from the store by its identifier.
	Delete(ctx context.Context, id string) error

	// ListAll returns all active categories as a flat list.
	ListAll(ctx context.Context) ([]Category, error)

	// ListTree returns all active categories assembled into a nested tree.
	ListTree(ctx context.Context) ([]*Category, error)
}
