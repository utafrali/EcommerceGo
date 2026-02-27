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

	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/checkout/internal/domain"
	"github.com/utafrali/EcommerceGo/services/checkout/internal/event"
	"github.com/utafrali/EcommerceGo/services/checkout/internal/service"
)

// --- Mock Checkout Repository ---

type mockCheckoutRepository struct {
	mock.Mock
}

func (m *mockCheckoutRepository) Create(ctx context.Context, session *domain.CheckoutSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *mockCheckoutRepository) GetByID(ctx context.Context, id string) (*domain.CheckoutSession, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CheckoutSession), args.Error(1)
}

func (m *mockCheckoutRepository) Update(ctx context.Context, session *domain.CheckoutSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *mockCheckoutRepository) GetActiveByUserID(ctx context.Context, userID string) (*domain.CheckoutSession, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CheckoutSession), args.Error(1)
}

func (m *mockCheckoutRepository) ListExpired(ctx context.Context, before time.Time) ([]domain.CheckoutSession, error) {
	args := m.Called(ctx, before)
	return args.Get(0).([]domain.CheckoutSession), args.Error(1)
}

// --- Test Helpers ---

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func testEventProducer() *event.Producer {
	logger := testLogger()
	kafkaCfg := pkgkafka.DefaultProducerConfig([]string{"localhost:9092"})
	kafkaProducer := pkgkafka.NewProducer(kafkaCfg, logger)
	return event.NewProducer(kafkaProducer, logger)
}

func testService(repo *mockCheckoutRepository) *service.CheckoutService {
	logger := testLogger()
	producer := testEventProducer()
	return service.NewCheckoutService(repo, producer, logger)
}

func testHandler(repo *mockCheckoutRepository) *CheckoutHandler {
	svc := testService(repo)
	return NewCheckoutHandler(svc, testLogger())
}

func activeSession() *domain.CheckoutSession {
	return &domain.CheckoutSession{
		ID:     "session-123",
		UserID: "user-456",
		Status: domain.StatusInitiated,
		Items: []domain.CheckoutItem{
			{
				ProductID: "550e8400-e29b-41d4-a716-446655440001",
				VariantID: "550e8400-e29b-41d4-a716-446655440002",
				Name:      "Test Product",
				SKU:       "TST-001",
				Price:     2999,
				Quantity:  2,
			},
		},
		SubtotalAmount: 5998,
		TotalAmount:    5998,
		Currency:       "USD",
		ExpiresAt:      time.Now().UTC().Add(30 * time.Minute),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
}

// setupRouter creates a chi router with the checkout handler routes,
// matching the production router layout.
func setupRouter(handler *CheckoutHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1/checkout", func(r chi.Router) {
		r.Post("/", handler.InitiateCheckout)
		r.Get("/{id}", handler.GetCheckout)
		r.Put("/{id}/shipping", handler.SetShippingAddress)
		r.Put("/{id}/payment", handler.SetPaymentMethod)
		r.Post("/{id}/process", handler.ProcessCheckout)
		r.Post("/{id}/cancel", handler.CancelCheckout)
	})
	return r
}

// decodeResponse reads the response body into the response struct.
func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) response {
	t.Helper()
	var resp response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	return resp
}

// ============================================================================
// Unit tests for getUserID helper
// ============================================================================

func TestGetUserID_Present(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", "user-456")

	got := getUserID(req)
	assert.Equal(t, "user-456", got)
}

func TestGetUserID_Missing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	got := getUserID(req)
	assert.Equal(t, "", got)
}

func TestGetUserID_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", "")

	got := getUserID(req)
	assert.Equal(t, "", got)
}

// ============================================================================
// Unit tests for authorizeSession helper
// ============================================================================

func TestAuthorizeSession_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", "user-456")
	rec := httptest.NewRecorder()

	session := &domain.CheckoutSession{UserID: "user-456"}
	result := authorizeSession(rec, req, session)

	assert.True(t, result)
	// No response body should be written on success.
	assert.Equal(t, http.StatusOK, rec.Code) // default 200 (not explicitly written)
}

func TestAuthorizeSession_MissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No X-User-ID header set.
	rec := httptest.NewRecorder()

	session := &domain.CheckoutSession{UserID: "user-456"}
	result := authorizeSession(rec, req, session)

	assert.False(t, result)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "X-User-ID")
}

func TestAuthorizeSession_UserMismatch(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-ID", "attacker-999")
	rec := httptest.NewRecorder()

	session := &domain.CheckoutSession{UserID: "user-456"}
	result := authorizeSession(rec, req, session)

	assert.False(t, result)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "FORBIDDEN", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "do not have access")
}

// ============================================================================
// GetCheckout handler authorization tests
// ============================================================================

func TestGetCheckout_AuthorizedUser(t *testing.T) {
	repo := new(mockCheckoutRepository)
	handler := testHandler(repo)
	router := setupRouter(handler)

	session := activeSession()
	repo.On("GetByID", mock.Anything, "session-123").Return(session, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/checkout/session-123", nil)
	req.Header.Set("X-User-ID", "user-456") // matches session.UserID
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestGetCheckout_MissingUserIDHeader(t *testing.T) {
	repo := new(mockCheckoutRepository)
	handler := testHandler(repo)
	router := setupRouter(handler)

	session := activeSession()
	repo.On("GetByID", mock.Anything, "session-123").Return(session, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/checkout/session-123", nil)
	// No X-User-ID header.
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "X-User-ID")
	repo.AssertExpectations(t)
}

func TestGetCheckout_ForbiddenDifferentUser(t *testing.T) {
	repo := new(mockCheckoutRepository)
	handler := testHandler(repo)
	router := setupRouter(handler)

	session := activeSession()
	repo.On("GetByID", mock.Anything, "session-123").Return(session, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/checkout/session-123", nil)
	req.Header.Set("X-User-ID", "attacker-999") // does NOT match session.UserID
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "FORBIDDEN", resp.Error.Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// SetShippingAddress handler authorization tests
// ============================================================================

func validShippingJSON() []byte {
	body := SetShippingAddressRequest{
		FullName:    "John Doe",
		AddressLine: "123 Main St",
		City:        "New York",
		State:       "NY",
		PostalCode:  "10001",
		Country:     "US",
		Phone:       "+1234567890",
	}
	b, _ := json.Marshal(body)
	return b
}

func TestSetShippingAddress_AuthorizedUser(t *testing.T) {
	repo := new(mockCheckoutRepository)
	handler := testHandler(repo)
	router := setupRouter(handler)

	session := activeSession()
	// First call: authorization lookup. Second call: the actual SetShippingAddress service call.
	repo.On("GetByID", mock.Anything, "session-123").Return(session, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.CheckoutSession")).Return(nil)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/checkout/session-123/shipping", bytes.NewReader(validShippingJSON()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-456")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	repo.AssertExpectations(t)
}

func TestSetShippingAddress_MissingUserIDHeader(t *testing.T) {
	repo := new(mockCheckoutRepository)
	handler := testHandler(repo)
	router := setupRouter(handler)

	session := activeSession()
	repo.On("GetByID", mock.Anything, "session-123").Return(session, nil)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/checkout/session-123/shipping", bytes.NewReader(validShippingJSON()))
	req.Header.Set("Content-Type", "application/json")
	// No X-User-ID header.
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	repo.AssertExpectations(t)
}

func TestSetShippingAddress_ForbiddenDifferentUser(t *testing.T) {
	repo := new(mockCheckoutRepository)
	handler := testHandler(repo)
	router := setupRouter(handler)

	session := activeSession()
	repo.On("GetByID", mock.Anything, "session-123").Return(session, nil)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/checkout/session-123/shipping", bytes.NewReader(validShippingJSON()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "other-user-789")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "FORBIDDEN", resp.Error.Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// SetPaymentMethod handler authorization tests
// ============================================================================

func validPaymentJSON() []byte {
	body := SetPaymentMethodRequest{
		PaymentMethod: "credit_card",
	}
	b, _ := json.Marshal(body)
	return b
}

func TestSetPaymentMethod_AuthorizedUser(t *testing.T) {
	repo := new(mockCheckoutRepository)
	handler := testHandler(repo)
	router := setupRouter(handler)

	session := activeSession()
	repo.On("GetByID", mock.Anything, "session-123").Return(session, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.CheckoutSession")).Return(nil)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/checkout/session-123/payment", bytes.NewReader(validPaymentJSON()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-456")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	repo.AssertExpectations(t)
}

func TestSetPaymentMethod_MissingUserIDHeader(t *testing.T) {
	repo := new(mockCheckoutRepository)
	handler := testHandler(repo)
	router := setupRouter(handler)

	session := activeSession()
	repo.On("GetByID", mock.Anything, "session-123").Return(session, nil)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/checkout/session-123/payment", bytes.NewReader(validPaymentJSON()))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	repo.AssertExpectations(t)
}

func TestSetPaymentMethod_ForbiddenDifferentUser(t *testing.T) {
	repo := new(mockCheckoutRepository)
	handler := testHandler(repo)
	router := setupRouter(handler)

	session := activeSession()
	repo.On("GetByID", mock.Anything, "session-123").Return(session, nil)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/checkout/session-123/payment", bytes.NewReader(validPaymentJSON()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "other-user-789")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "FORBIDDEN", resp.Error.Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// ProcessCheckout handler authorization tests
// ============================================================================

func TestProcessCheckout_AuthorizedUser(t *testing.T) {
	repo := new(mockCheckoutRepository)
	handler := testHandler(repo)
	router := setupRouter(handler)

	session := activeSession()
	session.ShippingAddress = &domain.Address{
		FullName:    "John Doe",
		AddressLine: "123 Main St",
		City:        "New York",
		State:       "NY",
		PostalCode:  "10001",
		Country:     "US",
	}
	session.PaymentMethod = "credit_card"
	repo.On("GetByID", mock.Anything, "session-123").Return(session, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.CheckoutSession")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/session-123/process", nil)
	req.Header.Set("X-User-ID", "user-456")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	repo.AssertExpectations(t)
}

func TestProcessCheckout_MissingUserIDHeader(t *testing.T) {
	repo := new(mockCheckoutRepository)
	handler := testHandler(repo)
	router := setupRouter(handler)

	session := activeSession()
	repo.On("GetByID", mock.Anything, "session-123").Return(session, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/session-123/process", nil)
	// No X-User-ID header.
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	repo.AssertExpectations(t)
}

func TestProcessCheckout_ForbiddenDifferentUser(t *testing.T) {
	repo := new(mockCheckoutRepository)
	handler := testHandler(repo)
	router := setupRouter(handler)

	session := activeSession()
	repo.On("GetByID", mock.Anything, "session-123").Return(session, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/session-123/process", nil)
	req.Header.Set("X-User-ID", "attacker-999")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "FORBIDDEN", resp.Error.Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// CancelCheckout handler authorization tests
// ============================================================================

func TestCancelCheckout_AuthorizedUser(t *testing.T) {
	repo := new(mockCheckoutRepository)
	handler := testHandler(repo)
	router := setupRouter(handler)

	session := activeSession()
	repo.On("GetByID", mock.Anything, "session-123").Return(session, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.CheckoutSession")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/session-123/cancel", nil)
	req.Header.Set("X-User-ID", "user-456")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	repo.AssertExpectations(t)
}

func TestCancelCheckout_MissingUserIDHeader(t *testing.T) {
	repo := new(mockCheckoutRepository)
	handler := testHandler(repo)
	router := setupRouter(handler)

	session := activeSession()
	repo.On("GetByID", mock.Anything, "session-123").Return(session, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/session-123/cancel", nil)
	// No X-User-ID header.
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	repo.AssertExpectations(t)
}

func TestCancelCheckout_ForbiddenDifferentUser(t *testing.T) {
	repo := new(mockCheckoutRepository)
	handler := testHandler(repo)
	router := setupRouter(handler)

	session := activeSession()
	repo.On("GetByID", mock.Anything, "session-123").Return(session, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/session-123/cancel", nil)
	req.Header.Set("X-User-ID", "attacker-999")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "FORBIDDEN", resp.Error.Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// InitiateCheckout handler tests (pre-existing X-User-ID check; verify it still works)
// ============================================================================

func validInitiateJSON() []byte {
	body := InitiateCheckoutRequest{
		Items: []CheckoutItemRequest{
			{
				ProductID: "550e8400-e29b-41d4-a716-446655440001",
				VariantID: "550e8400-e29b-41d4-a716-446655440002",
				Name:      "Test Product",
				SKU:       "TST-001",
				Price:     2999,
				Quantity:  2,
			},
		},
		Currency: "USD",
	}
	b, _ := json.Marshal(body)
	return b
}

func TestInitiateCheckout_MissingUserIDHeader(t *testing.T) {
	repo := new(mockCheckoutRepository)
	handler := testHandler(repo)
	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/", bytes.NewReader(validInitiateJSON()))
	req.Header.Set("Content-Type", "application/json")
	// No X-User-ID header.
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "X-User-ID")
}

func TestInitiateCheckout_WithUserIDHeader(t *testing.T) {
	repo := new(mockCheckoutRepository)
	handler := testHandler(repo)
	router := setupRouter(handler)

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.CheckoutSession")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/", bytes.NewReader(validInitiateJSON()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user-456")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

// ============================================================================
// writeJSON helper test
// ============================================================================

func TestWriteJSON_SetsContentType(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusTeapot, response{
		Error: &errorResponse{Code: "TEST", Message: "teapot"},
	})

	assert.Equal(t, http.StatusTeapot, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "TEST", resp.Error.Code)
}

// ============================================================================
// Table-driven test: authorizeSession across multiple scenarios
// ============================================================================

func TestAuthorizeSession_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		headerUserID  string
		sessionUserID string
		expectAuth    bool
		expectHTTP    int
		expectErrCode string
	}{
		{
			name:          "exact match",
			headerUserID:  "user-1",
			sessionUserID: "user-1",
			expectAuth:    true,
			expectHTTP:    http.StatusOK, // default (no write)
		},
		{
			name:          "missing header",
			headerUserID:  "",
			sessionUserID: "user-1",
			expectAuth:    false,
			expectHTTP:    http.StatusBadRequest,
			expectErrCode: "INVALID_INPUT",
		},
		{
			name:          "user mismatch",
			headerUserID:  "user-2",
			sessionUserID: "user-1",
			expectAuth:    false,
			expectHTTP:    http.StatusForbidden,
			expectErrCode: "FORBIDDEN",
		},
		{
			name:          "empty session user id with non-empty header",
			headerUserID:  "user-1",
			sessionUserID: "",
			expectAuth:    false,
			expectHTTP:    http.StatusForbidden,
			expectErrCode: "FORBIDDEN",
		},
		{
			name:          "both empty",
			headerUserID:  "",
			sessionUserID: "",
			expectAuth:    false,
			expectHTTP:    http.StatusBadRequest,
			expectErrCode: "INVALID_INPUT",
		},
		{
			name:          "uuid-style ids match",
			headerUserID:  "550e8400-e29b-41d4-a716-446655440001",
			sessionUserID: "550e8400-e29b-41d4-a716-446655440001",
			expectAuth:    true,
			expectHTTP:    http.StatusOK,
		},
		{
			name:          "uuid-style ids mismatch",
			headerUserID:  "550e8400-e29b-41d4-a716-446655440001",
			sessionUserID: "550e8400-e29b-41d4-a716-446655440099",
			expectAuth:    false,
			expectHTTP:    http.StatusForbidden,
			expectErrCode: "FORBIDDEN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.headerUserID != "" {
				req.Header.Set("X-User-ID", tt.headerUserID)
			}
			rec := httptest.NewRecorder()
			session := &domain.CheckoutSession{UserID: tt.sessionUserID}

			result := authorizeSession(rec, req, session)

			assert.Equal(t, tt.expectAuth, result, "authorizeSession return value")
			assert.Equal(t, tt.expectHTTP, rec.Code, "HTTP status code")

			if !tt.expectAuth {
				resp := decodeResponse(t, rec)
				require.NotNil(t, resp.Error)
				assert.Equal(t, tt.expectErrCode, resp.Error.Code)
			}
		})
	}
}

// ============================================================================
// Table-driven test: all five authorized endpoints reject unauthorized access
// ============================================================================

func TestAllAuthorizedEndpoints_RejectMissingHeader(t *testing.T) {
	endpoints := []struct {
		method string
		path   string
		body   []byte
	}{
		{http.MethodGet, "/api/v1/checkout/session-123", nil},
		{http.MethodPut, "/api/v1/checkout/session-123/shipping", validShippingJSON()},
		{http.MethodPut, "/api/v1/checkout/session-123/payment", validPaymentJSON()},
		{http.MethodPost, "/api/v1/checkout/session-123/process", nil},
		{http.MethodPost, "/api/v1/checkout/session-123/cancel", nil},
	}

	for _, ep := range endpoints {
		name := fmt.Sprintf("%s %s", ep.method, ep.path)
		t.Run(name, func(t *testing.T) {
			repo := new(mockCheckoutRepository)
			handler := testHandler(repo)
			router := setupRouter(handler)

			session := activeSession()
			repo.On("GetByID", mock.Anything, "session-123").Return(session, nil)

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

			assert.Equal(t, http.StatusBadRequest, rec.Code, "expected 400 for missing X-User-ID")
			resp := decodeResponse(t, rec)
			require.NotNil(t, resp.Error)
			assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
		})
	}
}

func TestAllAuthorizedEndpoints_RejectWrongUser(t *testing.T) {
	endpoints := []struct {
		method string
		path   string
		body   []byte
	}{
		{http.MethodGet, "/api/v1/checkout/session-123", nil},
		{http.MethodPut, "/api/v1/checkout/session-123/shipping", validShippingJSON()},
		{http.MethodPut, "/api/v1/checkout/session-123/payment", validPaymentJSON()},
		{http.MethodPost, "/api/v1/checkout/session-123/process", nil},
		{http.MethodPost, "/api/v1/checkout/session-123/cancel", nil},
	}

	for _, ep := range endpoints {
		name := fmt.Sprintf("%s %s", ep.method, ep.path)
		t.Run(name, func(t *testing.T) {
			repo := new(mockCheckoutRepository)
			handler := testHandler(repo)
			router := setupRouter(handler)

			session := activeSession() // UserID = "user-456"
			repo.On("GetByID", mock.Anything, "session-123").Return(session, nil)

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
			req.Header.Set("X-User-ID", "wrong-user-999")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusForbidden, rec.Code, "expected 403 for wrong user")
			resp := decodeResponse(t, rec)
			require.NotNil(t, resp.Error)
			assert.Equal(t, "FORBIDDEN", resp.Error.Code)
		})
	}
}
