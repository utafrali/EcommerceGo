package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
)

// --- Mock Review Repository ---

type mockReviewRepository struct {
	mock.Mock
}

func (m *mockReviewRepository) Create(ctx context.Context, review *domain.Review) error {
	args := m.Called(ctx, review)
	return args.Error(0)
}

func (m *mockReviewRepository) ListByProductID(ctx context.Context, productID string, page, perPage int) ([]domain.Review, int, error) {
	args := m.Called(ctx, productID, page, perPage)
	return args.Get(0).([]domain.Review), args.Int(1), args.Error(2)
}

func (m *mockReviewRepository) GetSummary(ctx context.Context, productID string) (*domain.ReviewSummary, error) {
	args := m.Called(ctx, productID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ReviewSummary), args.Error(1)
}

// --- Test Helpers ---

func newTestReviewService(repo *mockReviewRepository) *ReviewService {
	logger := newTestLogger()
	return NewReviewService(repo, logger)
}

// --- Tests ---

func TestCreateReview_Success(t *testing.T) {
	repo := new(mockReviewRepository)
	svc := newTestReviewService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Review")).Return(nil)

	input := CreateReviewInput{
		ProductID: "prod-123",
		UserID:    "user-456",
		Title:     "Great product",
		Body:      "I really enjoyed using this product.",
		Rating:    5,
	}

	review, err := svc.CreateReview(ctx, &input)

	require.NoError(t, err)
	assert.NotEmpty(t, review.ID)
	assert.Equal(t, "prod-123", review.ProductID)
	assert.Equal(t, "user-456", review.UserID)
	assert.Equal(t, "Great product", review.Title)
	assert.Equal(t, "I really enjoyed using this product.", review.Body)
	assert.Equal(t, 5, review.Rating)
	assert.NotZero(t, review.CreatedAt)
	assert.NotZero(t, review.UpdatedAt)

	repo.AssertExpectations(t)
}

func TestCreateReview_ValidationError_EmptyProductID(t *testing.T) {
	repo := new(mockReviewRepository)
	svc := newTestReviewService(repo)
	ctx := context.Background()

	input := CreateReviewInput{
		ProductID: "",
		UserID:    "user-456",
		Rating:    3,
	}

	review, err := svc.CreateReview(ctx, &input)

	assert.Nil(t, review)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCreateReview_ValidationError_EmptyUserID(t *testing.T) {
	repo := new(mockReviewRepository)
	svc := newTestReviewService(repo)
	ctx := context.Background()

	input := CreateReviewInput{
		ProductID: "prod-123",
		UserID:    "",
		Rating:    3,
	}

	review, err := svc.CreateReview(ctx, &input)

	assert.Nil(t, review)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCreateReview_ValidationError_RatingTooLow(t *testing.T) {
	repo := new(mockReviewRepository)
	svc := newTestReviewService(repo)
	ctx := context.Background()

	input := CreateReviewInput{
		ProductID: "prod-123",
		UserID:    "user-456",
		Rating:    0,
	}

	review, err := svc.CreateReview(ctx, &input)

	assert.Nil(t, review)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCreateReview_ValidationError_RatingTooHigh(t *testing.T) {
	repo := new(mockReviewRepository)
	svc := newTestReviewService(repo)
	ctx := context.Background()

	input := CreateReviewInput{
		ProductID: "prod-123",
		UserID:    "user-456",
		Rating:    6,
	}

	review, err := svc.CreateReview(ctx, &input)

	assert.Nil(t, review)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCreateReview_RepositoryError(t *testing.T) {
	repo := new(mockReviewRepository)
	svc := newTestReviewService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Review")).
		Return(fmt.Errorf("database connection failed"))

	input := CreateReviewInput{
		ProductID: "prod-123",
		UserID:    "user-456",
		Rating:    4,
		Title:     "Good product",
	}

	review, err := svc.CreateReview(ctx, &input)

	assert.Nil(t, review)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create review")

	repo.AssertExpectations(t)
}

func TestListReviewsByProductID_Success(t *testing.T) {
	repo := new(mockReviewRepository)
	svc := newTestReviewService(repo)
	ctx := context.Background()

	expectedReviews := []domain.Review{
		{ID: "rev-1", ProductID: "prod-123", UserID: "user-1", Rating: 5, Title: "Great"},
		{ID: "rev-2", ProductID: "prod-123", UserID: "user-2", Rating: 4, Title: "Good"},
		{ID: "rev-3", ProductID: "prod-123", UserID: "user-3", Rating: 3, Title: "Okay"},
	}
	expectedSummary := &domain.ReviewSummary{
		AverageRating: 4.0,
		TotalCount:    3,
	}

	repo.On("ListByProductID", ctx, "prod-123", 1, 20).Return(expectedReviews, 3, nil)
	repo.On("GetSummary", ctx, "prod-123").Return(expectedSummary, nil)

	result, err := svc.ListReviews(ctx, "prod-123", 1, 20)

	require.NoError(t, err)
	assert.Len(t, result.Reviews, 3)
	assert.Equal(t, 3, result.TotalCount)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.PerPage)
	assert.Equal(t, 1, result.TotalPages)
	assert.Equal(t, 4.0, result.Summary.AverageRating)
	assert.Equal(t, 3, result.Summary.TotalCount)

	repo.AssertExpectations(t)
}

func TestListReviewsByProductID_DefaultPagination(t *testing.T) {
	repo := new(mockReviewRepository)
	svc := newTestReviewService(repo)
	ctx := context.Background()

	repo.On("ListByProductID", ctx, "prod-123", 1, 20).Return([]domain.Review{}, 0, nil)
	repo.On("GetSummary", ctx, "prod-123").Return(&domain.ReviewSummary{}, nil)

	// Pass zero values; the service should clamp to defaults.
	result, err := svc.ListReviews(ctx, "prod-123", 0, 0)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.PerPage)
	assert.Empty(t, result.Reviews)

	repo.AssertExpectations(t)
}

func TestListReviewsByProductID_CapPerPage(t *testing.T) {
	repo := new(mockReviewRepository)
	svc := newTestReviewService(repo)
	ctx := context.Background()

	repo.On("ListByProductID", ctx, "prod-123", 1, 100).Return([]domain.Review{}, 0, nil)
	repo.On("GetSummary", ctx, "prod-123").Return(&domain.ReviewSummary{}, nil)

	// Pass a perPage > 100; the service should cap it at 100.
	result, err := svc.ListReviews(ctx, "prod-123", 1, 500)

	require.NoError(t, err)
	assert.Equal(t, 100, result.PerPage)

	repo.AssertExpectations(t)
}

func TestListReviewsByProductID_MultiplePages(t *testing.T) {
	repo := new(mockReviewRepository)
	svc := newTestReviewService(repo)
	ctx := context.Background()

	expectedReviews := []domain.Review{
		{ID: "rev-3", ProductID: "prod-123", Rating: 3},
	}
	expectedSummary := &domain.ReviewSummary{
		AverageRating: 4.0,
		TotalCount:    5,
	}

	repo.On("ListByProductID", ctx, "prod-123", 2, 2).Return(expectedReviews, 5, nil)
	repo.On("GetSummary", ctx, "prod-123").Return(expectedSummary, nil)

	result, err := svc.ListReviews(ctx, "prod-123", 2, 2)

	require.NoError(t, err)
	assert.Len(t, result.Reviews, 1)
	assert.Equal(t, 5, result.TotalCount)
	assert.Equal(t, 2, result.Page)
	assert.Equal(t, 2, result.PerPage)
	assert.Equal(t, 3, result.TotalPages) // 5 total / 2 per page = 3 pages

	repo.AssertExpectations(t)
}

func TestListReviewsByProductID_ListError(t *testing.T) {
	repo := new(mockReviewRepository)
	svc := newTestReviewService(repo)
	ctx := context.Background()

	repo.On("ListByProductID", ctx, "prod-123", 1, 20).
		Return([]domain.Review{}, 0, fmt.Errorf("database error"))

	result, err := svc.ListReviews(ctx, "prod-123", 1, 20)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list reviews")

	repo.AssertExpectations(t)
}

func TestGetReviewSummary_Success(t *testing.T) {
	repo := new(mockReviewRepository)
	svc := newTestReviewService(repo)
	ctx := context.Background()

	expectedReviews := []domain.Review{
		{ID: "rev-1", ProductID: "prod-123", Rating: 5},
	}
	expectedSummary := &domain.ReviewSummary{
		AverageRating: 4.5,
		TotalCount:    10,
	}

	repo.On("ListByProductID", ctx, "prod-123", 1, 20).Return(expectedReviews, 1, nil)
	repo.On("GetSummary", ctx, "prod-123").Return(expectedSummary, nil)

	// ListReviews returns the summary as part of the result.
	result, err := svc.ListReviews(ctx, "prod-123", 1, 20)

	require.NoError(t, err)
	require.NotNil(t, result.Summary)
	assert.Equal(t, 4.5, result.Summary.AverageRating)
	assert.Equal(t, 10, result.Summary.TotalCount)

	repo.AssertExpectations(t)
}

func TestGetReviewSummary_Error(t *testing.T) {
	repo := new(mockReviewRepository)
	svc := newTestReviewService(repo)
	ctx := context.Background()

	repo.On("ListByProductID", ctx, "prod-123", 1, 20).Return([]domain.Review{}, 0, nil)
	repo.On("GetSummary", ctx, "prod-123").Return(nil, fmt.Errorf("summary error"))

	result, err := svc.ListReviews(ctx, "prod-123", 1, 20)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get review summary")

	repo.AssertExpectations(t)
}
