package domain

import (
	"time"
)

// Allowed content types for media uploads.
var AllowedContentTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

// MaxFileSize is the maximum allowed file size in bytes (10 MB).
const MaxFileSize int64 = 10 * 1024 * 1024

// Owner type constants.
const (
	OwnerTypeProduct  = "product"
	OwnerTypeUser     = "user"
	OwnerTypeCategory = "category"
)

// MediaFile represents a media file in the system.
type MediaFile struct {
	ID           string         `json:"id"`
	OwnerID      string         `json:"owner_id"`
	OwnerType    string         `json:"owner_type"`
	FileName     string         `json:"file_name"`
	OriginalName string         `json:"original_name"`
	ContentType  string         `json:"content_type"`
	Size         int64          `json:"size"`
	URL          string         `json:"url"`
	ThumbnailURL *string        `json:"thumbnail_url,omitempty"`
	AltText      string         `json:"alt_text"`
	SortOrder    int            `json:"sort_order"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// IsAllowedContentType checks whether the given content type is allowed.
func IsAllowedContentType(contentType string) bool {
	return AllowedContentTypes[contentType]
}

// ValidOwnerTypes returns the set of valid owner types.
func ValidOwnerTypes() []string {
	return []string{OwnerTypeProduct, OwnerTypeUser, OwnerTypeCategory}
}

// IsValidOwnerType checks whether the given owner type is valid.
func IsValidOwnerType(ownerType string) bool {
	for _, t := range ValidOwnerTypes() {
		if t == ownerType {
			return true
		}
	}
	return false
}
