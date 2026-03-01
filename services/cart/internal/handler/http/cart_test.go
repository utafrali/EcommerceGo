package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/utafrali/EcommerceGo/services/cart/internal/domain"
	"github.com/utafrali/EcommerceGo/services/cart/internal/event"
	"github.com/utafrali/EcommerceGo/services/cart/internal/service"
)

// ============================================================================
// Mock CartRepository
// ============================================================================

type mockCartRepository struct {
	mock.Mock
}

func (m *mockCartRepository) Get(ctx context.Context, userID string) (*domain.Cart, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Cart), args.Error(1)
}

func (m *mockCartRepository) Save(ctx context.Context, cart *domain.Cart) error {
	args := m.Called(ctx, cart)
	return args.Error(0)
}

func (m *mockCartRepository) SaveIfVersion(ctx context.Context, cart *domain.Cart, expectedVersion int) (bool, error) {
	args := m.Called(ctx, cart, expectedVersion)
	return args.Bool(0), args.Error(1)
}

func (m *mockCartRepository) Delete(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// ============================================================================
// Test helpers
// ============================================================================

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func testEventProducer() *event.Producer {
	logger := testLogger()
	kafkaCfg := pkgkafka.DefaultProducerConfig([]string{"localhost:19092"})
	kafkaProducer := pkgkafka.NewProducer(kafkaCfg, logger)
	return event.NewProducer(kafkaProducer, logger)
}

func testCartService(repo *mockCartRepository) *service.CartService {
	logger := testLogger()
	producer := testEventProducer()
	return service.NewCartService(repo, producer, logger, 24*time.Hour)
}

func testCartHandler(repo *mockCartRepository) *CartHandler {
	svc := testCartService(repo)
	return NewCartHandler(svc, testLogger())
}

// setupCartRouter creates a chi router matching the production route layout
// for the cart service, including the UserIDFromHeader and ContentTypeJSON
// middleware so that auth behavior is tested end-to-end.
func setupCartRouter(handler *CartHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1/cart", func(r chi.Router) {
		r.Use(ContentTypeJSON)
		r.Use(UserIDFromHeader)

		r.Get("/", handler.GetCart)
		r.Delete("/", handler.ClearCart)

		r.Post("/items", handler.AddItem)
		r.Put("/items/{productId}/{variantId}", handler.UpdateItemQuantity)
		r.Delete("/items/{productId}/{variantId}", handler.RemoveItem)
	})
	return r
}

// decodeResponse reads the response body into the standard Response struct.
func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) httputil.Response {
	t.Helper()
	var resp httputil.Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	return resp
}

// sampleCart returns a cart with one item, suitable for test assertions.
func sampleCart() *domain.Cart {
	now := time.Now().UTC()
	return &domain.Cart{
		ID:     "cart-001",
		UserID: "user-123",
		Items: []domain.CartItem{
			{
				ProductID: "550e8400-e29b-41d4-a716-446655440001",
				VariantID: "550e8400-e29b-41d4-a716-446655440002",
				Name:      "Test Widget",
				SKU:       "WDG-001",
				Price:     1999,
				Quantity:  2,
				ImageURL:  "https://img.example.com/widget.jpg",
			},
		},
		Currency:  "USD",
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
}

// Valid UUIDs for URL params.
const (
	validProductID = "550e8400-e29b-41d4-a716-446655440001"
	validVariantID = "550e8400-e29b-41d4-a716-446655440002"
)

// ============================================================================
// GET /api/v1/cart - GetCart
// ============================================================================

func TestGetCart_Success(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	cart := sampleCart()
	repo.On("Get", mock.Anything, "user-123").Return(cart, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestGetCart_EmptyCart_NotFound(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	// When the repository returns ErrNotFound, the service creates an empty cart.
	repo.On("Get", mock.Anything, "user-123").Return(nil, apperrors.NotFound("cart", "user-123"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestGetCart_MissingUserID_Returns401(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
	// No X-User-ID header.
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "UNAUTHORIZED", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "authentication required")
}

func TestGetCart_ServiceError(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	repo.On("Get", mock.Anything, "user-123").Return(nil, fmt.Errorf("redis connection refused"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	repo.AssertExpectations(t)
}

// ============================================================================
// POST /api/v1/cart/items - AddItem
// ============================================================================

func validAddItemJSON() []byte {
	body := AddItemRequest{
		ProductID: validProductID,
		VariantID: validVariantID,
		Name:      "Test Widget",
		SKU:       "WDG-001",
		Price:     1999,
		Quantity:  2,
		ImageURL:  "https://img.example.com/widget.jpg",
	}
	b, _ := json.Marshal(body)
	return b
}

func TestAddItem_Success(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	// GetOrCreateCart: returns not found so service creates an empty cart.
	repo.On("Get", mock.Anything, "user-123").Return(nil, apperrors.NotFound("cart", "user-123"))
	// SaveIfVersion succeeds (no conflict).
	repo.On("SaveIfVersion", mock.Anything, mock.AnythingOfType("*domain.Cart"), 0).Return(true, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/items", bytes.NewReader(validAddItemJSON()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestAddItem_MissingUserID_Returns401(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/items", bytes.NewReader(validAddItemJSON()))
	req.Header.Set("Content-Type", "application/json")
	// No X-User-ID header.
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "UNAUTHORIZED", resp.Error.Code)
}

func TestAddItem_InvalidJSON(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/items", bytes.NewReader([]byte(`{invalid json`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestAddItem_ValidationError_MissingFields(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	// Send a request body with missing required fields.
	body := map[string]interface{}{
		"product_id": "", // required
		"variant_id": "", // required
		"name":       "", // required
		"sku":        "", // required
		"price":      0,  // required
		"quantity":   0,  // required gte=1
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/items", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestAddItem_ServiceError(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	// Repository returns an internal error.
	repo.On("Get", mock.Anything, "user-123").Return(nil, fmt.Errorf("redis timeout"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/items", bytes.NewReader(validAddItemJSON()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	repo.AssertExpectations(t)
}

func TestAddItem_VersionConflict_Returns409(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	cart := sampleCart()
	repo.On("Get", mock.Anything, "user-123").Return(cart, nil)
	// SaveIfVersion returns false (version conflict).
	repo.On("SaveIfVersion", mock.Anything, mock.AnythingOfType("*domain.Cart"), 1).Return(false, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/items", bytes.NewReader(validAddItemJSON()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "CONFLICT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "modified concurrently")
	repo.AssertExpectations(t)
}

// ============================================================================
// PUT /api/v1/cart/items/{productId}/{variantId} - UpdateItemQuantity
// ============================================================================

func validUpdateQuantityJSON(qty int) []byte {
	body := UpdateQuantityRequest{Quantity: qty}
	b, _ := json.Marshal(body)
	return b
}

func TestUpdateItemQuantity_Success(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	cart := sampleCart()
	repo.On("Get", mock.Anything, "user-123").Return(cart, nil)
	repo.On("SaveIfVersion", mock.Anything, mock.AnythingOfType("*domain.Cart"), 1).Return(true, nil)

	url := fmt.Sprintf("/api/v1/cart/items/%s/%s", validProductID, validVariantID)
	req := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(validUpdateQuantityJSON(5)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestUpdateItemQuantity_MissingUserID_Returns401(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	url := fmt.Sprintf("/api/v1/cart/items/%s/%s", validProductID, validVariantID)
	req := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(validUpdateQuantityJSON(5)))
	req.Header.Set("Content-Type", "application/json")
	// No X-User-ID header.
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "UNAUTHORIZED", resp.Error.Code)
}

func TestUpdateItemQuantity_InvalidProductID(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	url := fmt.Sprintf("/api/v1/cart/items/%s/%s", "not-a-uuid", validVariantID)
	req := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(validUpdateQuantityJSON(5)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "not-a-uuid")
}

func TestUpdateItemQuantity_InvalidVariantID(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	url := fmt.Sprintf("/api/v1/cart/items/%s/%s", validProductID, "bad-variant")
	req := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(validUpdateQuantityJSON(5)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "bad-variant")
}

func TestUpdateItemQuantity_InvalidJSON(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	url := fmt.Sprintf("/api/v1/cart/items/%s/%s", validProductID, validVariantID)
	req := httptest.NewRequest(http.MethodPut, url, bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
}

func TestUpdateItemQuantity_VersionConflict_Returns409(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	cart := sampleCart()
	repo.On("Get", mock.Anything, "user-123").Return(cart, nil)
	// SaveIfVersion returns false (concurrent modification).
	repo.On("SaveIfVersion", mock.Anything, mock.AnythingOfType("*domain.Cart"), 1).Return(false, nil)

	url := fmt.Sprintf("/api/v1/cart/items/%s/%s", validProductID, validVariantID)
	req := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(validUpdateQuantityJSON(3)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "CONFLICT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "modified concurrently")
	repo.AssertExpectations(t)
}

func TestUpdateItemQuantity_NegativeQuantity_ValidationError(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	// Negative quantity (-1) is rejected by the handler-level struct validator
	// (validate:"gte=0") before the service layer is ever called. No repo
	// expectations are needed.
	url := fmt.Sprintf("/api/v1/cart/items/%s/%s", validProductID, validVariantID)
	req := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(validUpdateQuantityJSON(-1)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
	repo.AssertExpectations(t)
}

func TestUpdateItemQuantity_ItemNotFound(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	// Cart with no items.
	cart := sampleCart()
	cart.Items = []domain.CartItem{}
	repo.On("Get", mock.Anything, "user-123").Return(cart, nil)

	url := fmt.Sprintf("/api/v1/cart/items/%s/%s", validProductID, validVariantID)
	req := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(validUpdateQuantityJSON(3)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	repo.AssertExpectations(t)
}

// ============================================================================
// DELETE /api/v1/cart/items/{productId}/{variantId} - RemoveItem
// ============================================================================

func TestRemoveItem_Success(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	cart := sampleCart()
	repo.On("Get", mock.Anything, "user-123").Return(cart, nil)
	repo.On("SaveIfVersion", mock.Anything, mock.AnythingOfType("*domain.Cart"), 1).Return(true, nil)

	url := fmt.Sprintf("/api/v1/cart/items/%s/%s", validProductID, validVariantID)
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestRemoveItem_MissingUserID_Returns401(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	url := fmt.Sprintf("/api/v1/cart/items/%s/%s", validProductID, validVariantID)
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	// No X-User-ID header.
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "UNAUTHORIZED", resp.Error.Code)
}

func TestRemoveItem_InvalidProductUUID(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	url := fmt.Sprintf("/api/v1/cart/items/%s/%s", "xyz", validVariantID)
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "xyz")
}

func TestRemoveItem_InvalidVariantUUID(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	url := fmt.Sprintf("/api/v1/cart/items/%s/%s", validProductID, "bad-id")
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "bad-id")
}

func TestRemoveItem_ItemNotFound(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	// Cart with no items.
	cart := sampleCart()
	cart.Items = []domain.CartItem{}
	repo.On("Get", mock.Anything, "user-123").Return(cart, nil)

	url := fmt.Sprintf("/api/v1/cart/items/%s/%s", validProductID, validVariantID)
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	repo.AssertExpectations(t)
}

// ============================================================================
// DELETE /api/v1/cart - ClearCart
// ============================================================================

func TestClearCart_Success(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	repo.On("Delete", mock.Anything, "user-123").Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart", nil)
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestClearCart_MissingUserID_Returns401(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart", nil)
	// No X-User-ID header.
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "UNAUTHORIZED", resp.Error.Code)
}

func TestClearCart_ServiceError(t *testing.T) {
	repo := new(mockCartRepository)
	handler := testCartHandler(repo)
	router := setupCartRouter(handler)

	repo.On("Delete", mock.Anything, "user-123").Return(fmt.Errorf("redis connection lost"))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart", nil)
	req.Header.Set("X-User-ID", "user-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	repo.AssertExpectations(t)
}

// ============================================================================
// Middleware tests
// ============================================================================

func TestUserIDFromHeader_Middleware_SetsContext(t *testing.T) {
	var capturedUID string
	handler := UserIDFromHeader(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := userIDFromContext(r.Context())
		if ok {
			capturedUID = uid
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", "user-abc")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "user-abc", capturedUID)
}

func TestUserIDFromHeader_Middleware_MissingHeader(t *testing.T) {
	called := false
	handler := UserIDFromHeader(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No X-User-ID header.
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called, "handler should not have been called")
}

func TestUserIDFromContext_EmptyString(t *testing.T) {
	// When context has no user ID, it returns empty string + false.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	uid, ok := userIDFromContext(req.Context())
	assert.False(t, ok)
	assert.Empty(t, uid)
}

func TestContentTypeJSON_Middleware_RejectsNonJSON(t *testing.T) {
	called := false
	handler := ContentTypeJSON(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("data")))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
	assert.False(t, called, "handler should not have been called")
}

func TestContentTypeJSON_Middleware_AcceptsJSON(t *testing.T) {
	called := false
	handler := ContentTypeJSON(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called, "handler should have been called")
}

// ============================================================================
// Table-driven: all endpoints reject missing X-User-ID with 401
// ============================================================================

func TestAllEndpoints_RejectMissingUserID(t *testing.T) {
	endpoints := []struct {
		method string
		path   string
		body   []byte
	}{
		{http.MethodGet, "/api/v1/cart", nil},
		{http.MethodPost, "/api/v1/cart/items", validAddItemJSON()},
		{http.MethodPut, fmt.Sprintf("/api/v1/cart/items/%s/%s", validProductID, validVariantID), validUpdateQuantityJSON(1)},
		{http.MethodDelete, fmt.Sprintf("/api/v1/cart/items/%s/%s", validProductID, validVariantID), nil},
		{http.MethodDelete, "/api/v1/cart", nil},
	}

	for _, ep := range endpoints {
		name := fmt.Sprintf("%s %s", ep.method, ep.path)
		t.Run(name, func(t *testing.T) {
			repo := new(mockCartRepository)
			handler := testCartHandler(repo)
			router := setupCartRouter(handler)

			var body *bytes.Reader
			if ep.body != nil {
				body = bytes.NewReader(ep.body)
			} else {
				body = bytes.NewReader(nil)
			}

			req := httptest.NewRequest(ep.method, ep.path, body)
			if ep.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			// No X-User-ID header.
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code, "expected 401 for missing X-User-ID on %s", name)
			resp := decodeResponse(t, rec)
			require.NotNil(t, resp.Error)
			assert.Equal(t, "UNAUTHORIZED", resp.Error.Code)
		})
	}
}

// ============================================================================
// Optimistic locking: version conflict across multiple mutation endpoints
// ============================================================================

func TestVersionConflict_AcrossEndpoints(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   []byte
		setup  func(repo *mockCartRepository)
	}{
		{
			name:   "AddItem conflict",
			method: http.MethodPost,
			path:   "/api/v1/cart/items",
			body:   validAddItemJSON(),
			setup: func(repo *mockCartRepository) {
				cart := sampleCart()
				repo.On("Get", mock.Anything, "user-123").Return(cart, nil)
				repo.On("SaveIfVersion", mock.Anything, mock.AnythingOfType("*domain.Cart"), 1).Return(false, nil)
			},
		},
		{
			name:   "UpdateItemQuantity conflict",
			method: http.MethodPut,
			path:   fmt.Sprintf("/api/v1/cart/items/%s/%s", validProductID, validVariantID),
			body:   validUpdateQuantityJSON(5),
			setup: func(repo *mockCartRepository) {
				cart := sampleCart()
				repo.On("Get", mock.Anything, "user-123").Return(cart, nil)
				repo.On("SaveIfVersion", mock.Anything, mock.AnythingOfType("*domain.Cart"), 1).Return(false, nil)
			},
		},
		{
			name:   "RemoveItem conflict",
			method: http.MethodDelete,
			path:   fmt.Sprintf("/api/v1/cart/items/%s/%s", validProductID, validVariantID),
			body:   nil,
			setup: func(repo *mockCartRepository) {
				cart := sampleCart()
				repo.On("Get", mock.Anything, "user-123").Return(cart, nil)
				repo.On("SaveIfVersion", mock.Anything, mock.AnythingOfType("*domain.Cart"), 1).Return(false, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockCartRepository)
			handler := testCartHandler(repo)
			router := setupCartRouter(handler)
			tt.setup(repo)

			var body *bytes.Reader
			if tt.body != nil {
				body = bytes.NewReader(tt.body)
			} else {
				body = bytes.NewReader(nil)
			}

			req := httptest.NewRequest(tt.method, tt.path, body)
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			req.Header.Set("X-User-ID", "user-123")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusConflict, rec.Code, "expected 409 for version conflict")
			resp := decodeResponse(t, rec)
			require.NotNil(t, resp.Error)
			assert.Equal(t, "CONFLICT", resp.Error.Code)
			assert.Contains(t, resp.Error.Message, "modified concurrently")
			repo.AssertExpectations(t)
		})
	}
}
