package service

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
	"github.com/utafrali/EcommerceGo/services/product/internal/event"
	"github.com/utafrali/EcommerceGo/services/product/internal/repository"
)

// --- Mock Repository ---

type mockProductRepository struct {
	mock.Mock
}

func (m *mockProductRepository) Create(ctx context.Context, product *domain.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *mockProductRepository) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Product), args.Error(1)
}

func (m *mockProductRepository) GetBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Product), args.Error(1)
}

func (m *mockProductRepository) List(ctx context.Context, filter repository.ProductFilter) ([]domain.Product, int, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]domain.Product), args.Int(1), args.Error(2)
}

func (m *mockProductRepository) Update(ctx context.Context, product *domain.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *mockProductRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// --- Test Helpers ---

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestService(repo *mockProductRepository) *ProductService {
	logger := newTestLogger()
	// Create a Kafka producer that will fail silently in tests (no real broker).
	kafkaCfg := pkgkafka.DefaultProducerConfig([]string{"localhost:9092"})
	kafkaProducer := pkgkafka.NewProducer(kafkaCfg, logger)
	producer := event.NewProducer(kafkaProducer, logger)
	return NewProductService(repo, producer, logger)
}

func strPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}

// --- Tests ---

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "Widget Pro",
			expected: "widget-pro",
		},
		{
			name:     "name with special characters",
			input:    "Super Widget (2024 Edition)",
			expected: "super-widget-2024-edition",
		},
		{
			name:     "name with extra spaces",
			input:    "  Widget   Pro  ",
			expected: "widget-pro",
		},
		{
			name:     "name with unicode",
			input:    "Caf\u00e9 Latte Machine",
			expected: "caf-latte-machine",
		},
		{
			name:     "already lowercase",
			input:    "widget-pro",
			expected: "widget-pro",
		},
		{
			name:     "single word",
			input:    "Widget",
			expected: "widget",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlug(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateProduct_Success(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Product")).Return(nil)

	input := CreateProductInput{
		Name:        "Test Product",
		Description: "A great product",
		BasePrice:   1999,
		Currency:    "USD",
		Metadata:    map[string]any{"color": "red"},
	}

	product, err := svc.CreateProduct(ctx, input)

	require.NoError(t, err)
	assert.NotEmpty(t, product.ID)
	assert.Equal(t, "Test Product", product.Name)
	assert.Equal(t, "test-product", product.Slug)
	assert.Equal(t, "A great product", product.Description)
	assert.Equal(t, int64(1999), product.BasePrice)
	assert.Equal(t, "USD", product.Currency)
	assert.Equal(t, domain.ProductStatusDraft, product.Status)
	assert.NotZero(t, product.CreatedAt)
	assert.NotZero(t, product.UpdatedAt)
	assert.Equal(t, map[string]any{"color": "red"}, product.Metadata)

	repo.AssertExpectations(t)
}

func TestCreateProduct_EmptyName(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := CreateProductInput{
		Name:      "",
		BasePrice: 1999,
		Currency:  "USD",
	}

	product, err := svc.CreateProduct(ctx, input)

	assert.Nil(t, product)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCreateProduct_NegativePrice(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := CreateProductInput{
		Name:      "Test",
		BasePrice: -100,
		Currency:  "USD",
	}

	product, err := svc.CreateProduct(ctx, input)

	assert.Nil(t, product)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCreateProduct_InvalidCurrency(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := CreateProductInput{
		Name:      "Test",
		BasePrice: 1000,
		Currency:  "US",
	}

	product, err := svc.CreateProduct(ctx, input)

	assert.Nil(t, product)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCreateProduct_RepositoryError(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Product")).
		Return(apperrors.AlreadyExists("product", "slug", "test-product"))

	input := CreateProductInput{
		Name:      "Test Product",
		BasePrice: 1000,
		Currency:  "USD",
	}

	product, err := svc.CreateProduct(ctx, input)

	assert.Nil(t, product)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrAlreadyExists)

	repo.AssertExpectations(t)
}

func TestGetProduct_Success(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	expected := &domain.Product{
		ID:   "abc-123",
		Name: "Test Product",
		Slug: "test-product",
	}

	repo.On("GetByID", ctx, "abc-123").Return(expected, nil)

	product, err := svc.GetProduct(ctx, "abc-123")

	require.NoError(t, err)
	assert.Equal(t, expected, product)

	repo.AssertExpectations(t)
}

func TestGetProduct_NotFound(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	product, err := svc.GetProduct(ctx, "nonexistent")

	assert.Nil(t, product)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	repo.AssertExpectations(t)
}

func TestGetProductBySlug_Success(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	expected := &domain.Product{
		ID:   "abc-123",
		Name: "Test Product",
		Slug: "test-product",
	}

	repo.On("GetBySlug", ctx, "test-product").Return(expected, nil)

	product, err := svc.GetProductBySlug(ctx, "test-product")

	require.NoError(t, err)
	assert.Equal(t, expected, product)

	repo.AssertExpectations(t)
}

func TestListProducts_Success(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	expectedProducts := []domain.Product{
		{ID: "1", Name: "Product A"},
		{ID: "2", Name: "Product B"},
	}

	filter := repository.ProductFilter{
		Page:    1,
		PerPage: 20,
	}

	repo.On("List", ctx, filter).Return(expectedProducts, 2, nil)

	products, total, err := svc.ListProducts(ctx, filter)

	require.NoError(t, err)
	assert.Len(t, products, 2)
	assert.Equal(t, 2, total)

	repo.AssertExpectations(t)
}

func TestListProducts_DefaultPagination(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	// The service should clamp these to defaults.
	filter := repository.ProductFilter{
		Page:    0,
		PerPage: 0,
	}

	expectedFilter := repository.ProductFilter{
		Page:    1,
		PerPage: 20,
	}

	repo.On("List", ctx, expectedFilter).Return([]domain.Product{}, 0, nil)

	products, total, err := svc.ListProducts(ctx, filter)

	require.NoError(t, err)
	assert.Empty(t, products)
	assert.Equal(t, 0, total)

	repo.AssertExpectations(t)
}

func TestListProducts_CapPerPage(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	filter := repository.ProductFilter{
		Page:    1,
		PerPage: 500,
	}

	expectedFilter := repository.ProductFilter{
		Page:    1,
		PerPage: 100,
	}

	repo.On("List", ctx, expectedFilter).Return([]domain.Product{}, 0, nil)

	products, total, err := svc.ListProducts(ctx, filter)

	require.NoError(t, err)
	assert.Empty(t, products)
	assert.Equal(t, 0, total)

	repo.AssertExpectations(t)
}

func TestUpdateProduct_Success(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := &domain.Product{
		ID:        "abc-123",
		Name:      "Old Name",
		Slug:      "old-name",
		Status:    domain.ProductStatusDraft,
		BasePrice: 1000,
		Currency:  "USD",
	}

	repo.On("GetByID", ctx, "abc-123").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.Product")).Return(nil)

	input := UpdateProductInput{
		Name:      strPtr("New Name"),
		BasePrice: int64Ptr(2000),
		Status:    strPtr(domain.ProductStatusPublished),
	}

	product, err := svc.UpdateProduct(ctx, "abc-123", input)

	require.NoError(t, err)
	assert.Equal(t, "New Name", product.Name)
	assert.Equal(t, "new-name", product.Slug)
	assert.Equal(t, int64(2000), product.BasePrice)
	assert.Equal(t, domain.ProductStatusPublished, product.Status)

	repo.AssertExpectations(t)
}

func TestUpdateProduct_NotFound(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	input := UpdateProductInput{
		Name: strPtr("New Name"),
	}

	product, err := svc.UpdateProduct(ctx, "nonexistent", input)

	assert.Nil(t, product)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	repo.AssertExpectations(t)
}

func TestUpdateProduct_InvalidStatus(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := &domain.Product{
		ID:     "abc-123",
		Name:   "Test",
		Slug:   "test",
		Status: domain.ProductStatusDraft,
	}

	repo.On("GetByID", ctx, "abc-123").Return(existing, nil)

	input := UpdateProductInput{
		Status: strPtr("invalid_status"),
	}

	product, err := svc.UpdateProduct(ctx, "abc-123", input)

	assert.Nil(t, product)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestUpdateProduct_EmptyName(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := &domain.Product{
		ID:   "abc-123",
		Name: "Test",
		Slug: "test",
	}

	repo.On("GetByID", ctx, "abc-123").Return(existing, nil)

	emptyName := ""
	input := UpdateProductInput{
		Name: &emptyName,
	}

	product, err := svc.UpdateProduct(ctx, "abc-123", input)

	assert.Nil(t, product)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestUpdateProduct_NegativePrice(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := &domain.Product{
		ID:        "abc-123",
		Name:      "Test",
		Slug:      "test",
		BasePrice: 1000,
	}

	repo.On("GetByID", ctx, "abc-123").Return(existing, nil)

	input := UpdateProductInput{
		BasePrice: int64Ptr(-500),
	}

	product, err := svc.UpdateProduct(ctx, "abc-123", input)

	assert.Nil(t, product)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestDeleteProduct_Success(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := &domain.Product{
		ID:   "abc-123",
		Name: "Test",
		Slug: "test",
	}

	repo.On("GetByID", ctx, "abc-123").Return(existing, nil)
	repo.On("Delete", ctx, "abc-123").Return(nil)

	err := svc.DeleteProduct(ctx, "abc-123")

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeleteProduct_NotFound(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	err := svc.DeleteProduct(ctx, "nonexistent")

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	repo.AssertExpectations(t)
}

func TestCreateProduct_NilMetadata(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Product")).Return(nil)

	input := CreateProductInput{
		Name:      "Test Product",
		BasePrice: 1999,
		Currency:  "USD",
		Metadata:  nil,
	}

	product, err := svc.CreateProduct(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, product.Metadata)
	assert.Empty(t, product.Metadata)

	repo.AssertExpectations(t)
}

func TestCreateProduct_CurrencyUppercased(t *testing.T) {
	repo := new(mockProductRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Product")).Return(nil)

	input := CreateProductInput{
		Name:      "Test Product",
		BasePrice: 1999,
		Currency:  "usd",
	}

	product, err := svc.CreateProduct(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, "USD", product.Currency)

	repo.AssertExpectations(t)
}
