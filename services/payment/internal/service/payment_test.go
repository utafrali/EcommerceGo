package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/payment/internal/domain"
	"github.com/utafrali/EcommerceGo/services/payment/internal/provider"
)

// --- Mock Repository ---

type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) Create(ctx context.Context, payment *domain.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *mockRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *mockRepository) GetByCheckoutID(ctx context.Context, checkoutID string) (*domain.Payment, error) {
	args := m.Called(ctx, checkoutID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *mockRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Payment, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Payment), args.Error(1)
}

func (m *mockRepository) Update(ctx context.Context, payment *domain.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *mockRepository) ListByUserID(ctx context.Context, userID string, offset, limit int) ([]domain.Payment, int, error) {
	args := m.Called(ctx, userID, offset, limit)
	return args.Get(0).([]domain.Payment), args.Int(1), args.Error(2)
}

func (m *mockRepository) CreateRefund(ctx context.Context, refund *domain.Refund) error {
	args := m.Called(ctx, refund)
	return args.Error(0)
}

func (m *mockRepository) GetRefundByID(ctx context.Context, id string) (*domain.Refund, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Refund), args.Error(1)
}

func (m *mockRepository) ListRefundsByPaymentID(ctx context.Context, paymentID string) ([]domain.Refund, error) {
	args := m.Called(ctx, paymentID)
	return args.Get(0).([]domain.Refund), args.Error(1)
}

func (m *mockRepository) UpdateRefund(ctx context.Context, refund *domain.Refund) error {
	args := m.Called(ctx, refund)
	return args.Error(0)
}

// --- Mock Provider ---

type mockProvider struct {
	mock.Mock
}

func (m *mockProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockProvider) Charge(ctx context.Context, input *provider.ChargeInput) (*provider.ChargeResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.ChargeResult), args.Error(1)
}

func (m *mockProvider) Refund(ctx context.Context, input *provider.RefundInput) (*provider.RefundResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.RefundResult), args.Error(1)
}

// --- Test Helpers ---

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestService(repo *mockRepository, prov *mockProvider) *PaymentService {
	// The service's producer is nil since we do not have a real Kafka producer in tests.
	// We use a custom service that skips event publishing.
	return &PaymentService{
		repo:     repo,
		provider: prov,
		producer: nil,
		logger:   newTestLogger(),
	}
}

func newTestPayment() *domain.Payment {
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

func newSucceededPayment() *domain.Payment {
	p := newTestPayment()
	p.Status = domain.PaymentStatusSucceeded
	p.ProviderPayID = "mock_pay_" + uuid.New().String()
	return p
}

// --- Tests ---

func TestCreatePayment_Success(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	prov.On("Name").Return("mock")
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Payment")).Return(nil)

	input := &CreatePaymentInput{
		CheckoutID: uuid.New().String(),
		OrderID:    uuid.New().String(),
		UserID:     uuid.New().String(),
		Amount:     5000,
		Currency:   "usd",
		Method:     domain.PaymentMethodCreditCard,
	}

	payment, err := svc.CreatePayment(context.Background(), input)

	require.NoError(t, err)
	assert.NotEmpty(t, payment.ID)
	assert.Equal(t, domain.PaymentStatusPending, payment.Status)
	assert.Equal(t, "USD", payment.Currency)
	assert.Equal(t, int64(5000), payment.Amount)
	assert.Equal(t, "mock", payment.ProviderName)
	repo.AssertExpectations(t)
}

func TestCreatePayment_InvalidAmount(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	prov.On("Name").Return("mock")

	input := &CreatePaymentInput{
		CheckoutID: uuid.New().String(),
		OrderID:    uuid.New().String(),
		UserID:     uuid.New().String(),
		Amount:     0,
		Currency:   "USD",
		Method:     domain.PaymentMethodCreditCard,
	}

	payment, err := svc.CreatePayment(context.Background(), input)

	assert.Nil(t, payment)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrInvalidInput))
}

func TestCreatePayment_InvalidCurrency(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	prov.On("Name").Return("mock")

	input := &CreatePaymentInput{
		CheckoutID: uuid.New().String(),
		OrderID:    uuid.New().String(),
		UserID:     uuid.New().String(),
		Amount:     5000,
		Currency:   "US",
		Method:     domain.PaymentMethodCreditCard,
	}

	payment, err := svc.CreatePayment(context.Background(), input)

	assert.Nil(t, payment)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrInvalidInput))
}

func TestCreatePayment_InvalidMethod(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	prov.On("Name").Return("mock")

	input := &CreatePaymentInput{
		CheckoutID: uuid.New().String(),
		OrderID:    uuid.New().String(),
		UserID:     uuid.New().String(),
		Amount:     5000,
		Currency:   "USD",
		Method:     "bitcoin",
	}

	payment, err := svc.CreatePayment(context.Background(), input)

	assert.Nil(t, payment)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrInvalidInput))
}

func TestCreatePayment_EmptyCheckoutID(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	prov.On("Name").Return("mock")

	input := &CreatePaymentInput{
		CheckoutID: "",
		OrderID:    uuid.New().String(),
		UserID:     uuid.New().String(),
		Amount:     5000,
		Currency:   "USD",
		Method:     domain.PaymentMethodCreditCard,
	}

	payment, err := svc.CreatePayment(context.Background(), input)

	assert.Nil(t, payment)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrInvalidInput))
}

func TestCreatePayment_EmptyOrderID(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	prov.On("Name").Return("mock")

	input := &CreatePaymentInput{
		CheckoutID: uuid.New().String(),
		OrderID:    "",
		UserID:     uuid.New().String(),
		Amount:     5000,
		Currency:   "USD",
		Method:     domain.PaymentMethodCreditCard,
	}

	payment, err := svc.CreatePayment(context.Background(), input)

	assert.Nil(t, payment)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrInvalidInput))
}

func TestCreatePayment_EmptyUserID(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	prov.On("Name").Return("mock")

	input := &CreatePaymentInput{
		CheckoutID: uuid.New().String(),
		OrderID:    uuid.New().String(),
		UserID:     "",
		Amount:     5000,
		Currency:   "USD",
		Method:     domain.PaymentMethodCreditCard,
	}

	payment, err := svc.CreatePayment(context.Background(), input)

	assert.Nil(t, payment)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrInvalidInput))
}

func TestCreatePayment_RepositoryError(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	prov.On("Name").Return("mock")
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Payment")).Return(errors.New("db error"))

	input := &CreatePaymentInput{
		CheckoutID: uuid.New().String(),
		OrderID:    uuid.New().String(),
		UserID:     uuid.New().String(),
		Amount:     5000,
		Currency:   "USD",
		Method:     domain.PaymentMethodCreditCard,
	}

	payment, err := svc.CreatePayment(context.Background(), input)

	assert.Nil(t, payment)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create payment")
}

func TestProcessPayment_Success(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	payment := newTestPayment()

	repo.On("GetByID", mock.Anything, payment.ID).Return(payment, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Payment")).Return(nil)
	prov.On("Charge", mock.Anything, mock.AnythingOfType("*provider.ChargeInput")).Return(&provider.ChargeResult{
		ProviderPaymentID: "mock_pay_123",
		Status:            "succeeded",
	}, nil)

	result, err := svc.ProcessPayment(context.Background(), payment.ID)

	require.NoError(t, err)
	assert.Equal(t, domain.PaymentStatusSucceeded, result.Status)
	assert.Equal(t, "mock_pay_123", result.ProviderPayID)
	repo.AssertExpectations(t)
	prov.AssertExpectations(t)
}

func TestProcessPayment_ProviderFailure(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	payment := newTestPayment()

	repo.On("GetByID", mock.Anything, payment.ID).Return(payment, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Payment")).Return(nil)
	prov.On("Charge", mock.Anything, mock.AnythingOfType("*provider.ChargeInput")).Return(nil, errors.New("card declined"))

	result, err := svc.ProcessPayment(context.Background(), payment.ID)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrPaymentFailed))
}

func TestProcessPayment_ChargeReturnsFailed(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	payment := newTestPayment()

	repo.On("GetByID", mock.Anything, payment.ID).Return(payment, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Payment")).Return(nil)
	prov.On("Charge", mock.Anything, mock.AnythingOfType("*provider.ChargeInput")).Return(&provider.ChargeResult{
		ProviderPaymentID: "mock_pay_456",
		Status:            "failed",
		FailureReason:     "insufficient funds",
	}, nil)

	result, err := svc.ProcessPayment(context.Background(), payment.ID)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrPaymentFailed))
}

func TestProcessPayment_NotPending(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	payment := newSucceededPayment()

	repo.On("GetByID", mock.Anything, payment.ID).Return(payment, nil)

	result, err := svc.ProcessPayment(context.Background(), payment.ID)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrInvalidInput))
}

func TestProcessPayment_NotFound(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	repo.On("GetByID", mock.Anything, "nonexistent").Return(nil, apperrors.ErrNotFound)

	result, err := svc.ProcessPayment(context.Background(), "nonexistent")

	assert.Nil(t, result)
	require.Error(t, err)
}

func TestGetPayment_Success(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	payment := newTestPayment()

	repo.On("GetByID", mock.Anything, payment.ID).Return(payment, nil)

	result, err := svc.GetPayment(context.Background(), payment.ID)

	require.NoError(t, err)
	assert.Equal(t, payment.ID, result.ID)
	assert.Equal(t, payment.Amount, result.Amount)
}

func TestGetPayment_NotFound(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	repo.On("GetByID", mock.Anything, "missing").Return(nil, apperrors.ErrNotFound)

	result, err := svc.GetPayment(context.Background(), "missing")

	assert.Nil(t, result)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrNotFound))
}

func TestGetPaymentByCheckoutID_Success(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	payment := newTestPayment()

	repo.On("GetByCheckoutID", mock.Anything, payment.CheckoutID).Return(payment, nil)

	result, err := svc.GetPaymentByCheckoutID(context.Background(), payment.CheckoutID)

	require.NoError(t, err)
	assert.Equal(t, payment.CheckoutID, result.CheckoutID)
}

func TestRefundPayment_FullRefund(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	payment := newSucceededPayment()

	repo.On("GetByID", mock.Anything, payment.ID).Return(payment, nil)
	repo.On("ListRefundsByPaymentID", mock.Anything, payment.ID).Return([]domain.Refund{}, nil)
	repo.On("CreateRefund", mock.Anything, mock.AnythingOfType("*domain.Refund")).Return(nil)
	repo.On("UpdateRefund", mock.Anything, mock.AnythingOfType("*domain.Refund")).Return(nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Payment")).Return(nil)
	prov.On("Refund", mock.Anything, mock.AnythingOfType("*provider.RefundInput")).Return(&provider.RefundResult{
		ProviderRefundID: "mock_ref_123",
		Status:           "succeeded",
	}, nil)

	input := &RefundPaymentInput{
		Amount: payment.Amount,
		Reason: "customer request",
	}

	refund, err := svc.RefundPayment(context.Background(), payment.ID, input)

	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusSucceeded, refund.Status)
	assert.Equal(t, payment.Amount, refund.Amount)
	assert.Equal(t, "mock_ref_123", refund.ProviderRefID)

	// Verify payment status was updated to refunded.
	repo.AssertCalled(t, "Update", mock.Anything, mock.MatchedBy(func(p *domain.Payment) bool {
		return p.Status == domain.PaymentStatusRefunded
	}))
}

func TestRefundPayment_PartialRefund(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	payment := newSucceededPayment()
	payment.Amount = 10000

	repo.On("GetByID", mock.Anything, payment.ID).Return(payment, nil)
	repo.On("ListRefundsByPaymentID", mock.Anything, payment.ID).Return([]domain.Refund{}, nil)
	repo.On("CreateRefund", mock.Anything, mock.AnythingOfType("*domain.Refund")).Return(nil)
	repo.On("UpdateRefund", mock.Anything, mock.AnythingOfType("*domain.Refund")).Return(nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Payment")).Return(nil)
	prov.On("Refund", mock.Anything, mock.AnythingOfType("*provider.RefundInput")).Return(&provider.RefundResult{
		ProviderRefundID: "mock_ref_456",
		Status:           "succeeded",
	}, nil)

	input := &RefundPaymentInput{
		Amount: 3000,
		Reason: "partial refund for damaged item",
	}

	refund, err := svc.RefundPayment(context.Background(), payment.ID, input)

	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusSucceeded, refund.Status)
	assert.Equal(t, int64(3000), refund.Amount)

	// Verify payment status was updated to partially_refunded.
	repo.AssertCalled(t, "Update", mock.Anything, mock.MatchedBy(func(p *domain.Payment) bool {
		return p.Status == domain.PaymentStatusPartiallyRefunded
	}))
}

func TestRefundPayment_ExceedsAmount(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	payment := newSucceededPayment()
	payment.Amount = 5000

	existingRefund := domain.Refund{
		ID:        uuid.New().String(),
		PaymentID: payment.ID,
		Amount:    3000,
		Status:    domain.RefundStatusSucceeded,
	}

	repo.On("GetByID", mock.Anything, payment.ID).Return(payment, nil)
	repo.On("ListRefundsByPaymentID", mock.Anything, payment.ID).Return([]domain.Refund{existingRefund}, nil)

	input := &RefundPaymentInput{
		Amount: 3000, // 3000 existing + 3000 new = 6000 > 5000
		Reason: "another refund",
	}

	refund, err := svc.RefundPayment(context.Background(), payment.ID, input)

	assert.Nil(t, refund)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrInvalidInput))
}

func TestRefundPayment_InvalidAmount(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	input := &RefundPaymentInput{
		Amount: 0,
		Reason: "test refund",
	}

	refund, err := svc.RefundPayment(context.Background(), uuid.New().String(), input)

	assert.Nil(t, refund)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrInvalidInput))
}

func TestRefundPayment_InvalidReason(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	input := &RefundPaymentInput{
		Amount: 1000,
		Reason: "ab", // too short, min 3
	}

	refund, err := svc.RefundPayment(context.Background(), uuid.New().String(), input)

	assert.Nil(t, refund)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrInvalidInput))
}

func TestRefundPayment_PaymentNotSucceeded(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	payment := newTestPayment()
	payment.Status = domain.PaymentStatusPending

	repo.On("GetByID", mock.Anything, payment.ID).Return(payment, nil)

	input := &RefundPaymentInput{
		Amount: 1000,
		Reason: "refund attempt on pending payment",
	}

	refund, err := svc.RefundPayment(context.Background(), payment.ID, input)

	assert.Nil(t, refund)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrInvalidInput))
}

func TestRefundPayment_ProviderError(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	payment := newSucceededPayment()

	repo.On("GetByID", mock.Anything, payment.ID).Return(payment, nil)
	repo.On("ListRefundsByPaymentID", mock.Anything, payment.ID).Return([]domain.Refund{}, nil)
	repo.On("CreateRefund", mock.Anything, mock.AnythingOfType("*domain.Refund")).Return(nil)
	repo.On("UpdateRefund", mock.Anything, mock.AnythingOfType("*domain.Refund")).Return(nil)
	prov.On("Refund", mock.Anything, mock.AnythingOfType("*provider.RefundInput")).Return(nil, errors.New("provider unavailable"))

	input := &RefundPaymentInput{
		Amount: 1000,
		Reason: "customer request",
	}

	refund, err := svc.RefundPayment(context.Background(), payment.ID, input)

	assert.Nil(t, refund)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider refund")
}

func TestListPaymentsByUser_Success(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	userID := uuid.New().String()
	payments := []domain.Payment{*newTestPayment(), *newTestPayment()}
	payments[0].UserID = userID
	payments[1].UserID = userID

	repo.On("ListByUserID", mock.Anything, userID, 0, 20).Return(payments, 2, nil)

	result, total, err := svc.ListPaymentsByUser(context.Background(), userID, 1, 20)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, total)
}

func TestListPaymentsByUser_Pagination(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	userID := uuid.New().String()

	repo.On("ListByUserID", mock.Anything, userID, 10, 10).Return([]domain.Payment{}, 25, nil)

	result, total, err := svc.ListPaymentsByUser(context.Background(), userID, 2, 10)

	require.NoError(t, err)
	assert.Len(t, result, 0)
	assert.Equal(t, 25, total)
}

func TestListPaymentsByUser_DefaultPagination(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	userID := uuid.New().String()

	repo.On("ListByUserID", mock.Anything, userID, 0, 20).Return([]domain.Payment{}, 0, nil)

	result, total, err := svc.ListPaymentsByUser(context.Background(), userID, 0, 0)

	require.NoError(t, err)
	assert.Len(t, result, 0)
	assert.Equal(t, 0, total)
}

func TestListPaymentsByUser_MaxPerPage(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	userID := uuid.New().String()

	// When perPage > 100, it should be clamped to 100.
	repo.On("ListByUserID", mock.Anything, userID, 0, 100).Return([]domain.Payment{}, 0, nil)

	result, total, err := svc.ListPaymentsByUser(context.Background(), userID, 1, 200)

	require.NoError(t, err)
	assert.Len(t, result, 0)
	assert.Equal(t, 0, total)
}

func TestRefundPayment_AlreadyPartiallyRefunded_CanRefundMore(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	payment := newSucceededPayment()
	payment.Amount = 10000
	payment.Status = domain.PaymentStatusPartiallyRefunded

	existingRefund := domain.Refund{
		ID:        uuid.New().String(),
		PaymentID: payment.ID,
		Amount:    3000,
		Status:    domain.RefundStatusSucceeded,
	}

	repo.On("GetByID", mock.Anything, payment.ID).Return(payment, nil)
	repo.On("ListRefundsByPaymentID", mock.Anything, payment.ID).Return([]domain.Refund{existingRefund}, nil)
	repo.On("CreateRefund", mock.Anything, mock.AnythingOfType("*domain.Refund")).Return(nil)
	repo.On("UpdateRefund", mock.Anything, mock.AnythingOfType("*domain.Refund")).Return(nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Payment")).Return(nil)
	prov.On("Refund", mock.Anything, mock.AnythingOfType("*provider.RefundInput")).Return(&provider.RefundResult{
		ProviderRefundID: "mock_ref_789",
		Status:           "succeeded",
	}, nil)

	input := &RefundPaymentInput{
		Amount: 7000, // 3000 existing + 7000 = 10000, equals payment.Amount
		Reason: "remaining refund",
	}

	refund, err := svc.RefundPayment(context.Background(), payment.ID, input)

	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusSucceeded, refund.Status)
	assert.Equal(t, int64(7000), refund.Amount)

	// Payment should now be fully refunded.
	repo.AssertCalled(t, "Update", mock.Anything, mock.MatchedBy(func(p *domain.Payment) bool {
		return p.Status == domain.PaymentStatusRefunded
	}))
}

func TestCreatePayment_NegativeAmount(t *testing.T) {
	repo := new(mockRepository)
	prov := new(mockProvider)
	svc := newTestService(repo, prov)

	prov.On("Name").Return("mock")

	input := &CreatePaymentInput{
		CheckoutID: uuid.New().String(),
		OrderID:    uuid.New().String(),
		UserID:     uuid.New().String(),
		Amount:     -100,
		Currency:   "USD",
		Method:     domain.PaymentMethodCreditCard,
	}

	payment, err := svc.CreatePayment(context.Background(), input)

	assert.Nil(t, payment)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrInvalidInput))
}
