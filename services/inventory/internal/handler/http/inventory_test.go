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

	"github.com/utafrali/EcommerceGo/pkg/httputil"
	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/inventory/internal/domain"
	"github.com/utafrali/EcommerceGo/services/inventory/internal/event"
	"github.com/utafrali/EcommerceGo/services/inventory/internal/service"
)

// ============================================================================
// Mock Repositories
// ============================================================================

type mockStockRepository struct {
	mock.Mock
}

func (m *mockStockRepository) GetByProductVariant(ctx context.Context, productID, variantID string) (*domain.Stock, error) {
	args := m.Called(ctx, productID, variantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Stock), args.Error(1)
}

func (m *mockStockRepository) CreateStock(ctx context.Context, stock *domain.Stock) (*domain.Stock, error) {
	args := m.Called(ctx, stock)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Stock), args.Error(1)
}

func (m *mockStockRepository) Upsert(ctx context.Context, stock *domain.Stock) error {
	args := m.Called(ctx, stock)
	return args.Error(0)
}

func (m *mockStockRepository) AdjustQuantity(ctx context.Context, productID, variantID string, delta int, reason string, refID *string) error {
	args := m.Called(ctx, productID, variantID, delta, reason, refID)
	return args.Error(0)
}

func (m *mockStockRepository) ListLowStock(ctx context.Context, page, perPage int) ([]domain.Stock, int, error) {
	args := m.Called(ctx, page, perPage)
	return args.Get(0).([]domain.Stock), args.Int(1), args.Error(2)
}

func (m *mockStockRepository) BulkCheck(ctx context.Context, items []domain.StockCheckItem) ([]domain.StockCheckResult, error) {
	args := m.Called(ctx, items)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.StockCheckResult), args.Error(1)
}

type mockReservationRepository struct {
	mock.Mock
}

func (m *mockReservationRepository) Create(ctx context.Context, reservation *domain.StockReservation) error {
	args := m.Called(ctx, reservation)
	return args.Error(0)
}

func (m *mockReservationRepository) GetByID(ctx context.Context, id string) (*domain.StockReservation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.StockReservation), args.Error(1)
}

func (m *mockReservationRepository) GetByCheckoutID(ctx context.Context, checkoutID string) ([]domain.StockReservation, error) {
	args := m.Called(ctx, checkoutID)
	return args.Get(0).([]domain.StockReservation), args.Error(1)
}

func (m *mockReservationRepository) UpdateStatus(ctx context.Context, id, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *mockReservationRepository) GetExpired(ctx context.Context) ([]domain.StockReservation, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.StockReservation), args.Error(1)
}

// ============================================================================
// Test Helpers
// ============================================================================

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func testEventProducer() *event.Producer {
	logger := testLogger()
	kafkaCfg := pkgkafka.DefaultProducerConfig([]string{"localhost:9092"})
	kafkaProducer := pkgkafka.NewProducer(kafkaCfg, logger)
	return event.NewProducer(kafkaProducer, logger)
}

func testInventoryService(stockRepo *mockStockRepository, reservationRepo *mockReservationRepository) *service.InventoryService {
	logger := testLogger()
	producer := testEventProducer()
	// pool is nil -- only used by ReserveStock/ReleaseReservation/ConfirmReservation
	// which require actual DB transactions. We test those paths at the JSON/validation level.
	return service.NewInventoryService(stockRepo, reservationRepo, nil, producer, logger, 300)
}

func testHandler(stockRepo *mockStockRepository, reservationRepo *mockReservationRepository) *InventoryHandler {
	svc := testInventoryService(stockRepo, reservationRepo)
	return NewInventoryHandler(svc, testLogger())
}

// setupRouter creates a chi router matching the production route layout for inventory.
func setupRouter(handler *InventoryHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1/inventory", func(r chi.Router) {
		r.Use(ContentTypeJSON)
		r.Post("/", handler.InitializeStock)
		r.Get("/{productId}/variants/{variantId}", handler.GetStock)
		r.Put("/{productId}/variants/{variantId}", handler.AdjustStock)
		r.Post("/check", handler.CheckAvailability)
		r.Post("/reserve", handler.ReserveStock)
		r.Post("/release", handler.ReleaseReservation)
		r.Post("/confirm", handler.ConfirmReservation)
		r.Get("/low-stock", handler.ListLowStock)
	})
	return r
}

// decodeResponse reads the response body into the httputil.Response struct.
func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) httputil.Response {
	t.Helper()
	var resp httputil.Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	return resp
}

// validUUID returns a fixed valid UUID for use in tests.
const (
	validProductID     = "550e8400-e29b-41d4-a716-446655440001"
	validVariantID     = "550e8400-e29b-41d4-a716-446655440002"
	validCheckoutID    = "550e8400-e29b-41d4-a716-446655440003"
	validReservationID = "550e8400-e29b-41d4-a716-446655440004"
)

func sampleStock() *domain.Stock {
	return &domain.Stock{
		ID:                "stock-001",
		ProductID:         validProductID,
		VariantID:         validVariantID,
		WarehouseID:       domain.DefaultWarehouseID,
		Quantity:          100,
		Reserved:          5,
		LowStockThreshold: 10,
		UpdatedAt:         time.Now().UTC(),
	}
}

// ============================================================================
// POST /api/v1/inventory - InitializeStock
// ============================================================================

func TestInitializeStock_Success(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	stockRepo.On("CreateStock", mock.Anything, mock.AnythingOfType("*domain.Stock")).
		Return(sampleStock(), nil)

	body, _ := json.Marshal(InitializeStockRequest{
		ProductID:         validProductID,
		VariantID:         validVariantID,
		Quantity:          100,
		LowStockThreshold: 10,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	stockRepo.AssertExpectations(t)
}

func TestInitializeStock_InvalidJSON(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestInitializeStock_ValidationError_MissingProductID(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	body, _ := json.Marshal(InitializeStockRequest{
		// ProductID is missing
		VariantID: validVariantID,
		Quantity:  100,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestInitializeStock_ValidationError_InvalidUUID(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	body, _ := json.Marshal(InitializeStockRequest{
		ProductID: "not-a-uuid",
		VariantID: validVariantID,
		Quantity:  100,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestInitializeStock_ValidationError_NegativeQuantity(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	body, _ := json.Marshal(map[string]interface{}{
		"product_id": validProductID,
		"variant_id": validVariantID,
		"quantity":   -5,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

// ============================================================================
// GET /api/v1/inventory/{productId}/variants/{variantId} - GetStock
// ============================================================================

func TestGetStock_Success(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	stock := sampleStock()
	stockRepo.On("GetByProductVariant", mock.Anything, validProductID, validVariantID).
		Return(stock, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/"+validProductID+"/variants/"+validVariantID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	stockRepo.AssertExpectations(t)
}

func TestGetStock_InvalidProductID(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/not-a-uuid/variants/"+validVariantID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid UUID")
}

func TestGetStock_InvalidVariantID(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/"+validProductID+"/variants/bad-variant-id", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid UUID")
}

// ============================================================================
// PUT /api/v1/inventory/{productId}/variants/{variantId} - AdjustStock
// ============================================================================

func TestAdjustStock_Success(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	stock := sampleStock()
	stockRepo.On("AdjustQuantity", mock.Anything, validProductID, validVariantID, 10, "adjustment", (*string)(nil)).
		Return(nil)
	stockRepo.On("GetByProductVariant", mock.Anything, validProductID, validVariantID).
		Return(stock, nil)

	body, _ := json.Marshal(AdjustStockRequest{
		Delta:  10,
		Reason: "adjustment",
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/inventory/"+validProductID+"/variants/"+validVariantID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	stockRepo.AssertExpectations(t)
}

func TestAdjustStock_InvalidJSON(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/inventory/"+validProductID+"/variants/"+validVariantID, bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestAdjustStock_InvalidReasonEnum(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	body, _ := json.Marshal(AdjustStockRequest{
		Delta:  10,
		Reason: "invalid_reason",
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/inventory/"+validProductID+"/variants/"+validVariantID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestAdjustStock_InvalidProductUUID(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	body, _ := json.Marshal(AdjustStockRequest{
		Delta:  10,
		Reason: "adjustment",
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/inventory/bad-uuid/variants/"+validVariantID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

func TestAdjustStock_InvalidVariantUUID(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	body, _ := json.Marshal(AdjustStockRequest{
		Delta:  10,
		Reason: "adjustment",
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/inventory/"+validProductID+"/variants/bad-uuid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

func TestAdjustStock_MissingDelta(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	// Delta is 0 which fails "required" validation
	body, _ := json.Marshal(map[string]interface{}{
		"delta":  0,
		"reason": "adjustment",
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/inventory/"+validProductID+"/variants/"+validVariantID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestAdjustStock_AllValidReasons(t *testing.T) {
	validReasons := []string{"order", "return", "adjustment", "reservation"}

	for _, reason := range validReasons {
		t.Run("reason_"+reason, func(t *testing.T) {
			stockRepo := new(mockStockRepository)
			reservationRepo := new(mockReservationRepository)
			handler := testHandler(stockRepo, reservationRepo)
			router := setupRouter(handler)

			stock := sampleStock()
			stockRepo.On("AdjustQuantity", mock.Anything, validProductID, validVariantID, 5, reason, (*string)(nil)).
				Return(nil)
			stockRepo.On("GetByProductVariant", mock.Anything, validProductID, validVariantID).
				Return(stock, nil)

			body, _ := json.Marshal(AdjustStockRequest{
				Delta:  5,
				Reason: reason,
			})

			req := httptest.NewRequest(http.MethodPut, "/api/v1/inventory/"+validProductID+"/variants/"+validVariantID, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			resp := decodeResponse(t, rec)
			assert.Nil(t, resp.Error)
			stockRepo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// POST /api/v1/inventory/check - CheckAvailability
// ============================================================================

func TestCheckAvailability_Success(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	results := []domain.StockCheckResult{
		{
			ProductID: validProductID,
			VariantID: validVariantID,
			Requested: 5,
			Available: 95,
			InStock:   true,
		},
	}
	stockRepo.On("BulkCheck", mock.Anything, mock.AnythingOfType("[]domain.StockCheckItem")).
		Return(results, nil)

	body, _ := json.Marshal(CheckAvailabilityRequest{
		Items: []StockCheckItemRequest{
			{
				ProductID: validProductID,
				VariantID: validVariantID,
				Quantity:  5,
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)

	// Verify response contains items and all_available fields
	dataMap, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, dataMap, "items")
	assert.Contains(t, dataMap, "all_available")
	assert.Equal(t, true, dataMap["all_available"])
	stockRepo.AssertExpectations(t)
}

func TestCheckAvailability_EmptyItems(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	body, _ := json.Marshal(CheckAvailabilityRequest{
		Items: []StockCheckItemRequest{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestCheckAvailability_ValidationError_MissingProductID(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	body, _ := json.Marshal(CheckAvailabilityRequest{
		Items: []StockCheckItemRequest{
			{
				// ProductID missing
				VariantID: validVariantID,
				Quantity:  5,
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestCheckAvailability_InvalidJSON(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/check", bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
}

func TestCheckAvailability_ZeroQuantity(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	body, _ := json.Marshal(CheckAvailabilityRequest{
		Items: []StockCheckItemRequest{
			{
				ProductID: validProductID,
				VariantID: validVariantID,
				Quantity:  0, // fails gte=1
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

// ============================================================================
// POST /api/v1/inventory/reserve - ReserveStock
// ============================================================================

func TestReserveStock_ValidationError_MissingCheckoutID(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	body, _ := json.Marshal(ReserveStockRequest{
		// CheckoutID missing
		Items: []StockCheckItemRequest{
			{
				ProductID: validProductID,
				VariantID: validVariantID,
				Quantity:  2,
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/reserve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestReserveStock_ValidationError_EmptyItems(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	body, _ := json.Marshal(ReserveStockRequest{
		CheckoutID: validCheckoutID,
		Items:      []StockCheckItemRequest{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/reserve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestReserveStock_InvalidJSON(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/reserve", bytes.NewReader([]byte(`broken`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestReserveStock_ValidationError_InvalidItemUUID(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	body, _ := json.Marshal(ReserveStockRequest{
		CheckoutID: validCheckoutID,
		Items: []StockCheckItemRequest{
			{
				ProductID: "not-a-uuid",
				VariantID: validVariantID,
				Quantity:  2,
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/reserve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

// ============================================================================
// POST /api/v1/inventory/release - ReleaseReservation
// ============================================================================

func TestReleaseReservation_InvalidJSON(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/release", bytes.NewReader([]byte(`{bad json`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestReleaseReservation_ValidationError_MissingReservationID(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	body, _ := json.Marshal(ReleaseReservationRequest{
		ReservationID: "", // fails required
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/release", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestReleaseReservation_ValidationError_InvalidUUID(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	body, _ := json.Marshal(ReleaseReservationRequest{
		ReservationID: "not-a-uuid",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/release", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

// ============================================================================
// POST /api/v1/inventory/confirm - ConfirmReservation
// ============================================================================

func TestConfirmReservation_InvalidJSON(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/confirm", bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestConfirmReservation_ValidationError_MissingReservationID(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	body, _ := json.Marshal(ConfirmReservationRequest{
		ReservationID: "",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestConfirmReservation_ValidationError_InvalidUUID(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	body, _ := json.Marshal(ConfirmReservationRequest{
		ReservationID: "bad-uuid",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

// ============================================================================
// GET /api/v1/inventory/low-stock - ListLowStock
// ============================================================================

func TestListLowStock_Success(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	stocks := []domain.Stock{
		{
			ID:                "stock-001",
			ProductID:         validProductID,
			VariantID:         validVariantID,
			WarehouseID:       domain.DefaultWarehouseID,
			Quantity:          8,
			Reserved:          3,
			LowStockThreshold: 10,
			UpdatedAt:         time.Now().UTC(),
		},
	}
	stockRepo.On("ListLowStock", mock.Anything, 1, 20).
		Return(stocks, 1, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/low-stock", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Decode raw JSON to check paginated structure
	var raw map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&raw)
	require.NoError(t, err)
	assert.NotNil(t, raw["data"])
	assert.Equal(t, float64(1), raw["total_count"])
	assert.Equal(t, float64(1), raw["page"])
	assert.Equal(t, float64(20), raw["per_page"])
	assert.Equal(t, float64(1), raw["total_pages"])
	assert.Equal(t, false, raw["has_next"])
	stockRepo.AssertExpectations(t)
}

func TestListLowStock_WithPagination(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	stocks := []domain.Stock{
		{
			ID:                "stock-001",
			ProductID:         validProductID,
			VariantID:         validVariantID,
			WarehouseID:       domain.DefaultWarehouseID,
			Quantity:          5,
			Reserved:          0,
			LowStockThreshold: 10,
			UpdatedAt:         time.Now().UTC(),
		},
	}
	stockRepo.On("ListLowStock", mock.Anything, 2, 10).
		Return(stocks, 25, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/low-stock?page=2&per_page=10", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var raw map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&raw)
	require.NoError(t, err)
	assert.Equal(t, float64(25), raw["total_count"])
	assert.Equal(t, float64(2), raw["page"])
	assert.Equal(t, float64(10), raw["per_page"])
	assert.Equal(t, float64(3), raw["total_pages"])
	assert.Equal(t, true, raw["has_next"])
	stockRepo.AssertExpectations(t)
}

func TestListLowStock_InvalidPage(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/low-stock?page=0", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "page")
}

func TestListLowStock_InvalidPerPage(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/low-stock?per_page=200", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "per_page")
}

func TestListLowStock_NonNumericPage(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/low-stock?page=abc", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

// ============================================================================
// ContentTypeJSON middleware tests
// ============================================================================

func TestContentTypeJSON_RejectsNonJSON(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	body, _ := json.Marshal(InitializeStockRequest{
		ProductID: validProductID,
		VariantID: validVariantID,
		Quantity:  10,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
}

func TestContentTypeJSON_AcceptsJSON(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	handler := testHandler(stockRepo, reservationRepo)
	router := setupRouter(handler)

	stockRepo.On("CreateStock", mock.Anything, mock.AnythingOfType("*domain.Stock")).
		Return(sampleStock(), nil)

	body, _ := json.Marshal(InitializeStockRequest{
		ProductID:         validProductID,
		VariantID:         validVariantID,
		Quantity:          100,
		LowStockThreshold: 10,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	stockRepo.AssertExpectations(t)
}

// ============================================================================
// Table-driven tests for validation edge cases
// ============================================================================

func TestInitializeStock_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		body          interface{}
		expectStatus  int
		expectErrCode string
	}{
		{
			name: "valid with all fields",
			body: InitializeStockRequest{
				ProductID:         validProductID,
				VariantID:         validVariantID,
				WarehouseID:       "550e8400-e29b-41d4-a716-446655440099",
				Quantity:          50,
				LowStockThreshold: 5,
			},
			expectStatus: http.StatusCreated,
		},
		{
			name: "valid with zero quantity",
			body: InitializeStockRequest{
				ProductID: validProductID,
				VariantID: validVariantID,
				Quantity:  0,
			},
			expectStatus: http.StatusCreated,
		},
		{
			name: "missing variant_id",
			body: InitializeStockRequest{
				ProductID: validProductID,
				Quantity:  10,
			},
			expectStatus:  http.StatusBadRequest,
			expectErrCode: "VALIDATION_ERROR",
		},
		{
			name: "invalid warehouse UUID",
			body: InitializeStockRequest{
				ProductID:   validProductID,
				VariantID:   validVariantID,
				WarehouseID: "bad-warehouse-id",
				Quantity:    10,
			},
			expectStatus:  http.StatusBadRequest,
			expectErrCode: "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stockRepo := new(mockStockRepository)
			reservationRepo := new(mockReservationRepository)
			handler := testHandler(stockRepo, reservationRepo)
			router := setupRouter(handler)

			if tt.expectStatus == http.StatusCreated {
				stockRepo.On("CreateStock", mock.Anything, mock.AnythingOfType("*domain.Stock")).
					Return(sampleStock(), nil)
			}

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectStatus, rec.Code)

			if tt.expectErrCode != "" {
				resp := decodeResponse(t, rec)
				require.NotNil(t, resp.Error)
				assert.Equal(t, tt.expectErrCode, resp.Error.Code)
			}
		})
	}
}
