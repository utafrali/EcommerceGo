package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/utafrali/EcommerceGo/pkg/database"
	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
)

// categoryColumns is the standard SELECT column list for categories.
const categoryColumns = `id, name, slug, parent_id, sort_order, is_active,
	image_url, icon_url, description, level, product_count, created_at, updated_at`

// CategoryRepository implements category persistence operations using PostgreSQL.
type CategoryRepository struct {
	pool database.DBTX
}

// NewCategoryRepository creates a new PostgreSQL-backed category repository.
func NewCategoryRepository(pool database.DBTX) *CategoryRepository {
	return &CategoryRepository{pool: pool}
}

// Create inserts a new category into the database.
func (r *CategoryRepository) Create(ctx context.Context, c *domain.Category) error {
	query := `
		INSERT INTO categories (id, name, slug, parent_id, sort_order, is_active,
			image_url, icon_url, description, level, product_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := r.pool.Exec(ctx, query,
		c.ID,
		c.Name,
		c.Slug,
		c.ParentID,
		c.SortOrder,
		c.IsActive,
		c.ImageURL,
		c.IconURL,
		c.Description,
		c.Level,
		c.ProductCount,
		c.CreatedAt,
		c.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return apperrors.AlreadyExists("category", "slug", c.Slug)
		}
		return fmt.Errorf("insert category: %w", err)
	}

	return nil
}

// GetByID retrieves a category by its unique identifier.
func (r *CategoryRepository) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	query := fmt.Sprintf(`SELECT %s FROM categories WHERE id = $1`, categoryColumns)
	return r.scanCategory(ctx, query, id)
}

// GetBySlug retrieves a category by its URL-friendly slug.
func (r *CategoryRepository) GetBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	query := fmt.Sprintf(`SELECT %s FROM categories WHERE slug = $1`, categoryColumns)
	return r.scanCategory(ctx, query, slug)
}

// Update modifies an existing category in the database.
func (r *CategoryRepository) Update(ctx context.Context, c *domain.Category) error {
	c.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE categories
		SET name = $1, slug = $2, parent_id = $3, sort_order = $4, is_active = $5,
		    image_url = $6, icon_url = $7, description = $8, level = $9,
		    product_count = $10, updated_at = $11
		WHERE id = $12`

	ct, err := r.pool.Exec(ctx, query,
		c.Name,
		c.Slug,
		c.ParentID,
		c.SortOrder,
		c.IsActive,
		c.ImageURL,
		c.IconURL,
		c.Description,
		c.Level,
		c.ProductCount,
		c.UpdatedAt,
		c.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return apperrors.AlreadyExists("category", "slug", c.Slug)
		}
		return fmt.Errorf("update category: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("category", c.ID)
	}

	return nil
}

// Delete removes a category from the database by its ID.
func (r *CategoryRepository) Delete(ctx context.Context, id string) error {
	// First, re-parent any children to the deleted category's parent.
	parent, err := r.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get category for delete: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`UPDATE categories SET parent_id = $1 WHERE parent_id = $2`,
		parent.ParentID, id,
	)
	if err != nil {
		return fmt.Errorf("reparent child categories: %w", err)
	}

	ct, err := r.pool.Exec(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("category", id)
	}

	return nil
}

// ListAll returns all active categories as a flat list ordered by sort_order and name.
func (r *CategoryRepository) ListAll(ctx context.Context) ([]domain.Category, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM categories
		WHERE is_active = true
		ORDER BY sort_order, name`, categoryColumns)

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var categories []domain.Category

	for rows.Next() {
		var c domain.Category
		if err := scanCategoryRow(rows, &c); err != nil {
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

// ListTree returns all active categories assembled into a nested tree structure.
// Root categories (parent_id IS NULL) appear at the top level; children are nested
// under their respective parents recursively.
func (r *CategoryRepository) ListTree(ctx context.Context) ([]*domain.Category, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM categories
		WHERE is_active = true
		ORDER BY level, sort_order, name`, categoryColumns)

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list categories for tree: %w", err)
	}
	defer rows.Close()

	var allCategories []*domain.Category
	categoryMap := make(map[string]*domain.Category)

	for rows.Next() {
		c := &domain.Category{}
		if err := scanCategoryRow(rows, c); err != nil {
			return nil, fmt.Errorf("scan category row for tree: %w", err)
		}
		c.Children = []*domain.Category{} // Initialize empty children slice.
		allCategories = append(allCategories, c)
		categoryMap[c.ID] = c
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate category rows for tree: %w", err)
	}

	// Build the tree by assigning children to their parents.
	var roots []*domain.Category

	for _, c := range allCategories {
		if c.ParentID == nil {
			roots = append(roots, c)
		} else {
			if parent, ok := categoryMap[*c.ParentID]; ok {
				parent.Children = append(parent.Children, c)
			} else {
				// Orphan category (parent inactive or missing): treat as root.
				roots = append(roots, c)
			}
		}
	}

	if roots == nil {
		roots = []*domain.Category{}
	}

	return roots, nil
}

// scanCategory executes a query expected to return a single category row.
func (r *CategoryRepository) scanCategory(ctx context.Context, query string, args ...any) (*domain.Category, error) {
	var c domain.Category

	err := r.pool.QueryRow(ctx, query, args...).Scan(
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
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan category: %w", err)
	}

	return &c, nil
}

// scanCategoryRow scans a single row from a rows iterator into a Category struct.
func scanCategoryRow(rows pgx.Rows, c *domain.Category) error {
	return rows.Scan(
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
}
