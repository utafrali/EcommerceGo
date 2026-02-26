package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
)

// CategoryRepository implements category persistence operations using PostgreSQL.
type CategoryRepository struct {
	pool *pgxpool.Pool
}

// NewCategoryRepository creates a new PostgreSQL-backed category repository.
func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool: pool}
}

// ListAll returns all active categories ordered by sort_order and name.
func (r *CategoryRepository) ListAll(ctx context.Context) ([]domain.Category, error) {
	query := `
		SELECT id, name, slug, parent_id, sort_order, is_active
		FROM categories
		WHERE is_active = true
		ORDER BY sort_order, name`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var categories []domain.Category

	for rows.Next() {
		var c domain.Category

		if err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.Slug,
			&c.ParentID,
			&c.SortOrder,
			&c.IsActive,
		); err != nil {
			return nil, fmt.Errorf("scan category row: %w", err)
		}

		categories = append(categories, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate category rows: %w", err)
	}

	if categories == nil {
		categories = []domain.Category{}
	}

	return categories, nil
}
