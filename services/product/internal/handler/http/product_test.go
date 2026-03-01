package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/pkg/httputil"
	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
	"github.com/utafrali/EcommerceGo/services/product/internal/event"
	"github.com/utafrali/EcommerceGo/services/product/internal/repository"
	"github.com/utafrali/EcommerceGo/services/product/internal/service"
)

// =============================================================================
// Mock ProductRepository
// =============================================================================

type mockProductRepo struct {
	mock.Mock
}

func (m *mockProductRepo) Create(ctx context.Context, product *domain.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *mockProductRepo) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Product), args.Error(1)
}

func (m *mockProductRepo) GetBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Product), args.Error(1)
}

func (m *mockProductRepo) List(ctx context.Context, filter repository.ProductFilter) ([]domain.Product, int, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]domain.Product), args.Int(1), args.Error(2)
}

func (m *mockProductRepo) Update(ctx context.Context, product *domain.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *mockProductRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockProductRepo) GetImages(ctx context.Context, productID string) ([]domain.ProductImage, error) {
	args := m.Called(ctx, productID)
	return args.Get(0).([]domain.ProductImage), args.Error(1)
}

func (m *mockProductRepo) GetVariants(ctx context.Context, productID string) ([]domain.ProductVariant, error) {
	args := m.Called(ctx, productID)
	return args.Get(0).([]domain.ProductVariant), args.Error(1)
}

func (m *mockProductRepo) GetCategory(ctx context.Context, categoryID string) (*domain.Category, error) {
	args := m.Called(ctx, categoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Category), args.Error(1)
}

func (m *mockProductRepo) GetBrand(ctx context.Context, brandID string) (*domain.Brand, error) {
	args := m.Called(ctx, brandID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Brand), args.Error(1)
}

func (m *mockProductRepo) GetPrimaryImages(ctx context.Context, productIDs []string) (map[string]domain.ProductImage, error) {
	args := m.Called(ctx, productIDs)
	return args.Get(0).(map[string]domain.ProductImage), args.Error(1)
}

// =============================================================================
// Test helpers
// =============================================================================

func productTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func productTestEventProducer() *event.Producer {
	logger := productTestLogger()
	kafkaCfg := pkgkafka.DefaultProducerConfig([]string{"localhost:9092"})
	kafkaProducer := pkgkafka.NewProducer(kafkaCfg, logger)
	return event.NewProducer(kafkaProducer, logger)
}

func productTestService(repo *mockProductRepo) *service.ProductService {
	logger := productTestLogger()
	producer := productTestEventProducer()
	return service.NewProductService(repo, producer, logger)
}

func productTestHandler(repo *mockProductRepo) *ProductHandler {
	svc := productTestService(repo)
	return NewProductHandler(svc, productTestLogger())
}

func productRouter(handler *ProductHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1/products", func(r chi.Router) {
		r.Get("/", handler.ListProducts)
		r.Get("/{idOrSlug}", handler.GetProduct)
		r.Post("/", handler.CreateProduct)
		r.Put("/{id}", handler.UpdateProduct)
		r.Delete("/{id}", handler.DeleteProduct)
	})
	return r
}

func decodeProductResponse(t *testing.T, rec *httptest.ResponseRecorder) httputil.Response {
	t.Helper()
	var resp httputil.Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	return resp
}

func sampleProduct() *domain.Product {
	now := time.Now().UTC()
	return &domain.Product{
		ID:          "550e8400-e29b-41d4-a716-446655440001",
		Name:        "Test Product",
		Slug:        "test-product",
		Description: "A test product",
		Status:      domain.ProductStatusDraft,
		BasePrice:   1999,
		Currency:    "USD",
		Metadata:    map[string]any{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func sampleProductDetail() *domain.ProductDetail {
	p := sampleProduct()
	return &domain.ProductDetail{
		Product:  *p,
		Images:   []domain.ProductImage{},
		Variants: []domain.ProductVariant{},
	}
}

// =============================================================================
// POST /api/v1/products - CreateProduct
// =============================================================================

func TestCreateProduct_Success(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Product")).Return(nil)

	body := CreateProductRequest{
		Name:      "New Product",
		BasePrice: 2999,
		Currency:  "USD",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	resp := decodeProductResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestCreateProduct_InvalidJSON(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeProductResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestCreateProduct_ValidationError(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	// Missing required fields: name, currency
	body := CreateProductRequest{
		BasePrice: 2999,
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeProductResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestCreateProduct_ServiceError(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Product")).
		Return(apperrors.Internal(nil))

	body := CreateProductRequest{
		Name:      "New Product",
		BasePrice: 2999,
		Currency:  "USD",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	resp := decodeProductResponse(t, rec)
	require.NotNil(t, resp.Error)
	repo.AssertExpectations(t)
}

// =============================================================================
// GET /api/v1/products - ListProducts
// =============================================================================

func TestListProducts_Success(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	products := []domain.Product{*sampleProduct()}
	repo.On("List", mock.Anything, mock.AnythingOfType("repository.ProductFilter")).
		Return(products, 1, nil)
	repo.On("GetPrimaryImages", mock.Anything, mock.Anything).
		Return(map[string]domain.ProductImage{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products?page=1&per_page=10", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var paginatedResp httputil.PaginatedResponse[json.RawMessage]
	err := json.NewDecoder(rec.Body).Decode(&paginatedResp)
	require.NoError(t, err)
	assert.Equal(t, 1, paginatedResp.TotalCount)
	assert.Equal(t, 1, paginatedResp.Page)
	assert.Equal(t, 10, paginatedResp.PerPage)
	assert.Len(t, paginatedResp.Data, 1)
	repo.AssertExpectations(t)
}

func TestListProducts_DefaultPagination(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	repo.On("List", mock.Anything, mock.MatchedBy(func(f repository.ProductFilter) bool {
		return f.Page == 1 && f.PerPage == 20
	})).Return([]domain.Product{}, 0, nil)
	repo.On("GetPrimaryImages", mock.Anything, mock.Anything).
		Return(map[string]domain.ProductImage{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	repo.AssertExpectations(t)
}

func TestListProducts_InvalidPage(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products?page=abc", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeProductResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

func TestListProducts_InvalidPerPage(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products?per_page=999", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeProductResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

func TestListProducts_InvalidStatus(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products?status=unknown", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeProductResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

func TestListProducts_InvalidSortBy(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products?sort_by=invalid", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeProductResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

func TestListProducts_MinPriceGreaterThanMaxPrice(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products?min_price=5000&max_price=1000", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeProductResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "min_price must not exceed max_price")
}

func TestListProducts_ServiceError(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	repo.On("List", mock.Anything, mock.Anything).
		Return([]domain.Product(nil), 0, apperrors.Internal(nil))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	repo.AssertExpectations(t)
}

// =============================================================================
// GET /api/v1/products/{idOrSlug} - GetProduct
// =============================================================================

func TestGetProduct_ByUUID_Success(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	p := sampleProduct()
	repo.On("GetByID", mock.Anything, p.ID).Return(p, nil)
	repo.On("GetImages", mock.Anything, p.ID).Return([]domain.ProductImage{}, nil)
	repo.On("GetVariants", mock.Anything, p.ID).Return([]domain.ProductVariant{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/"+p.ID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeProductResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestGetProduct_BySlug_Success(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	p := sampleProduct()
	repo.On("GetBySlug", mock.Anything, "test-product").Return(p, nil)
	repo.On("GetImages", mock.Anything, p.ID).Return([]domain.ProductImage{}, nil)
	repo.On("GetVariants", mock.Anything, p.ID).Return([]domain.ProductVariant{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/test-product", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeProductResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestGetProduct_NotFound(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	repo.On("GetByID", mock.Anything, "550e8400-e29b-41d4-a716-446655440099").
		Return(nil, apperrors.NotFound("product", "550e8400-e29b-41d4-a716-446655440099"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/550e8400-e29b-41d4-a716-446655440099", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeProductResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	repo.AssertExpectations(t)
}

func TestGetProduct_ServiceError(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	repo.On("GetBySlug", mock.Anything, "some-slug").
		Return(nil, apperrors.Internal(nil))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/some-slug", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	resp := decodeProductResponse(t, rec)
	require.NotNil(t, resp.Error)
	repo.AssertExpectations(t)
}

// =============================================================================
// PUT /api/v1/products/{id} - UpdateProduct
// =============================================================================

func TestUpdateProduct_Success(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	p := sampleProduct()
	repo.On("GetByID", mock.Anything, p.ID).Return(p, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Product")).Return(nil)

	newName := "Updated Product"
	body := UpdateProductRequest{
		Name: &newName,
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/products/"+p.ID, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeProductResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestUpdateProduct_InvalidUUID(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	body := UpdateProductRequest{}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/products/not-a-uuid", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeProductResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid UUID")
}

func TestUpdateProduct_InvalidJSON(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	productID := "550e8400-e29b-41d4-a716-446655440001"

	req := httptest.NewRequest(http.MethodPut, "/api/v1/products/"+productID, bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeProductResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestUpdateProduct_NotFound(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	productID := "550e8400-e29b-41d4-a716-446655440099"
	repo.On("GetByID", mock.Anything, productID).
		Return(nil, apperrors.NotFound("product", productID))

	newName := "Updated"
	body := UpdateProductRequest{Name: &newName}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/products/"+productID, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeProductResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	repo.AssertExpectations(t)
}

func TestUpdateProduct_ValidationError(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	productID := "550e8400-e29b-41d4-a716-446655440001"

	// Invalid status value
	badStatus := "invalid_status"
	body := UpdateProductRequest{Status: &badStatus}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/products/"+productID, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeProductResponse(t, rec)
	require.NotNil(t, resp.Error)
}

// =============================================================================
// DELETE /api/v1/products/{id} - DeleteProduct
// =============================================================================

func TestDeleteProduct_Success(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	productID := "550e8400-e29b-41d4-a716-446655440001"
	p := sampleProduct()
	repo.On("GetByID", mock.Anything, productID).Return(p, nil)
	repo.On("Delete", mock.Anything, productID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/products/"+productID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeProductResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestDeleteProduct_InvalidUUID(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/products/not-a-uuid", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeProductResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid UUID")
}

func TestDeleteProduct_NotFound(t *testing.T) {
	repo := new(mockProductRepo)
	handler := productTestHandler(repo)
	router := productRouter(handler)

	productID := "550e8400-e29b-41d4-a716-446655440099"
	repo.On("GetByID", mock.Anything, productID).
		Return(nil, apperrors.NotFound("product", productID))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/products/"+productID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeProductResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	repo.AssertExpectations(t)
}

// =============================================================================
// Table-driven: CreateProduct edge cases
// =============================================================================

func TestCreateProduct_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectStatus   int
		expectErrCode  string
	}{
		{
			name:          "empty body",
			body:          `{}`,
			expectStatus:  http.StatusBadRequest,
			expectErrCode: "VALIDATION_ERROR",
		},
		{
			name:          "missing name",
			body:          `{"base_price": 100, "currency": "USD"}`,
			expectStatus:  http.StatusBadRequest,
			expectErrCode: "VALIDATION_ERROR",
		},
		{
			name:          "missing currency",
			body:          `{"name": "Test", "base_price": 100}`,
			expectStatus:  http.StatusBadRequest,
			expectErrCode: "VALIDATION_ERROR",
		},
		{
			name:          "currency too short",
			body:          `{"name": "Test", "base_price": 100, "currency": "US"}`,
			expectStatus:  http.StatusBadRequest,
			expectErrCode: "VALIDATION_ERROR",
		},
		{
			name:          "invalid brand_id uuid",
			body:          `{"name": "Test", "base_price": 100, "currency": "USD", "brand_id": "not-uuid"}`,
			expectStatus:  http.StatusBadRequest,
			expectErrCode: "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockProductRepo)
			handler := productTestHandler(repo)
			router := productRouter(handler)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectStatus, rec.Code)
			resp := decodeProductResponse(t, rec)
			require.NotNil(t, resp.Error)
			assert.Equal(t, tt.expectErrCode, resp.Error.Code)
		})
	}
}

// =============================================================================
// Table-driven: ListProducts filter parameters
// =============================================================================

func TestListProducts_FilterParams_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		expectStatus int
		expectErr    bool
		errCode      string
	}{
		{
			name:         "valid page",
			query:        "?page=2",
			expectStatus: http.StatusOK,
		},
		{
			name:         "page zero",
			query:        "?page=0",
			expectStatus: http.StatusBadRequest,
			expectErr:    true,
			errCode:      "INVALID_PARAMETER",
		},
		{
			name:         "negative page",
			query:        "?page=-1",
			expectStatus: http.StatusBadRequest,
			expectErr:    true,
			errCode:      "INVALID_PARAMETER",
		},
		{
			name:         "per_page zero",
			query:        "?per_page=0",
			expectStatus: http.StatusBadRequest,
			expectErr:    true,
			errCode:      "INVALID_PARAMETER",
		},
		{
			name:         "per_page over 100",
			query:        "?per_page=101",
			expectStatus: http.StatusBadRequest,
			expectErr:    true,
			errCode:      "INVALID_PARAMETER",
		},
		{
			name:         "invalid min_price",
			query:        "?min_price=abc",
			expectStatus: http.StatusBadRequest,
			expectErr:    true,
			errCode:      "INVALID_PARAMETER",
		},
		{
			name:         "invalid max_price",
			query:        "?max_price=abc",
			expectStatus: http.StatusBadRequest,
			expectErr:    true,
			errCode:      "INVALID_PARAMETER",
		},
		{
			name:         "valid status filter",
			query:        "?status=published",
			expectStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockProductRepo)
			handler := productTestHandler(repo)
			router := productRouter(handler)

			if !tt.expectErr {
				repo.On("List", mock.Anything, mock.Anything).
					Return([]domain.Product{}, 0, nil)
				repo.On("GetPrimaryImages", mock.Anything, mock.Anything).
					Return(map[string]domain.ProductImage{}, nil)
			}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/products"+tt.query, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectStatus, rec.Code)

			if tt.expectErr {
				resp := decodeProductResponse(t, rec)
				require.NotNil(t, resp.Error)
				assert.Equal(t, tt.errCode, resp.Error.Code)
			}
		})
	}
}
