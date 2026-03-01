package postgres

import (
	"context"
	"fmt"
	"math"

	"github.com/utafrali/EcommerceGo/pkg/database"
	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
)

// ReviewRepository implements review persistence operations using PostgreSQL.
type ReviewRepository struct {
	pool database.DBTX
}

// NewReviewRepository creates a new PostgreSQL-backed review repository.
func NewReviewRepository(pool database.DBTX) *ReviewRepository {
	return &ReviewRepository{pool: pool}
}

// Create inserts a new product review into the database.
func (r *ReviewRepository) Create(ctx context.Context, review *domain.Review) error {
	query := `
		INSERT INTO product_reviews (id, product_id, user_id, rating, title, body, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, query,
		review.ID,
		review.ProductID,
		review.UserID,
		review.Rating,
		review.Title,
		review.Body,
		review.CreatedAt,
		review.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert review: %w", err)
	}

	return nil
}

// ListByProductID returns paginated reviews for a given product along with the total count.
func (r *ReviewRepository) ListByProductID(ctx context.Context, productID string, page, perPage int) ([]domain.Review, int, error) {
	limit := perPage
	if limit <= 0 {
		limit = 20
	}
	offset := 0
	if page > 1 {
		offset = (page - 1) * limit
	}

	query := `
		SELECT id, product_id, user_id, rating, title, body, created_at, updated_at,
		       count(*) OVER() AS total_count
		FROM product_reviews
		WHERE product_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, productID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list reviews: %w", err)
	}
	defer rows.Close()

	var (
		reviews    []domain.Review
		totalCount int
	)

	for rows.Next() {
		var rv domain.Review

		if err := rows.Scan(
			&rv.ID,
			&rv.ProductID,
			&rv.UserID,
			&rv.Rating,
			&rv.Title,
			&rv.Body,
			&rv.CreatedAt,
			&rv.UpdatedAt,
			&totalCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan review row: %w", err)
		}

		reviews = append(reviews, rv)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate review rows: %w", err)
	}

	if reviews == nil {
		reviews = []domain.Review{}
	}

	return reviews, totalCount, nil
}

// GetSummary returns the average rating and total count of reviews for a product.
func (r *ReviewRepository) GetSummary(ctx context.Context, productID string) (*domain.ReviewSummary, error) {
	query := `
		SELECT COALESCE(AVG(rating), 0), COUNT(*)
		FROM product_reviews
		WHERE product_id = $1`

	var summary domain.ReviewSummary

	err := r.pool.QueryRow(ctx, query, productID).Scan(
		&summary.AverageRating,
		&summary.TotalCount,
	)
	if err != nil {
		return nil, fmt.Errorf("get review summary: %w", err)
	}

	// Round average rating to one decimal place.
	summary.AverageRating = math.Round(summary.AverageRating*10) / 10

	return &summary, nil
}
