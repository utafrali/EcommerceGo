package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/user/internal/domain"
)

// WishlistRepository implements domain.WishlistRepository using PostgreSQL.
type WishlistRepository struct {
	pool *pgxpool.Pool
}

// NewWishlistRepository creates a new PostgreSQL-backed wishlist repository.
func NewWishlistRepository(pool *pgxpool.Pool) *WishlistRepository {
	return &WishlistRepository{pool: pool}
}

// Add inserts a product into the user's wishlist.
// Uses ON CONFLICT DO NOTHING for idempotent behavior.
func (r *WishlistRepository) Add(ctx context.Context, userID, productID string) error {
	query := `
		INSERT INTO wishlists (user_id, product_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, product_id) DO NOTHING`

	_, err := r.pool.Exec(ctx, query, userID, productID)
	if err != nil {
		return fmt.Errorf("add to wishlist: %w", err)
	}

	return nil
}

// Remove deletes a product from the user's wishlist.
func (r *WishlistRepository) Remove(ctx context.Context, userID, productID string) error {
	query := `DELETE FROM wishlists WHERE user_id = $1 AND product_id = $2`

	ct, err := r.pool.Exec(ctx, query, userID, productID)
	if err != nil {
		return fmt.Errorf("remove from wishlist: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("wishlist item", productID)
	}

	return nil
}

// List returns a paginated list of wishlist items for the user and the total count.
func (r *WishlistRepository) List(ctx context.Context, userID string, page, perPage int) ([]*domain.WishlistItem, int, error) {
	// Get total count.
	countQuery := `SELECT COUNT(*) FROM wishlists WHERE user_id = $1`

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count wishlist items: %w", err)
	}

	// Get paginated items.
	offset := (page - 1) * perPage
	query := `
		SELECT user_id, product_id, created_at
		FROM wishlists
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list wishlist items: %w", err)
	}
	defer rows.Close()

	var items []*domain.WishlistItem
	for rows.Next() {
		var item domain.WishlistItem
		if err := rows.Scan(&item.UserID, &item.ProductID, &item.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan wishlist item: %w", err)
		}
		items = append(items, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate wishlist rows: %w", err)
	}

	if items == nil {
		items = []*domain.WishlistItem{}
	}

	return items, total, nil
}

// Exists checks whether a product is in the user's wishlist.
func (r *WishlistRepository) Exists(ctx context.Context, userID, productID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM wishlists WHERE user_id = $1 AND product_id = $2)`

	var exists bool
	if err := r.pool.QueryRow(ctx, query, userID, productID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check wishlist item exists: %w", err)
	}

	return exists, nil
}
