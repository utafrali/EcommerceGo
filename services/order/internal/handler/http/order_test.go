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
	"github.com/utafrali/EcommerceGo/services/order/internal/domain"
	"github.com/utafrali/EcommerceGo/services/order/internal/event"
	"github.com/utafrali/EcommerceGo/services/order/internal/repository"
	"github.com/utafrali/EcommerceGo/services/order/internal/service"
)

// --- Mock OrderRepository ---

type mockOrderRepository struct {
	mock.Mock
}

func (m *mockOrderRepository) Create(ctx context.Context, order *domain.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *mockOrderRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Order), args.Error(1)
}

func (m *mockOrderRepository) List(ctx context.Context, filter repository.OrderFilter) ([]domain.Order, int, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]domain.Order), args.Int(1), args.Error(2)
}

func (m *mockOrderRepository) UpdateStatus(ctx context.Context, id string, status string, reason string) error {
	args := m.Called(ctx, id, status, reason)
	return args.Error(0)
}

// --- Test Helpers ---

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func testEventProducer() *event.Producer {
	logger := testLogger()
	kafkaCfg := pkgkafka.DefaultProducerConfig([]string{"localhost:19092"})
	kafkaProducer := pkgkafka.NewProducer(kafkaCfg, logger)
	return event.NewProducer(kafkaProducer, logger)
}

func testOrderService(repo *mockOrderRepository) *service.OrderService {
	logger := testLogger()
	producer := testEventProducer()
	return service.NewOrderService(repo, producer, logger)
}

func testOrderHandler(repo *mockOrderRepository) *OrderHandler {
	svc := testOrderService(repo)
	return NewOrderHandler(svc, testLogger())
}

// setupOrderRouter creates a chi router matching the production route layout.
func setupOrderRouter(handler *OrderHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1/orders", func(r chi.Router) {
		r.Use(ContentTypeJSON)
		r.Post("/", handler.CreateOrder)
		r.Get("/", handler.ListOrders)
		r.Get("/{id}", handler.GetOrder)
		r.Put("/{id}/status", handler.UpdateOrderStatus)
		r.Post("/{id}/cancel", handler.CancelOrder)
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

// sampleOrder returns a realistic order for use in test expectations.
func sampleOrder() *domain.Order {
	now := time.Now().UTC()
	return &domain.Order{
		ID:     "550e8400-e29b-41d4-a716-446655440001",
		UserID: "user-456",
		Status: domain.OrderStatusPending,
		Items: []domain.OrderItem{
			{
				ID:        "550e8400-e29b-41d4-a716-446655440010",
				OrderID:   "550e8400-e29b-41d4-a716-446655440001",
				ProductID: "550e8400-e29b-41d4-a716-446655440020",
				VariantID: "550e8400-e29b-41d4-a716-446655440021",
				Name:      "Premium T-Shirt",
				SKU:       "TSH-BLK-M",
				Price:     1999,
				Quantity:  2,
			},
		},
		SubtotalAmount: 3998,
		DiscountAmount: 0,
		ShippingAmount: 500,
		TotalAmount:    4498,
		Currency:       "USD",
		ShippingAddress: &domain.Address{
			FullName:    "John Doe",
			AddressLine: "123 Main St",
			City:        "New York",
			State:       "NY",
			PostalCode:  "10001",
			Country:     "US",
			Phone:       "+12125551234",
		},
		Notes:     "Leave at door",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// validCreateOrderJSON returns a valid JSON body for POST /api/v1/orders.
func validCreateOrderJSON() []byte {
	body := CreateOrderRequest{
		UserID: "user-456",
		Items: []CreateOrderItemRequest{
			{
				ProductID: "550e8400-e29b-41d4-a716-446655440020",
				VariantID: "550e8400-e29b-41d4-a716-446655440021",
				Name:      "Premium T-Shirt",
				SKU:       "TSH-BLK-M",
				Price:     1999,
				Quantity:  2,
			},
		},
		DiscountAmount: 0,
		ShippingAmount: 500,
		Currency:       "USD",
		ShippingAddress: &domain.Address{
			FullName:    "John Doe",
			AddressLine: "123 Main St",
			City:        "New York",
			State:       "NY",
			PostalCode:  "10001",
			Country:     "US",
			Phone:       "+12125551234",
		},
		Notes: "Leave at door",
	}
	b, _ := json.Marshal(body)
	return b
}

// ============================================================================
// POST /api/v1/orders - CreateOrder
// ============================================================================

func TestCreateOrder_Success(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Order")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(validCreateOrderJSON()))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)

	// Verify the returned order data contains expected fields.
	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "user-456", data["user_id"])
	assert.Equal(t, "pending", data["status"])
	assert.Equal(t, "USD", data["currency"])
	assert.Equal(t, "Leave at door", data["notes"])

	repo.AssertExpectations(t)
}

func TestCreateOrder_InvalidJSON(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader([]byte(`{invalid json`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestCreateOrder_ValidationError_NoItems(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	body := CreateOrderRequest{
		UserID:   "user-456",
		Items:    []CreateOrderItemRequest{}, // empty items
		Currency: "USD",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
	assert.NotNil(t, resp.Error.Fields)
}

func TestCreateOrder_ValidationError_MissingUserID(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	body := CreateOrderRequest{
		UserID: "", // missing required field
		Items: []CreateOrderItemRequest{
			{
				ProductID: "prod-1",
				Name:      "Product",
				Price:     999,
				Quantity:  1,
			},
		},
		Currency: "USD",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestCreateOrder_ValidationError_InvalidCurrency(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	body := CreateOrderRequest{
		UserID: "user-456",
		Items: []CreateOrderItemRequest{
			{
				ProductID: "prod-1",
				Name:      "Product",
				Price:     999,
				Quantity:  1,
			},
		},
		Currency: "TOOLONG", // must be 3 characters
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestCreateOrder_ServiceError(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Order")).
		Return(apperrors.Internal(assert.AnError))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(validCreateOrderJSON()))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)

	repo.AssertExpectations(t)
}

func TestCreateOrder_EmptyBody(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader([]byte("")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
}

// ============================================================================
// GET /api/v1/orders - ListOrders
// ============================================================================

func TestListOrders_Success(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	order := sampleOrder()
	expectedFilter := repository.OrderFilter{Page: 1, PerPage: 20}
	repo.On("List", mock.Anything, expectedFilter).
		Return([]domain.Order{*order}, 1, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Decode into paginated response structure.
	var paginatedResp struct {
		Data       []map[string]interface{} `json:"data"`
		TotalCount int                      `json:"total_count"`
		Page       int                      `json:"page"`
		PerPage    int                      `json:"per_page"`
		TotalPages int                      `json:"total_pages"`
		HasNext    bool                     `json:"has_next"`
	}
	err := json.NewDecoder(rec.Body).Decode(&paginatedResp)
	require.NoError(t, err)
	assert.Equal(t, 1, paginatedResp.TotalCount)
	assert.Equal(t, 1, paginatedResp.Page)
	assert.Equal(t, 20, paginatedResp.PerPage)
	assert.False(t, paginatedResp.HasNext)
	assert.Len(t, paginatedResp.Data, 1)

	repo.AssertExpectations(t)
}

func TestListOrders_WithPagination(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	expectedFilter := repository.OrderFilter{Page: 2, PerPage: 10}
	repo.On("List", mock.Anything, expectedFilter).
		Return([]domain.Order{}, 25, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders?page=2&per_page=10", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var paginatedResp struct {
		Data       []map[string]interface{} `json:"data"`
		TotalCount int                      `json:"total_count"`
		Page       int                      `json:"page"`
		PerPage    int                      `json:"per_page"`
		TotalPages int                      `json:"total_pages"`
		HasNext    bool                     `json:"has_next"`
	}
	err := json.NewDecoder(rec.Body).Decode(&paginatedResp)
	require.NoError(t, err)
	assert.Equal(t, 25, paginatedResp.TotalCount)
	assert.Equal(t, 2, paginatedResp.Page)
	assert.Equal(t, 10, paginatedResp.PerPage)
	assert.True(t, paginatedResp.HasNext)

	repo.AssertExpectations(t)
}

func TestListOrders_FilterByUserID(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	userID := "user-456"
	expectedFilter := repository.OrderFilter{
		Page:    1,
		PerPage: 20,
		UserID:  &userID,
	}
	repo.On("List", mock.Anything, expectedFilter).
		Return([]domain.Order{*sampleOrder()}, 1, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders?user_id=user-456", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	repo.AssertExpectations(t)
}

func TestListOrders_FilterByStatus(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	status := "pending"
	expectedFilter := repository.OrderFilter{
		Page:    1,
		PerPage: 20,
		Status:  &status,
	}
	repo.On("List", mock.Anything, expectedFilter).
		Return([]domain.Order{*sampleOrder()}, 1, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders?status=pending", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	repo.AssertExpectations(t)
}

func TestListOrders_FilterByUserIDAndStatus(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	userID := "user-456"
	status := "confirmed"
	expectedFilter := repository.OrderFilter{
		Page:    1,
		PerPage: 20,
		UserID:  &userID,
		Status:  &status,
	}
	repo.On("List", mock.Anything, expectedFilter).
		Return([]domain.Order{}, 0, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders?user_id=user-456&status=confirmed", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	repo.AssertExpectations(t)
}

func TestListOrders_InvalidPage(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders?page=abc", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "page")
}

func TestListOrders_InvalidPerPage(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders?per_page=0", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "per_page")
}

func TestListOrders_PerPageTooLarge(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders?per_page=101", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

func TestListOrders_NegativePage(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders?page=-1", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

func TestListOrders_ServiceError(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	expectedFilter := repository.OrderFilter{Page: 1, PerPage: 20}
	repo.On("List", mock.Anything, expectedFilter).
		Return([]domain.Order{}, 0, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)

	repo.AssertExpectations(t)
}

// ============================================================================
// GET /api/v1/orders/{id} - GetOrder
// ============================================================================

func TestGetOrder_Success(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	order := sampleOrder()
	repo.On("GetByID", mock.Anything, order.ID).Return(order, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+order.ID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, order.ID, data["id"])
	assert.Equal(t, "user-456", data["user_id"])
	assert.Equal(t, "pending", data["status"])
	assert.Equal(t, float64(4498), data["total_amount"])
	assert.Equal(t, "USD", data["currency"])

	repo.AssertExpectations(t)
}

func TestGetOrder_InvalidUUID(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/not-a-uuid", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid UUID")
}

func TestGetOrder_NotFound(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	orderID := "550e8400-e29b-41d4-a716-446655440099"
	repo.On("GetByID", mock.Anything, orderID).
		Return(nil, apperrors.NotFound("order", orderID))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+orderID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)

	repo.AssertExpectations(t)
}

// ============================================================================
// PUT /api/v1/orders/{id}/status - UpdateOrderStatus
// ============================================================================

func TestUpdateOrderStatus_Success(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	order := sampleOrder()
	// Pending -> Confirmed is a valid transition.
	repo.On("GetByID", mock.Anything, order.ID).Return(order, nil)
	repo.On("UpdateStatus", mock.Anything, order.ID, "confirmed", "").Return(nil)

	body, _ := json.Marshal(UpdateStatusRequest{Status: "confirmed"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/"+order.ID+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "confirmed", data["status"])

	repo.AssertExpectations(t)
}

func TestUpdateOrderStatus_WithReason(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	order := sampleOrder()
	repo.On("GetByID", mock.Anything, order.ID).Return(order, nil)
	repo.On("UpdateStatus", mock.Anything, order.ID, "canceled", "customer request").Return(nil)

	body, _ := json.Marshal(UpdateStatusRequest{Status: "canceled", Reason: "customer request"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/"+order.ID+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "canceled", data["status"])
	assert.Equal(t, "customer request", data["canceled_reason"])

	repo.AssertExpectations(t)
}

func TestUpdateOrderStatus_InvalidUUID(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	body, _ := json.Marshal(UpdateStatusRequest{Status: "confirmed"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/bad-uuid/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid UUID")
}

func TestUpdateOrderStatus_InvalidStatus(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	orderID := "550e8400-e29b-41d4-a716-446655440001"

	body, _ := json.Marshal(UpdateStatusRequest{Status: "nonexistent_status"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/"+orderID+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// The service validates the status and returns InvalidInput.
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid status")
}

func TestUpdateOrderStatus_InvalidTransition(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	order := sampleOrder()
	order.Status = domain.OrderStatusDelivered
	repo.On("GetByID", mock.Anything, order.ID).Return(order, nil)

	// Delivered -> Confirmed is not a valid transition.
	body, _ := json.Marshal(UpdateStatusRequest{Status: "confirmed"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/"+order.ID+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "cannot transition")

	repo.AssertExpectations(t)
}

func TestUpdateOrderStatus_NotFound(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	orderID := "550e8400-e29b-41d4-a716-446655440099"
	repo.On("GetByID", mock.Anything, orderID).
		Return(nil, apperrors.NotFound("order", orderID))

	body, _ := json.Marshal(UpdateStatusRequest{Status: "confirmed"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/"+orderID+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)

	repo.AssertExpectations(t)
}

func TestUpdateOrderStatus_MissingStatus(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	orderID := "550e8400-e29b-41d4-a716-446655440001"

	// Empty status should fail validation.
	body, _ := json.Marshal(UpdateStatusRequest{Status: ""})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/"+orderID+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestUpdateOrderStatus_InvalidJSON(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	orderID := "550e8400-e29b-41d4-a716-446655440001"

	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/"+orderID+"/status", bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

// ============================================================================
// POST /api/v1/orders/{id}/cancel - CancelOrder
// ============================================================================

func TestCancelOrder_Success(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	order := sampleOrder()
	repo.On("GetByID", mock.Anything, order.ID).Return(order, nil)
	repo.On("UpdateStatus", mock.Anything, order.ID, domain.OrderStatusCanceled, "changed my mind").Return(nil)

	body, _ := json.Marshal(CancelOrderRequest{Reason: "changed my mind"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+order.ID+"/cancel", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "canceled", data["status"])
	assert.Equal(t, "changed my mind", data["canceled_reason"])

	repo.AssertExpectations(t)
}

func TestCancelOrder_EmptyBody(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	order := sampleOrder()
	repo.On("GetByID", mock.Anything, order.ID).Return(order, nil)
	repo.On("UpdateStatus", mock.Anything, order.ID, domain.OrderStatusCanceled, "").Return(nil)

	// Empty body should be allowed for cancel; reason defaults to empty.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+order.ID+"/cancel", bytes.NewReader([]byte("")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "canceled", data["status"])

	repo.AssertExpectations(t)
}

func TestCancelOrder_NilBody(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	order := sampleOrder()
	repo.On("GetByID", mock.Anything, order.ID).Return(order, nil)
	repo.On("UpdateStatus", mock.Anything, order.ID, domain.OrderStatusCanceled, "").Return(nil)

	// Nil body should be handled gracefully by the cancel handler.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+order.ID+"/cancel", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)

	repo.AssertExpectations(t)
}

func TestCancelOrder_InvalidUUID(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/invalid-uuid/cancel", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid UUID")
}

func TestCancelOrder_NotFound(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	orderID := "550e8400-e29b-41d4-a716-446655440099"
	repo.On("GetByID", mock.Anything, orderID).
		Return(nil, apperrors.NotFound("order", orderID))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+orderID+"/cancel", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)

	repo.AssertExpectations(t)
}

func TestCancelOrder_InvalidTransition(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	order := sampleOrder()
	order.Status = domain.OrderStatusDelivered // delivered cannot be canceled
	repo.On("GetByID", mock.Anything, order.ID).Return(order, nil)

	body, _ := json.Marshal(CancelOrderRequest{Reason: "too late"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+order.ID+"/cancel", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "cannot cancel")

	repo.AssertExpectations(t)
}

// ============================================================================
// ContentTypeJSON middleware tests
// ============================================================================

func TestContentTypeJSON_RejectsXML(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader([]byte(`<xml/>`)))
	req.Header.Set("Content-Type", "application/xml")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
}

func TestContentTypeJSON_AcceptsApplicationJSON(t *testing.T) {
	repo := new(mockOrderRepository)
	handler := testOrderHandler(repo)
	router := setupOrderRouter(handler)

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Order")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(validCreateOrderJSON()))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// Table-driven test: UpdateOrderStatus valid transitions
// ============================================================================

func TestUpdateOrderStatus_ValidTransitions(t *testing.T) {
	tests := []struct {
		name      string
		fromState string
		toState   string
	}{
		{"pending to confirmed", domain.OrderStatusPending, domain.OrderStatusConfirmed},
		{"pending to canceled", domain.OrderStatusPending, domain.OrderStatusCanceled},
		{"confirmed to processing", domain.OrderStatusConfirmed, domain.OrderStatusProcessing},
		{"confirmed to canceled", domain.OrderStatusConfirmed, domain.OrderStatusCanceled},
		{"processing to shipped", domain.OrderStatusProcessing, domain.OrderStatusShipped},
		{"processing to canceled", domain.OrderStatusProcessing, domain.OrderStatusCanceled},
		{"shipped to delivered", domain.OrderStatusShipped, domain.OrderStatusDelivered},
		{"delivered to refunded", domain.OrderStatusDelivered, domain.OrderStatusRefunded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockOrderRepository)
			handler := testOrderHandler(repo)
			router := setupOrderRouter(handler)

			order := sampleOrder()
			order.Status = tt.fromState
			repo.On("GetByID", mock.Anything, order.ID).Return(order, nil)
			repo.On("UpdateStatus", mock.Anything, order.ID, tt.toState, "").Return(nil)

			body, _ := json.Marshal(UpdateStatusRequest{Status: tt.toState})
			req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/"+order.ID+"/status", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code, "expected 200 for %s -> %s", tt.fromState, tt.toState)
			resp := decodeResponse(t, rec)
			assert.Nil(t, resp.Error)

			data, ok := resp.Data.(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, tt.toState, data["status"])

			repo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Table-driven test: UpdateOrderStatus invalid transitions
// ============================================================================

func TestUpdateOrderStatus_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name      string
		fromState string
		toState   string
	}{
		{"canceled to confirmed", domain.OrderStatusCanceled, domain.OrderStatusConfirmed},
		{"refunded to pending", domain.OrderStatusRefunded, domain.OrderStatusPending},
		{"shipped to confirmed", domain.OrderStatusShipped, domain.OrderStatusConfirmed},
		{"delivered to pending", domain.OrderStatusDelivered, domain.OrderStatusPending},
		{"pending to shipped", domain.OrderStatusPending, domain.OrderStatusShipped},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockOrderRepository)
			handler := testOrderHandler(repo)
			router := setupOrderRouter(handler)

			order := sampleOrder()
			order.Status = tt.fromState
			repo.On("GetByID", mock.Anything, order.ID).Return(order, nil)

			body, _ := json.Marshal(UpdateStatusRequest{Status: tt.toState})
			req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/"+order.ID+"/status", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code, "expected 400 for %s -> %s", tt.fromState, tt.toState)
			resp := decodeResponse(t, rec)
			require.NotNil(t, resp.Error)
			assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
			assert.Contains(t, resp.Error.Message, "cannot transition")

			repo.AssertExpectations(t)
		})
	}
}
