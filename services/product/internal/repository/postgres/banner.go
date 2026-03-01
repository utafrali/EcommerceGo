package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/utafrali/EcommerceGo/pkg/database"
	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
)

// BannerRepository implements domain.BannerRepository using PostgreSQL.
type BannerRepository struct {
	pool database.DBTX
}

// NewBannerRepository creates a new PostgreSQL-backed banner repository.
func NewBannerRepository(pool database.DBTX) *BannerRepository {
	return &BannerRepository{pool: pool}
}

// Create inserts a new banner into the database.
func (r *BannerRepository) Create(ctx context.Context, b *domain.Banner) error {
	query := `
		INSERT INTO banners (id, title, subtitle, image_url, link_url, link_type, position, sort_order, is_active, starts_at, ends_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := r.pool.Exec(ctx, query,
		b.ID,
		b.Title,
		b.Subtitle,
		b.ImageURL,
		b.LinkURL,
		b.LinkType,
		b.Position,
		b.SortOrder,
		b.IsActive,
		b.StartsAt,
		b.EndsAt,
		b.CreatedAt,
		b.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert banner: %w", err)
	}

	return nil
}

// GetByID retrieves a banner by its ID.
func (r *BannerRepository) GetByID(ctx context.Context, id string) (*domain.Banner, error) {
	query := `
		SELECT id, title, subtitle, image_url, link_url, link_type, position, sort_order, is_active, starts_at, ends_at, created_at, updated_at
		FROM banners
		WHERE id = $1`

	return r.scanBanner(ctx, query, id)
}

// Update modifies an existing banner in the database.
func (r *BannerRepository) Update(ctx context.Context, b *domain.Banner) error {
	b.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE banners
		SET title = $1, subtitle = $2, image_url = $3, link_url = $4, link_type = $5,
		    position = $6, sort_order = $7, is_active = $8, starts_at = $9, ends_at = $10, updated_at = $11
		WHERE id = $12`

	ct, err := r.pool.Exec(ctx, query,
		b.Title,
		b.Subtitle,
		b.ImageURL,
		b.LinkURL,
		b.LinkType,
		b.Position,
		b.SortOrder,
		b.IsActive,
		b.StartsAt,
		b.EndsAt,
		b.UpdatedAt,
		b.ID,
	)
	if err != nil {
		return fmt.Errorf("update banner: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("banner", b.ID)
	}

	return nil
}

// Delete removes a banner from the database by its ID.
func (r *BannerRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM banners WHERE id = $1`

	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete banner: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("banner", id)
	}

	return nil
}

// List returns banners matching the given filter with the total count.
func (r *BannerRepository) List(ctx context.Context, filter domain.BannerFilter) ([]domain.Banner, int, error) {
	var (
		conditions []string
		args       []any
		argIndex   = 1
	)

	if filter.Position != nil {
		conditions = append(conditions, fmt.Sprintf("position = $%d", argIndex))
		args = append(args, *filter.Position)
		argIndex++
	}

	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Use count(*) OVER() for total count in a single query.
	query := fmt.Sprintf(`
		SELECT id, title, subtitle, image_url, link_url, link_type, position, sort_order, is_active, starts_at, ends_at, created_at, updated_at,
			   count(*) OVER() AS total_count
		FROM banners
		%s
		ORDER BY sort_order, created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIndex, argIndex+1,
	)

	limit := filter.PerPage
	if limit <= 0 {
		limit = 20
	}
	offset := 0
	if filter.Page > 1 {
		offset = (filter.Page - 1) * limit
	}

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list banners: %w", err)
	}
	defer rows.Close()

	var (
		banners    []domain.Banner
		totalCount int
	)

	for rows.Next() {
		var b domain.Banner

		if err := rows.Scan(
			&b.ID,
			&b.Title,
			&b.Subtitle,
			&b.ImageURL,
			&b.LinkURL,
			&b.LinkType,
			&b.Position,
			&b.SortOrder,
			&b.IsActive,
			&b.StartsAt,
			&b.EndsAt,
			&b.CreatedAt,
			&b.UpdatedAt,
			&totalCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan banner row: %w", err)
		}

		banners = append(banners, b)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate banner rows: %w", err)
	}

	if banners == nil {
		banners = []domain.Banner{}
	}

	return banners, totalCount, nil
}

// scanBanner is a helper that executes a query expected to return a single banner row.
func (r *BannerRepository) scanBanner(ctx context.Context, query string, args ...any) (*domain.Banner, error) {
	var b domain.Banner

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&b.ID,
		&b.Title,
		&b.Subtitle,
		&b.ImageURL,
		&b.LinkURL,
		&b.LinkType,
		&b.Position,
		&b.SortOrder,
		&b.IsActive,
		&b.StartsAt,
		&b.EndsAt,
		&b.CreatedAt,
		&b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan banner: %w", err)
	}

	return &b, nil
}
