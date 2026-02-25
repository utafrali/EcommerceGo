package repository

import (
	"context"

	"github.com/utafrali/EcommerceGo/services/notification/internal/domain"
)

// NotificationRepository defines the interface for notification persistence operations.
type NotificationRepository interface {
	// Create inserts a new notification into the store.
	Create(ctx context.Context, notification *domain.Notification) error

	// GetByID retrieves a notification by its unique identifier.
	GetByID(ctx context.Context, id string) (*domain.Notification, error)

	// Update modifies an existing notification in the store.
	Update(ctx context.Context, notification *domain.Notification) error

	// ListByUserID returns notifications for a specific user with pagination.
	ListByUserID(ctx context.Context, userID string, offset, limit int) ([]domain.Notification, int, error)

	// ListPending returns pending notifications up to the given limit.
	ListPending(ctx context.Context, limit int) ([]domain.Notification, error)

	// ListFailed returns failed notifications up to the given limit.
	ListFailed(ctx context.Context, limit int) ([]domain.Notification, error)
}
