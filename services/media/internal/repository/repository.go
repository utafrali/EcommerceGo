package repository

import (
	"context"

	"github.com/utafrali/EcommerceGo/services/media/internal/domain"
)

// MediaRepository defines the interface for media file persistence operations.
type MediaRepository interface {
	// Create inserts a new media file record into the store.
	Create(ctx context.Context, media *domain.MediaFile) error

	// GetByID retrieves a media file by its unique identifier.
	GetByID(ctx context.Context, id string) (*domain.MediaFile, error)

	// ListByOwner returns media files for a given owner with pagination.
	// Returns the list of media files and the total count.
	ListByOwner(ctx context.Context, ownerID, ownerType string, offset, limit int) ([]domain.MediaFile, int, error)

	// Update modifies an existing media file record in the store.
	Update(ctx context.Context, media *domain.MediaFile) error

	// Delete removes a media file record from the store by its identifier.
	Delete(ctx context.Context, id string) error
}
