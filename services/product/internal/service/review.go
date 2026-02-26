package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
	"github.com/utafrali/EcommerceGo/services/product/internal/repository/postgres"
)

// CreateReviewInput holds the parameters for creating a review.
type CreateReviewInput struct {
	ProductID string
	UserID    string
	Title     string
	Body      string
	Rating    int
}

// ReviewListResult contains reviews and their aggregate summary.
type ReviewListResult struct {
	Reviews    []domain.Review  `json:"reviews"`
	Summary    *domain.ReviewSummary `json:"summary"`
	TotalCount int              `json:"total_count"`
	Page       int              `json:"page"`
	PerPage    int              `json:"per_page"`
	TotalPages int              `json:"total_pages"`
}

// ReviewService implements the business logic for review operations.
type ReviewService struct {
	repo   *postgres.ReviewRepository
	logger *slog.Logger
}

// NewReviewService creates a new review service.
func NewReviewService(repo *postgres.ReviewRepository, logger *slog.Logger) *ReviewService {
	return &ReviewService{
		repo:   repo,
		logger: logger,
	}
}

// CreateReview creates a new product review with the given input.
func (s *ReviewService) CreateReview(ctx context.Context, input *CreateReviewInput) (*domain.Review, error) {
	if input.ProductID == "" {
		return nil, apperrors.InvalidInput("product_id is required")
	}
	if input.UserID == "" {
		return nil, apperrors.InvalidInput("user_id is required")
	}
	if input.Rating < 1 || input.Rating > 5 {
		return nil, apperrors.InvalidInput("rating must be between 1 and 5")
	}

	now := time.Now().UTC()
	review := &domain.Review{
		ID:        uuid.New().String(),
		ProductID: input.ProductID,
		UserID:    input.UserID,
		Rating:    input.Rating,
		Title:     input.Title,
		Body:      input.Body,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, review); err != nil {
		return nil, fmt.Errorf("create review: %w", err)
	}

	s.logger.InfoContext(ctx, "review created",
		slog.String("review_id", review.ID),
		slog.String("product_id", review.ProductID),
		slog.String("user_id", review.UserID),
		slog.Int("rating", review.Rating),
	)

	return review, nil
}

// ListReviews returns paginated reviews for a product along with the aggregate summary.
func (s *ReviewService) ListReviews(ctx context.Context, productID string, page, perPage int) (*ReviewListResult, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	reviews, total, err := s.repo.ListByProductID(ctx, productID, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("list reviews: %w", err)
	}

	summary, err := s.repo.GetSummary(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("get review summary: %w", err)
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	return &ReviewListResult{
		Reviews:    reviews,
		Summary:    summary,
		TotalCount: total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}
