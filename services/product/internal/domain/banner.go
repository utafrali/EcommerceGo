package domain

import (
	"context"
	"time"
)

// Banner position constants.
const (
	BannerPositionHeroSlider     = "hero_slider"
	BannerPositionMidBanner      = "mid_banner"
	BannerPositionCategoryBanner = "category_banner"
)

// Banner link type constants.
const (
	BannerLinkTypeInternal = "internal"
	BannerLinkTypeExternal = "external"
)

// Banner represents a promotional banner in the storefront.
type Banner struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Subtitle  *string    `json:"subtitle,omitempty"`
	ImageURL  string     `json:"image_url"`
	LinkURL   string     `json:"link_url"`
	LinkType  string     `json:"link_type"`
	Position  string     `json:"position"`
	SortOrder int        `json:"sort_order"`
	IsActive  bool       `json:"is_active"`
	StartsAt  *time.Time `json:"starts_at,omitempty"`
	EndsAt    *time.Time `json:"ends_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// CreateBannerInput holds the parameters for creating a banner.
type CreateBannerInput struct {
	Title     string     `json:"title"`
	Subtitle  *string    `json:"subtitle"`
	ImageURL  string     `json:"image_url"`
	LinkURL   string     `json:"link_url"`
	LinkType  string     `json:"link_type"`
	Position  string     `json:"position"`
	SortOrder int        `json:"sort_order"`
	IsActive  bool       `json:"is_active"`
	StartsAt  *time.Time `json:"starts_at"`
	EndsAt    *time.Time `json:"ends_at"`
}

// UpdateBannerInput holds the parameters for updating a banner.
type UpdateBannerInput struct {
	Title     *string    `json:"title"`
	Subtitle  *string    `json:"subtitle"`
	ImageURL  *string    `json:"image_url"`
	LinkURL   *string    `json:"link_url"`
	LinkType  *string    `json:"link_type"`
	Position  *string    `json:"position"`
	SortOrder *int       `json:"sort_order"`
	IsActive  *bool      `json:"is_active"`
	StartsAt  *time.Time `json:"starts_at"`
	EndsAt    *time.Time `json:"ends_at"`
}

// BannerFilter defines filter criteria for listing banners.
type BannerFilter struct {
	Position *string
	IsActive *bool
	Page     int
	PerPage  int
}

// BannerRepository defines the interface for banner persistence operations.
type BannerRepository interface {
	// Create inserts a new banner into the store.
	Create(ctx context.Context, banner *Banner) error

	// GetByID retrieves a banner by its unique identifier.
	GetByID(ctx context.Context, id string) (*Banner, error)

	// Update modifies an existing banner in the store.
	Update(ctx context.Context, banner *Banner) error

	// Delete removes a banner from the store by its identifier.
	Delete(ctx context.Context, id string) error

	// List returns banners matching the given filter along with the total count.
	List(ctx context.Context, filter BannerFilter) ([]Banner, int, error)
}

// ValidBannerPositions returns the set of valid banner positions.
func ValidBannerPositions() []string {
	return []string{BannerPositionHeroSlider, BannerPositionMidBanner, BannerPositionCategoryBanner}
}

// IsValidBannerPosition checks whether the given position string is a valid banner position.
func IsValidBannerPosition(position string) bool {
	for _, p := range ValidBannerPositions() {
		if p == position {
			return true
		}
	}
	return false
}

// ValidBannerLinkTypes returns the set of valid banner link types.
func ValidBannerLinkTypes() []string {
	return []string{BannerLinkTypeInternal, BannerLinkTypeExternal}
}

// IsValidBannerLinkType checks whether the given link type string is a valid banner link type.
func IsValidBannerLinkType(linkType string) bool {
	for _, t := range ValidBannerLinkTypes() {
		if t == linkType {
			return true
		}
	}
	return false
}
