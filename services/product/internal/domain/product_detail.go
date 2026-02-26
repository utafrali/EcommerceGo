package domain

// ProductDetail is an enriched product response containing images, variants,
// category, and brand information alongside the base product fields.
type ProductDetail struct {
	Product
	Images   []ProductImage   `json:"images"`
	Variants []ProductVariant `json:"variants"`
	Category *Category        `json:"category,omitempty"`
	Brand    *Brand           `json:"brand,omitempty"`
}

// ProductListItem is a product summary for list endpoints that includes the
// primary image when available.
type ProductListItem struct {
	Product
	PrimaryImage *ProductImage `json:"primary_image,omitempty"`
}
