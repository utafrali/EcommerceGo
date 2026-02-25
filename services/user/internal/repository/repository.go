package repository

import (
	"context"
	"time"

	"github.com/utafrali/EcommerceGo/services/user/internal/domain"
)

// UserRepository defines the interface for user persistence operations.
type UserRepository interface {
	// Create inserts a new user into the store.
	Create(ctx context.Context, user *domain.User) error

	// GetByID retrieves a user by their unique identifier.
	GetByID(ctx context.Context, id string) (*domain.User, error)

	// GetByEmail retrieves a user by their email address.
	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	// Update modifies an existing user in the store.
	Update(ctx context.Context, user *domain.User) error

	// Delete removes a user from the store by their identifier.
	Delete(ctx context.Context, id string) error
}

// AddressRepository defines the interface for address persistence operations.
type AddressRepository interface {
	// Create inserts a new address into the store.
	Create(ctx context.Context, address *domain.Address) error

	// GetByID retrieves an address by its unique identifier.
	GetByID(ctx context.Context, id string) (*domain.Address, error)

	// ListByUserID returns all addresses for the given user.
	ListByUserID(ctx context.Context, userID string) ([]domain.Address, error)

	// Update modifies an existing address in the store.
	Update(ctx context.Context, address *domain.Address) error

	// Delete removes an address from the store by its identifier.
	Delete(ctx context.Context, id string) error

	// SetDefault marks the specified address as the default for the user,
	// unsetting any previous default.
	SetDefault(ctx context.Context, userID, addressID string) error
}

// RefreshTokenRepository defines the interface for refresh token persistence operations.
type RefreshTokenRepository interface {
	// Create stores a new refresh token hash.
	Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error

	// GetByHash retrieves a refresh token record by its hash.
	GetByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)

	// RevokeByUserID revokes all refresh tokens for the given user.
	RevokeByUserID(ctx context.Context, userID string) error

	// Revoke revokes a specific refresh token by its hash.
	Revoke(ctx context.Context, tokenHash string) error
}
