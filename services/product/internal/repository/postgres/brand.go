package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
)

// BrandRepository implements brand persistence operations using PostgreSQL.
type BrandRepository struct {
	pool *pgxpool.Pool
}

// NewBrandRepository creates a new PostgreSQL-backed brand repository.
func NewBrandRepository(pool *pgxpool.Pool) *BrandRepository {
	return &BrandRepository{pool: pool}
}

// ListAll returns all brands ordered by name.
func (r *BrandRepository) ListAll(ctx context.Context) ([]domain.Brand, error) {
	query := `
		SELECT id, name, slug, logo_url, created_at, updated_at
		FROM brands
		ORDER BY name`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list brands: %w", err)
	}
	defer rows.Close()

	var brands []domain.Brand

	for rows.Next() {
		var b domain.Brand

		if err := rows.Scan(
			&b.ID,
			&b.Name,
			&b.Slug,
			&b.LogoURL,
			&b.CreatedAt,
			&b.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan brand row: %w", err)
		}

		brands = append(brands, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate brand rows: %w", err)
	}

	if brands == nil {
		brands = []domain.Brand{}
	}

	return brands, nil
}
