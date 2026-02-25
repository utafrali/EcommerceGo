package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
	"github.com/utafrali/EcommerceGo/services/product/internal/repository"
)

// ProductRepository implements repository.ProductRepository using PostgreSQL.
type ProductRepository struct {
	pool *pgxpool.Pool
}

// NewProductRepository creates a new PostgreSQL-backed product repository.
func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{pool: pool}
}

// Create inserts a new product into the database.
func (r *ProductRepository) Create(ctx context.Context, p *domain.Product) error {
	metadataJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		INSERT INTO products (id, name, slug, description, brand_id, category_id, status, base_price, currency, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err = r.pool.Exec(ctx, query,
		p.ID,
		p.Name,
		p.Slug,
		p.Description,
		p.BrandID,
		p.CategoryID,
		p.Status,
		p.BasePrice,
		p.Currency,
		metadataJSON,
		p.CreatedAt,
		p.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return apperrors.AlreadyExists("product", "slug", p.Slug)
		}
		return fmt.Errorf("insert product: %w", err)
	}

	return nil
}

// GetByID retrieves a product by its ID.
func (r *ProductRepository) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	query := `
		SELECT id, name, slug, description, brand_id, category_id, status, base_price, currency, metadata, created_at, updated_at
		FROM products
		WHERE id = $1`

	return r.scanProduct(ctx, query, id)
}

// GetBySlug retrieves a product by its slug.
func (r *ProductRepository) GetBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	query := `
		SELECT id, name, slug, description, brand_id, category_id, status, base_price, currency, metadata, created_at, updated_at
		FROM products
		WHERE slug = $1`

	return r.scanProduct(ctx, query, slug)
}

// List returns products matching the given filter with the total count.
func (r *ProductRepository) List(ctx context.Context, filter repository.ProductFilter) ([]domain.Product, int, error) {
	var (
		conditions []string
		args       []any
		argIndex   = 1
	)

	if filter.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf("category_id = $%d", argIndex))
		args = append(args, *filter.CategoryID)
		argIndex++
	}

	if filter.BrandID != nil {
		conditions = append(conditions, fmt.Sprintf("brand_id = $%d", argIndex))
		args = append(args, *filter.BrandID)
		argIndex++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.Search != nil {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+*filter.Search+"%")
		argIndex++
	}

	if filter.MinPrice != nil {
		conditions = append(conditions, fmt.Sprintf("base_price >= $%d", argIndex))
		args = append(args, *filter.MinPrice)
		argIndex++
	}

	if filter.MaxPrice != nil {
		conditions = append(conditions, fmt.Sprintf("base_price <= $%d", argIndex))
		args = append(args, *filter.MaxPrice)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Use count(*) OVER() for total count in a single query.
	query := fmt.Sprintf(`
		SELECT id, name, slug, description, brand_id, category_id, status, base_price, currency, metadata, created_at, updated_at,
			   count(*) OVER() AS total_count
		FROM products
		%s
		ORDER BY created_at DESC
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
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var (
		products   []domain.Product
		totalCount int
	)

	for rows.Next() {
		var (
			p            domain.Product
			metadataJSON []byte
		)

		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Slug,
			&p.Description,
			&p.BrandID,
			&p.CategoryID,
			&p.Status,
			&p.BasePrice,
			&p.Currency,
			&metadataJSON,
			&p.CreatedAt,
			&p.UpdatedAt,
			&totalCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan product row: %w", err)
		}

		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &p.Metadata); err != nil {
				return nil, 0, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}

		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate product rows: %w", err)
	}

	if products == nil {
		products = []domain.Product{}
	}

	return products, totalCount, nil
}

// Update modifies an existing product in the database.
func (r *ProductRepository) Update(ctx context.Context, p *domain.Product) error {
	metadataJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	p.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE products
		SET name = $1, slug = $2, description = $3, brand_id = $4, category_id = $5,
		    status = $6, base_price = $7, currency = $8, metadata = $9, updated_at = $10
		WHERE id = $11`

	ct, err := r.pool.Exec(ctx, query,
		p.Name,
		p.Slug,
		p.Description,
		p.BrandID,
		p.CategoryID,
		p.Status,
		p.BasePrice,
		p.Currency,
		metadataJSON,
		p.UpdatedAt,
		p.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return apperrors.AlreadyExists("product", "slug", p.Slug)
		}
		return fmt.Errorf("update product: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("product", p.ID)
	}

	return nil
}

// Delete removes a product from the database by its ID.
func (r *ProductRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM products WHERE id = $1`

	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("product", id)
	}

	return nil
}

// scanProduct is a helper that executes a query expected to return a single product row.
func (r *ProductRepository) scanProduct(ctx context.Context, query string, args ...any) (*domain.Product, error) {
	var (
		p            domain.Product
		metadataJSON []byte
	)

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&p.ID,
		&p.Name,
		&p.Slug,
		&p.Description,
		&p.BrandID,
		&p.CategoryID,
		&p.Status,
		&p.BasePrice,
		&p.Currency,
		&metadataJSON,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan product: %w", err)
	}

	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &p.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}

	return &p, nil
}

// isUniqueViolation checks if the error is a PostgreSQL unique constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}
