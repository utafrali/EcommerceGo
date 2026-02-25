package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/user/internal/domain"
)

// UserRepository implements repository.UserRepository using PostgreSQL.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new PostgreSQL-backed user repository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create inserts a new user into the database.
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, first_name, last_name, phone, role, is_active, email_verified, oauth_provider, oauth_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := r.pool.Exec(ctx, query,
		u.ID,
		u.Email,
		u.PasswordHash,
		u.FirstName,
		u.LastName,
		u.Phone,
		u.Role,
		u.IsActive,
		u.EmailVerified,
		u.OAuthProvider,
		u.OAuthID,
		u.CreatedAt,
		u.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return apperrors.AlreadyExists("user", "email", u.Email)
		}
		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by their ID.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, phone, role, is_active, email_verified, oauth_provider, oauth_id, created_at, updated_at
		FROM users
		WHERE id = $1`

	return r.scanUser(ctx, query, id)
}

// GetByEmail retrieves a user by their email address.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, phone, role, is_active, email_verified, oauth_provider, oauth_id, created_at, updated_at
		FROM users
		WHERE email = $1`

	return r.scanUser(ctx, query, email)
}

// Update modifies an existing user in the database.
func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	u.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE users
		SET email = $1, password_hash = $2, first_name = $3, last_name = $4, phone = $5,
		    role = $6, is_active = $7, email_verified = $8, oauth_provider = $9, oauth_id = $10, updated_at = $11
		WHERE id = $12`

	ct, err := r.pool.Exec(ctx, query,
		u.Email,
		u.PasswordHash,
		u.FirstName,
		u.LastName,
		u.Phone,
		u.Role,
		u.IsActive,
		u.EmailVerified,
		u.OAuthProvider,
		u.OAuthID,
		u.UpdatedAt,
		u.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return apperrors.AlreadyExists("user", "email", u.Email)
		}
		return fmt.Errorf("update user: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("user", u.ID)
	}

	return nil
}

// Delete removes a user from the database by their ID.
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`

	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("user", id)
	}

	return nil
}

// scanUser is a helper that executes a query expected to return a single user row.
func (r *UserRepository) scanUser(ctx context.Context, query string, args ...any) (*domain.User, error) {
	var u domain.User

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.FirstName,
		&u.LastName,
		&u.Phone,
		&u.Role,
		&u.IsActive,
		&u.EmailVerified,
		&u.OAuthProvider,
		&u.OAuthID,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}

	return &u, nil
}

// --- Address Repository ---

// AddressRepository implements repository.AddressRepository using PostgreSQL.
type AddressRepository struct {
	pool *pgxpool.Pool
}

// NewAddressRepository creates a new PostgreSQL-backed address repository.
func NewAddressRepository(pool *pgxpool.Pool) *AddressRepository {
	return &AddressRepository{pool: pool}
}

// Create inserts a new address into the database.
func (r *AddressRepository) Create(ctx context.Context, a *domain.Address) error {
	query := `
		INSERT INTO addresses (id, user_id, label, first_name, last_name, address_line1, address_line2, city, state, postal_code, country_code, phone, is_default, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := r.pool.Exec(ctx, query,
		a.ID,
		a.UserID,
		a.Label,
		a.FirstName,
		a.LastName,
		a.AddressLine1,
		a.AddressLine2,
		a.City,
		a.State,
		a.PostalCode,
		a.CountryCode,
		a.Phone,
		a.IsDefault,
		a.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert address: %w", err)
	}

	return nil
}

// GetByID retrieves an address by its ID.
func (r *AddressRepository) GetByID(ctx context.Context, id string) (*domain.Address, error) {
	query := `
		SELECT id, user_id, label, first_name, last_name, address_line1, address_line2, city, state, postal_code, country_code, phone, is_default, created_at
		FROM addresses
		WHERE id = $1`

	var a domain.Address
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID,
		&a.UserID,
		&a.Label,
		&a.FirstName,
		&a.LastName,
		&a.AddressLine1,
		&a.AddressLine2,
		&a.City,
		&a.State,
		&a.PostalCode,
		&a.CountryCode,
		&a.Phone,
		&a.IsDefault,
		&a.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan address: %w", err)
	}

	return &a, nil
}

// ListByUserID returns all addresses for the given user.
func (r *AddressRepository) ListByUserID(ctx context.Context, userID string) ([]domain.Address, error) {
	query := `
		SELECT id, user_id, label, first_name, last_name, address_line1, address_line2, city, state, postal_code, country_code, phone, is_default, created_at
		FROM addresses
		WHERE user_id = $1
		ORDER BY is_default DESC, created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list addresses: %w", err)
	}
	defer rows.Close()

	var addresses []domain.Address
	for rows.Next() {
		var a domain.Address
		if err := rows.Scan(
			&a.ID,
			&a.UserID,
			&a.Label,
			&a.FirstName,
			&a.LastName,
			&a.AddressLine1,
			&a.AddressLine2,
			&a.City,
			&a.State,
			&a.PostalCode,
			&a.CountryCode,
			&a.Phone,
			&a.IsDefault,
			&a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan address row: %w", err)
		}
		addresses = append(addresses, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate address rows: %w", err)
	}

	if addresses == nil {
		addresses = []domain.Address{}
	}

	return addresses, nil
}

// Update modifies an existing address in the database.
func (r *AddressRepository) Update(ctx context.Context, a *domain.Address) error {
	query := `
		UPDATE addresses
		SET label = $1, first_name = $2, last_name = $3, address_line1 = $4, address_line2 = $5,
		    city = $6, state = $7, postal_code = $8, country_code = $9, phone = $10
		WHERE id = $11`

	ct, err := r.pool.Exec(ctx, query,
		a.Label,
		a.FirstName,
		a.LastName,
		a.AddressLine1,
		a.AddressLine2,
		a.City,
		a.State,
		a.PostalCode,
		a.CountryCode,
		a.Phone,
		a.ID,
	)
	if err != nil {
		return fmt.Errorf("update address: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("address", a.ID)
	}

	return nil
}

// Delete removes an address from the database by its ID.
func (r *AddressRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM addresses WHERE id = $1`

	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete address: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("address", id)
	}

	return nil
}

// SetDefault marks the specified address as the default for the user,
// unsetting any previous default within a transaction.
func (r *AddressRepository) SetDefault(ctx context.Context, userID, addressID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Unset any existing default for this user.
	_, err = tx.Exec(ctx,
		`UPDATE addresses SET is_default = false WHERE user_id = $1 AND is_default = true`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("unset default address: %w", err)
	}

	// Set the new default.
	ct, err := tx.Exec(ctx,
		`UPDATE addresses SET is_default = true WHERE id = $1 AND user_id = $2`,
		addressID, userID,
	)
	if err != nil {
		return fmt.Errorf("set default address: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("address", addressID)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// --- Refresh Token Repository ---

// RefreshTokenRepository implements repository.RefreshTokenRepository using PostgreSQL.
type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

// NewRefreshTokenRepository creates a new PostgreSQL-backed refresh token repository.
func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

// Create stores a new refresh token hash in the database.
func (r *RefreshTokenRepository) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4)`

	_, err := r.pool.Exec(ctx, query, userID, tokenHash, expiresAt, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}

	return nil
}

// GetByHash retrieves a refresh token record by its hash.
func (r *RefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1`

	var rt domain.RefreshToken
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&rt.ID,
		&rt.UserID,
		&rt.TokenHash,
		&rt.ExpiresAt,
		&rt.CreatedAt,
		&rt.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan refresh token: %w", err)
	}

	return &rt, nil
}

// RevokeByUserID revokes all refresh tokens for the given user.
func (r *RefreshTokenRepository) RevokeByUserID(ctx context.Context, userID string) error {
	query := `UPDATE refresh_tokens SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL`

	_, err := r.pool.Exec(ctx, query, time.Now().UTC(), userID)
	if err != nil {
		return fmt.Errorf("revoke refresh tokens by user: %w", err)
	}

	return nil
}

// Revoke revokes a specific refresh token by its hash.
func (r *RefreshTokenRepository) Revoke(ctx context.Context, tokenHash string) error {
	query := `UPDATE refresh_tokens SET revoked_at = $1 WHERE token_hash = $2 AND revoked_at IS NULL`

	_, err := r.pool.Exec(ctx, query, time.Now().UTC(), tokenHash)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

// isUniqueViolation checks if the error is a PostgreSQL unique constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}
