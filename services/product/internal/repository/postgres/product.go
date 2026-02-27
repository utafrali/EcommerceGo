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
		args = append(args, "%"+escapeILIKE(*filter.Search)+"%")
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

	// Determine sort clause based on filter.
	orderClause := "created_at DESC"
	switch filter.SortBy {
	case domain.SortByNewest:
		orderClause = "created_at DESC"
	case domain.SortByPriceAsc:
		orderClause = "base_price ASC"
	case domain.SortByPriceDesc:
		orderClause = "base_price DESC"
	case domain.SortByNameAsc:
		orderClause = "name ASC"
	case domain.SortByNameDesc:
		orderClause = "name DESC"
	}

	// Use count(*) OVER() for total count in a single query.
	query := fmt.Sprintf(`
		SELECT id, name, slug, description, brand_id, category_id, status, base_price, currency, metadata, created_at, updated_at,
			   count(*) OVER() AS total_count
		FROM products
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`,
		whereClause, orderClause, argIndex, argIndex+1,
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

	var totalCount int
	products := make([]domain.Product, 0)

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

// GetImages returns all images for a product ordered by sort_order.
func (r *ProductRepository) GetImages(ctx context.Context, productID string) ([]domain.ProductImage, error) {
	query := `
		SELECT id, product_id, url, alt_text, sort_order, is_primary, created_at
		FROM product_images
		WHERE product_id = $1
		ORDER BY sort_order`

	rows, err := r.pool.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("get product images: %w", err)
	}
	defer rows.Close()

	var images []domain.ProductImage
	for rows.Next() {
		var img domain.ProductImage
		if err := rows.Scan(
			&img.ID,
			&img.ProductID,
			&img.URL,
			&img.AltText,
			&img.SortOrder,
			&img.IsPrimary,
			&img.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan product image row: %w", err)
		}
		images = append(images, img)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product image rows: %w", err)
	}

	if images == nil {
		images = []domain.ProductImage{}
	}

	return images, nil
}

// GetVariants returns all active variants for a product ordered by name.
func (r *ProductRepository) GetVariants(ctx context.Context, productID string) ([]domain.ProductVariant, error) {
	query := `
		SELECT id, product_id, sku, name, price, attributes, weight_grams, is_active, created_at, updated_at
		FROM product_variants
		WHERE product_id = $1 AND is_active = true
		ORDER BY name`

	rows, err := r.pool.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("get product variants: %w", err)
	}
	defer rows.Close()

	var variants []domain.ProductVariant
	for rows.Next() {
		var (
			v          domain.ProductVariant
			attrsJSON  []byte
		)
		if err := rows.Scan(
			&v.ID,
			&v.ProductID,
			&v.SKU,
			&v.Name,
			&v.Price,
			&attrsJSON,
			&v.WeightGrams,
			&v.IsActive,
			&v.CreatedAt,
			&v.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan product variant row: %w", err)
		}

		if attrsJSON != nil {
			if err := json.Unmarshal(attrsJSON, &v.Attributes); err != nil {
				return nil, fmt.Errorf("unmarshal variant attributes: %w", err)
			}
		}

		variants = append(variants, v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product variant rows: %w", err)
	}

	if variants == nil {
		variants = []domain.ProductVariant{}
	}

	return variants, nil
}

// GetCategory retrieves a single category by its ID.
func (r *ProductRepository) GetCategory(ctx context.Context, categoryID string) (*domain.Category, error) {
	query := `
		SELECT id, name, slug, parent_id, sort_order, is_active,
			image_url, icon_url, description, level, product_count, created_at, updated_at
		FROM categories
		WHERE id = $1`

	var c domain.Category
	err := r.pool.QueryRow(ctx, query, categoryID).Scan(
		&c.ID,
		&c.Name,
		&c.Slug,
		&c.ParentID,
		&c.SortOrder,
		&c.IsActive,
		&c.ImageURL,
		&c.IconURL,
		&c.Description,
		&c.Level,
		&c.ProductCount,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get category: %w", err)
	}

	return &c, nil
}

// GetBrand retrieves a single brand by its ID.
func (r *ProductRepository) GetBrand(ctx context.Context, brandID string) (*domain.Brand, error) {
	query := `
		SELECT id, name, slug, logo_url, created_at, updated_at
		FROM brands
		WHERE id = $1`

	var b domain.Brand
	err := r.pool.QueryRow(ctx, query, brandID).Scan(
		&b.ID,
		&b.Name,
		&b.Slug,
		&b.LogoURL,
		&b.CreatedAt,
		&b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get brand: %w", err)
	}

	return &b, nil
}

// GetPrimaryImages returns the primary image for each of the given product IDs.
// The returned map is keyed by product ID.
func (r *ProductRepository) GetPrimaryImages(ctx context.Context, productIDs []string) (map[string]domain.ProductImage, error) {
	if len(productIDs) == 0 {
		return map[string]domain.ProductImage{}, nil
	}

	// Build a parameterised IN clause.
	placeholders := make([]string, len(productIDs))
	args := make([]any, len(productIDs))
	for i, id := range productIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT ON (product_id) id, product_id, url, alt_text, sort_order, is_primary, created_at
		FROM product_images
		WHERE product_id IN (%s)
		ORDER BY product_id, is_primary DESC, sort_order`,
		strings.Join(placeholders, ", "),
	)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get primary images: %w", err)
	}
	defer rows.Close()

	result := make(map[string]domain.ProductImage)
	for rows.Next() {
		var img domain.ProductImage
		if err := rows.Scan(
			&img.ID,
			&img.ProductID,
			&img.URL,
			&img.AltText,
			&img.SortOrder,
			&img.IsPrimary,
			&img.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan primary image row: %w", err)
		}
		result[img.ProductID] = img
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate primary image rows: %w", err)
	}

	return result, nil
}

// escapeILIKE escapes special ILIKE pattern characters in user input
// so that %, _, and \ are treated as literal characters.
func escapeILIKE(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// isUniqueViolation checks if the error is a PostgreSQL unique constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}
