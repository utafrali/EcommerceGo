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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/pkg/httputil"
	"github.com/utafrali/EcommerceGo/services/payment/internal/domain"
	"github.com/utafrali/EcommerceGo/services/payment/internal/provider"
	"github.com/utafrali/EcommerceGo/services/payment/internal/service"
)

// listResponse mirrors httputil.PaginatedResponse for test decoding.
type listResponse = httputil.PaginatedResponse[domain.Payment]

// --- Mock Repository ---

type mockPaymentRepository struct {
	mock.Mock
}

func (m *mockPaymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *mockPaymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *mockPaymentRepository) GetByCheckoutID(ctx context.Context, checkoutID string) (*domain.Payment, error) {
	args := m.Called(ctx, checkoutID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *mockPaymentRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Payment, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *mockPaymentRepository) Update(ctx context.Context, payment *domain.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *mockPaymentRepository) ListByUserID(ctx context.Context, userID string, offset, limit int) ([]domain.Payment, int, error) {
	args := m.Called(ctx, userID, offset, limit)
	return args.Get(0).([]domain.Payment), args.Int(1), args.Error(2)
}

func (m *mockPaymentRepository) CreateRefund(ctx context.Context, refund *domain.Refund) error {
	args := m.Called(ctx, refund)
	return args.Error(0)
}

func (m *mockPaymentRepository) GetRefundByID(ctx context.Context, id string) (*domain.Refund, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Refund), args.Error(1)
}

func (m *mockPaymentRepository) ListRefundsByPaymentID(ctx context.Context, paymentID string) ([]domain.Refund, error) {
	args := m.Called(ctx, paymentID)
	return args.Get(0).([]domain.Refund), args.Error(1)
}

func (m *mockPaymentRepository) UpdateRefund(ctx context.Context, refund *domain.Refund) error {
	args := m.Called(ctx, refund)
	return args.Error(0)
}

// --- Mock Provider ---

type mockPaymentProvider struct {
	mock.Mock
}

func (m *mockPaymentProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockPaymentProvider) Charge(ctx context.Context, input *provider.ChargeInput) (*provider.ChargeResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.ChargeResult), args.Error(1)
}

func (m *mockPaymentProvider) Refund(ctx context.Context, input *provider.RefundInput) (*provider.RefundResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.RefundResult), args.Error(1)
}

// --- Test Helpers ---

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// newTestPaymentService creates a PaymentService backed by mock repo and provider, with no Kafka producer.
func newTestPaymentService(repo *mockPaymentRepository, prov *mockPaymentProvider) *service.PaymentService {
	return service.NewPaymentService(repo, prov, nil, testLogger())
}

// newTestPaymentHandler creates a PaymentHandler backed by mock repo and provider.
func newTestPaymentHandler(repo *mockPaymentRepository, prov *mockPaymentProvider) *PaymentHandler {
	svc := newTestPaymentService(repo, prov)
	return NewPaymentHandler(svc, testLogger())
}

// setupPaymentRouter creates a chi router matching the production route layout.
func setupPaymentRouter(handler *PaymentHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1/payments", func(r chi.Router) {
		r.Use(ContentTypeJSON)
		r.Post("/", handler.CreatePayment)
		r.Get("/{id}", handler.GetPayment)
		r.Post("/{id}/process", handler.ProcessPayment)
		r.Post("/{id}/refund", handler.RefundPayment)
		r.Get("/checkout/{checkoutId}", handler.GetPaymentByCheckoutID)
		r.Get("/user/{userId}", handler.ListPaymentsByUser)
	})
	return r
}

// decodeResp reads the response body into an httputil.Response.
func decodeResp(t *testing.T, rec *httptest.ResponseRecorder) httputil.Response {
	t.Helper()
	var resp httputil.Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	return resp
}

// samplePayment returns a Payment with all fields set.
func samplePayment() *domain.Payment {
	now := time.Now().UTC()
	return &domain.Payment{
		ID:           uuid.New().String(),
		CheckoutID:   uuid.New().String(),
		OrderID:      uuid.New().String(),
		UserID:       uuid.New().String(),
		Amount:       5000,
		Currency:     "USD",
		Status:       domain.PaymentStatusPending,
		Method:       domain.PaymentMethodCreditCard,
		ProviderName: "mock",
		Metadata:     make(map[string]any),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// succeededPayment returns a Payment in the succeeded state.
func succeededPayment() *domain.Payment {
	p := samplePayment()
	p.Status = domain.PaymentStatusSucceeded
	p.ProviderPayID = "mock_pay_" + uuid.New().String()
	return p
}

// validCreateJSON returns a valid JSON body for CreatePayment.
func validCreateJSON() []byte {
	body := CreatePaymentRequest{
		CheckoutID: uuid.New().String(),
		OrderID:    uuid.New().String(),
		UserID:     uuid.New().String(),
		Amount:     5000,
		Currency:   "USD",
		Method:     "credit_card",
	}
	b, _ := json.Marshal(body)
	return b
}

// validRefundJSON returns a valid JSON body for RefundPayment.
func validRefundJSON() []byte {
	body := RefundPaymentRequest{
		Amount: 2000,
		Reason: "customer request",
	}
	b, _ := json.Marshal(body)
	return b
}

// ============================================================================
// POST /api/v1/payments - CreatePayment
// ============================================================================

func TestCreatePayment_Success(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	prov.On("Name").Return("mock")
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Payment")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader(validCreateJSON()))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	resp := decodeResp(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestCreatePayment_InvalidJSON(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader([]byte(`{invalid json`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestCreatePayment_ValidationError_MissingFields(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	// Empty body: all required fields missing.
	body, _ := json.Marshal(CreatePaymentRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestCreatePayment_ValidationError_InvalidUUID(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	body, _ := json.Marshal(CreatePaymentRequest{
		CheckoutID: "not-a-uuid",
		OrderID:    uuid.New().String(),
		UserID:     uuid.New().String(),
		Amount:     5000,
		Currency:   "USD",
		Method:     "credit_card",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestCreatePayment_ValidationError_ZeroAmount(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	body, _ := json.Marshal(CreatePaymentRequest{
		CheckoutID: uuid.New().String(),
		OrderID:    uuid.New().String(),
		UserID:     uuid.New().String(),
		Amount:     0,
		Currency:   "USD",
		Method:     "credit_card",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestCreatePayment_ServiceError(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	prov.On("Name").Return("mock")
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Payment")).Return(fmt.Errorf("db connection lost"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader(validCreateJSON()))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INTERNAL_ERROR", resp.Error.Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// GET /api/v1/payments/{id} - GetPayment
// ============================================================================

func TestGetPayment_Success(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	payment := samplePayment()
	repo.On("GetByID", mock.Anything, payment.ID).Return(payment, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/"+payment.ID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResp(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestGetPayment_InvalidUUID(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/not-a-valid-uuid", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid UUID")
}

func TestGetPayment_NotFound(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	missingID := uuid.New().String()
	repo.On("GetByID", mock.Anything, missingID).Return(nil, apperrors.NotFound("payment", missingID))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/"+missingID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// POST /api/v1/payments/{id}/process - ProcessPayment
// ============================================================================

func TestProcessPayment_Success(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	payment := samplePayment()
	repo.On("GetByID", mock.Anything, payment.ID).Return(payment, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Payment")).Return(nil)
	prov.On("Charge", mock.Anything, mock.AnythingOfType("*provider.ChargeInput")).Return(&provider.ChargeResult{
		ProviderPaymentID: "mock_pay_123",
		Status:            "succeeded",
	}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/"+payment.ID+"/process", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResp(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
	prov.AssertExpectations(t)
}

func TestProcessPayment_InvalidUUID(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/bad-id/process", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

func TestProcessPayment_ServiceError_NotFound(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	missingID := uuid.New().String()
	repo.On("GetByID", mock.Anything, missingID).Return(nil, apperrors.NotFound("payment", missingID))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/"+missingID+"/process", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	repo.AssertExpectations(t)
}

func TestProcessPayment_ServiceError_ProviderFailure(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	payment := samplePayment()
	repo.On("GetByID", mock.Anything, payment.ID).Return(payment, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Payment")).Return(nil)
	prov.On("Charge", mock.Anything, mock.AnythingOfType("*provider.ChargeInput")).Return(nil, fmt.Errorf("card declined"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/"+payment.ID+"/process", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "PAYMENT_FAILED", resp.Error.Code)
	repo.AssertExpectations(t)
	prov.AssertExpectations(t)
}

// ============================================================================
// POST /api/v1/payments/{id}/refund - RefundPayment
// ============================================================================

func TestRefundPayment_Success(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	payment := succeededPayment()
	repo.On("GetByID", mock.Anything, payment.ID).Return(payment, nil)
	repo.On("ListRefundsByPaymentID", mock.Anything, payment.ID).Return([]domain.Refund{}, nil)
	repo.On("CreateRefund", mock.Anything, mock.AnythingOfType("*domain.Refund")).Return(nil)
	repo.On("UpdateRefund", mock.Anything, mock.AnythingOfType("*domain.Refund")).Return(nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Payment")).Return(nil)
	prov.On("Refund", mock.Anything, mock.AnythingOfType("*provider.RefundInput")).Return(&provider.RefundResult{
		ProviderRefundID: "mock_ref_001",
		Status:           "succeeded",
	}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/"+payment.ID+"/refund", bytes.NewReader(validRefundJSON()))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResp(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
	prov.AssertExpectations(t)
}

func TestRefundPayment_InvalidJSON(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	paymentID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/"+paymentID+"/refund", bytes.NewReader([]byte(`{bad json`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestRefundPayment_InvalidUUID(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/not-uuid/refund", bytes.NewReader(validRefundJSON()))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

func TestRefundPayment_ValidationError_ZeroAmount(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	paymentID := uuid.New().String()
	body, _ := json.Marshal(RefundPaymentRequest{
		Amount: 0,
		Reason: "valid reason",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/"+paymentID+"/refund", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestRefundPayment_ValidationError_ShortReason(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	paymentID := uuid.New().String()
	body, _ := json.Marshal(RefundPaymentRequest{
		Amount: 1000,
		Reason: "ab", // too short, min 3
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/"+paymentID+"/refund", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestRefundPayment_ServiceError(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	payment := succeededPayment()
	repo.On("GetByID", mock.Anything, payment.ID).Return(payment, nil)
	repo.On("ListRefundsByPaymentID", mock.Anything, payment.ID).Return([]domain.Refund{}, nil)
	repo.On("CreateRefund", mock.Anything, mock.AnythingOfType("*domain.Refund")).Return(nil)
	repo.On("UpdateRefund", mock.Anything, mock.AnythingOfType("*domain.Refund")).Return(nil)
	prov.On("Refund", mock.Anything, mock.AnythingOfType("*provider.RefundInput")).Return(nil, fmt.Errorf("provider unavailable"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/"+payment.ID+"/refund", bytes.NewReader(validRefundJSON()))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// provider refund error is wrapped, so it becomes an internal error.
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	repo.AssertExpectations(t)
	prov.AssertExpectations(t)
}

// ============================================================================
// GET /api/v1/payments/checkout/{checkoutId} - GetPaymentByCheckoutID
// ============================================================================

func TestGetPaymentByCheckoutID_Success(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	payment := samplePayment()
	repo.On("GetByCheckoutID", mock.Anything, payment.CheckoutID).Return(payment, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/checkout/"+payment.CheckoutID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResp(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestGetPaymentByCheckoutID_InvalidUUID(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/checkout/bad-checkout-id", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid UUID")
}

func TestGetPaymentByCheckoutID_NotFound(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	missingID := uuid.New().String()
	repo.On("GetByCheckoutID", mock.Anything, missingID).Return(nil, apperrors.NotFound("payment", missingID))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/checkout/"+missingID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// GET /api/v1/payments/user/{userId} - ListPaymentsByUser
// ============================================================================

func TestListPaymentsByUser_Success(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	userID := uuid.New().String()
	payments := []domain.Payment{*samplePayment(), *samplePayment()}
	payments[0].UserID = userID
	payments[1].UserID = userID

	repo.On("ListByUserID", mock.Anything, userID, 0, 20).Return(payments, 2, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/user/"+userID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Decode into listResponse (the handler uses its own envelope, not httputil.Response).
	var resp listResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, 2, resp.TotalCount)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.PerPage)
	assert.Equal(t, 1, resp.TotalPages)
	repo.AssertExpectations(t)
}

func TestListPaymentsByUser_WithPagination(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	userID := uuid.New().String()
	payments := []domain.Payment{*samplePayment()}

	// page=2, per_page=5 => offset=5, limit=5
	repo.On("ListByUserID", mock.Anything, userID, 5, 5).Return(payments, 12, nil)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/payments/user/%s?page=2&per_page=5", userID), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp listResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, 12, resp.TotalCount)
	assert.Equal(t, 2, resp.Page)
	assert.Equal(t, 5, resp.PerPage)
	assert.Equal(t, 3, resp.TotalPages) // ceil(12/5) = 3
	repo.AssertExpectations(t)
}

func TestListPaymentsByUser_InvalidUUID(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/user/invalid-user-id", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid UUID")
}

func TestListPaymentsByUser_InvalidPage(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	userID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/payments/user/%s?page=-1", userID), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "page")
}

func TestListPaymentsByUser_InvalidPerPage(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	userID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/payments/user/%s?per_page=999", userID), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "per_page")
}

func TestListPaymentsByUser_EmptyResults(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	userID := uuid.New().String()
	repo.On("ListByUserID", mock.Anything, userID, 0, 20).Return([]domain.Payment{}, 0, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/user/"+userID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp listResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.TotalCount)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.PerPage)
	repo.AssertExpectations(t)
}

// ============================================================================
// ContentTypeJSON middleware tests
// ============================================================================

func TestContentTypeJSON_RejectsXML(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader([]byte(`<xml/>`)))
	req.Header.Set("Content-Type", "text/xml")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
}

func TestCreatePayment_ValidationError_InvalidMethod(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	body, _ := json.Marshal(CreatePaymentRequest{
		CheckoutID: uuid.New().String(),
		OrderID:    uuid.New().String(),
		UserID:     uuid.New().String(),
		Amount:     5000,
		Currency:   "USD",
		Method:     "bitcoin", // invalid method
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestCreatePayment_ValidationError_InvalidCurrencyLength(t *testing.T) {
	repo := new(mockPaymentRepository)
	prov := new(mockPaymentProvider)
	handler := newTestPaymentHandler(repo, prov)
	router := setupPaymentRouter(handler)

	body, _ := json.Marshal(CreatePaymentRequest{
		CheckoutID: uuid.New().String(),
		OrderID:    uuid.New().String(),
		UserID:     uuid.New().String(),
		Amount:     5000,
		Currency:   "US", // must be len=3
		Method:     "credit_card",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResp(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}
