package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/checkout/internal/domain"
	"github.com/utafrali/EcommerceGo/services/checkout/internal/event"
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

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestEventProducer() *event.Producer {
	logger := newTestLogger()
	kafkaCfg := pkgkafka.DefaultProducerConfig([]string{"localhost:9092"})
	kafkaProducer := pkgkafka.NewProducer(kafkaCfg, logger)
	return event.NewProducer(kafkaProducer, logger)
}

func newTestService(repo *mockCheckoutRepository) *CheckoutService {
	logger := newTestLogger()
	producer := newTestEventProducer()
	return NewCheckoutService(repo, producer, logger)
}

func validCheckoutInput() *InitiateCheckoutInput {
	return &InitiateCheckoutInput{
		Items: []CheckoutItemInput{
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
}

func validAddress() *domain.Address {
	return &domain.Address{
		FullName:    "John Doe",
		AddressLine: "123 Main St",
		City:        "New York",
		State:       "NY",
		PostalCode:  "10001",
		Country:     "US",
		Phone:       "+1234567890",
	}
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

// --- InitiateCheckout Tests ---

func TestInitiateCheckout_Success(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.CheckoutSession")).Return(nil)

	session, err := svc.InitiateCheckout(ctx, "user-456", validCheckoutInput())

	require.NoError(t, err)
	assert.NotNil(t, session)
	assert.NotEmpty(t, session.ID)
	assert.Equal(t, "user-456", session.UserID)
	assert.Equal(t, domain.StatusInitiated, session.Status)
	assert.Equal(t, "USD", session.Currency)
	assert.Len(t, session.Items, 1)
	assert.Equal(t, int64(5998), session.SubtotalAmount)
	assert.Equal(t, int64(5998), session.TotalAmount)
	assert.Equal(t, int64(0), session.DiscountAmount)
	assert.Equal(t, int64(0), session.ShippingAmount)
	assert.False(t, session.ExpiresAt.IsZero())
	assert.NotZero(t, session.CreatedAt)
	assert.NotZero(t, session.UpdatedAt)

	repo.AssertExpectations(t)
}

func TestInitiateCheckout_MultipleItems(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.CheckoutSession")).Return(nil)

	input := &InitiateCheckoutInput{
		Items: []CheckoutItemInput{
			{
				ProductID: "550e8400-e29b-41d4-a716-446655440001",
				VariantID: "550e8400-e29b-41d4-a716-446655440002",
				Name:      "Product A",
				SKU:       "SKU-A",
				Price:     1000,
				Quantity:  3,
			},
			{
				ProductID: "550e8400-e29b-41d4-a716-446655440003",
				VariantID: "550e8400-e29b-41d4-a716-446655440004",
				Name:      "Product B",
				SKU:       "SKU-B",
				Price:     2500,
				Quantity:  1,
			},
		},
		Currency: "EUR",
	}

	session, err := svc.InitiateCheckout(ctx, "user-789", input)

	require.NoError(t, err)
	assert.Len(t, session.Items, 2)
	assert.Equal(t, int64(5500), session.SubtotalAmount) // 1000*3 + 2500*1
	assert.Equal(t, int64(5500), session.TotalAmount)
	assert.Equal(t, "EUR", session.Currency)

	repo.AssertExpectations(t)
}

func TestInitiateCheckout_EmptyUserID(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	session, err := svc.InitiateCheckout(ctx, "", validCheckoutInput())

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestInitiateCheckout_NilInput(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	session, err := svc.InitiateCheckout(ctx, "user-456", nil)

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestInitiateCheckout_EmptyItems(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := &InitiateCheckoutInput{
		Items:    []CheckoutItemInput{},
		Currency: "USD",
	}

	session, err := svc.InitiateCheckout(ctx, "user-456", input)

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestInitiateCheckout_InvalidCurrency(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := &InitiateCheckoutInput{
		Items: []CheckoutItemInput{
			{
				ProductID: "550e8400-e29b-41d4-a716-446655440001",
				VariantID: "550e8400-e29b-41d4-a716-446655440002",
				Name:      "Test",
				SKU:       "T-1",
				Price:     100,
				Quantity:  1,
			},
		},
		Currency: "USDX",
	}

	session, err := svc.InitiateCheckout(ctx, "user-456", input)

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestInitiateCheckout_InvalidItemPrice(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := &InitiateCheckoutInput{
		Items: []CheckoutItemInput{
			{
				ProductID: "550e8400-e29b-41d4-a716-446655440001",
				VariantID: "550e8400-e29b-41d4-a716-446655440002",
				Name:      "Test",
				SKU:       "T-1",
				Price:     0,
				Quantity:  1,
			},
		},
		Currency: "USD",
	}

	session, err := svc.InitiateCheckout(ctx, "user-456", input)

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestInitiateCheckout_InvalidItemQuantity(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := &InitiateCheckoutInput{
		Items: []CheckoutItemInput{
			{
				ProductID: "550e8400-e29b-41d4-a716-446655440001",
				VariantID: "550e8400-e29b-41d4-a716-446655440002",
				Name:      "Test",
				SKU:       "T-1",
				Price:     100,
				Quantity:  0,
			},
		},
		Currency: "USD",
	}

	session, err := svc.InitiateCheckout(ctx, "user-456", input)

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestInitiateCheckout_RepositoryError(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.CheckoutSession")).
		Return(fmt.Errorf("database connection lost"))

	session, err := svc.InitiateCheckout(ctx, "user-456", validCheckoutInput())

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create checkout session")

	repo.AssertExpectations(t)
}

// --- GetCheckout Tests ---

func TestGetCheckout_Success(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	expected := activeSession()
	repo.On("GetByID", ctx, "session-123").Return(expected, nil)

	session, err := svc.GetCheckout(ctx, "session-123")

	require.NoError(t, err)
	assert.Equal(t, expected.ID, session.ID)
	assert.Equal(t, expected.UserID, session.UserID)

	repo.AssertExpectations(t)
}

func TestGetCheckout_NotFound(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	session, err := svc.GetCheckout(ctx, "nonexistent")

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	repo.AssertExpectations(t)
}

// --- SetShippingAddress Tests ---

func TestSetShippingAddress_Success(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := activeSession()
	repo.On("GetByID", ctx, "session-123").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.CheckoutSession")).Return(nil)

	session, err := svc.SetShippingAddress(ctx, "session-123", validAddress())

	require.NoError(t, err)
	assert.NotNil(t, session.ShippingAddress)
	assert.Equal(t, "John Doe", session.ShippingAddress.FullName)
	assert.Equal(t, "123 Main St", session.ShippingAddress.AddressLine)
	assert.Equal(t, "New York", session.ShippingAddress.City)
	assert.Equal(t, "NY", session.ShippingAddress.State)
	assert.Equal(t, "10001", session.ShippingAddress.PostalCode)
	assert.Equal(t, "US", session.ShippingAddress.Country)

	repo.AssertExpectations(t)
}

func TestSetShippingAddress_NilAddress(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	session, err := svc.SetShippingAddress(ctx, "session-123", nil)

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestSetShippingAddress_MissingFullName(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	addr := validAddress()
	addr.FullName = ""

	session, err := svc.SetShippingAddress(ctx, "session-123", addr)

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestSetShippingAddress_TerminalSession(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := activeSession()
	existing.Status = domain.StatusCompleted
	repo.On("GetByID", ctx, "session-123").Return(existing, nil)

	session, err := svc.SetShippingAddress(ctx, "session-123", validAddress())

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestSetShippingAddress_ExpiredSession(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := activeSession()
	existing.ExpiresAt = time.Now().UTC().Add(-1 * time.Hour) // Already expired.
	repo.On("GetByID", ctx, "session-123").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.CheckoutSession")).Return(nil)

	session, err := svc.SetShippingAddress(ctx, "session-123", validAddress())

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestSetShippingAddress_ExpiredSession_UpdateFails(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := activeSession()
	existing.ExpiresAt = time.Now().UTC().Add(-1 * time.Hour) // Already expired.
	repo.On("GetByID", ctx, "session-123").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.CheckoutSession")).Return(fmt.Errorf("db write failed"))

	session, err := svc.SetShippingAddress(ctx, "session-123", validAddress())

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update expired checkout session")

	repo.AssertExpectations(t)
}

// --- SetPaymentMethod Tests ---

func TestSetPaymentMethod_Success(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := activeSession()
	repo.On("GetByID", ctx, "session-123").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.CheckoutSession")).Return(nil)

	session, err := svc.SetPaymentMethod(ctx, "session-123", "credit_card")

	require.NoError(t, err)
	assert.Equal(t, "credit_card", session.PaymentMethod)

	repo.AssertExpectations(t)
}

func TestSetPaymentMethod_EmptyMethod(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	session, err := svc.SetPaymentMethod(ctx, "session-123", "")

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestSetPaymentMethod_TerminalSession(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := activeSession()
	existing.Status = domain.StatusFailed
	repo.On("GetByID", ctx, "session-123").Return(existing, nil)

	session, err := svc.SetPaymentMethod(ctx, "session-123", "credit_card")

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestSetPaymentMethod_ExpiredSession_UpdateFails(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := activeSession()
	existing.ExpiresAt = time.Now().UTC().Add(-1 * time.Hour) // Already expired.
	repo.On("GetByID", ctx, "session-123").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.CheckoutSession")).Return(fmt.Errorf("db write failed"))

	session, err := svc.SetPaymentMethod(ctx, "session-123", "credit_card")

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update expired checkout session")

	repo.AssertExpectations(t)
}

// --- ProcessCheckout Tests ---

func TestProcessCheckout_Success(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := activeSession()
	existing.ShippingAddress = validAddress()
	existing.PaymentMethod = "credit_card"

	repo.On("GetByID", ctx, "session-123").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.CheckoutSession")).Return(nil)

	session, err := svc.ProcessCheckout(ctx, "session-123")

	require.NoError(t, err)
	assert.Equal(t, domain.StatusCompleted, session.Status)
	assert.NotEmpty(t, session.OrderID)
	assert.NotEmpty(t, session.PaymentID)
	// All items should have reservation IDs.
	for _, item := range session.Items {
		assert.NotEmpty(t, item.ReservationID)
	}

	repo.AssertExpectations(t)
}

func TestProcessCheckout_SubtotalRevalidation(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := activeSession()
	existing.ShippingAddress = validAddress()
	existing.PaymentMethod = "credit_card"
	// Tamper with the stored subtotal to simulate a mismatch.
	existing.SubtotalAmount = 9999
	existing.TotalAmount = 9999

	repo.On("GetByID", ctx, "session-123").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.CheckoutSession")).Return(nil)

	session, err := svc.ProcessCheckout(ctx, "session-123")

	require.NoError(t, err)
	// Subtotal should be recalculated from items: 2999 * 2 = 5998
	assert.Equal(t, int64(5998), session.SubtotalAmount)
	assert.Equal(t, int64(5998), session.TotalAmount)
	assert.Equal(t, domain.StatusCompleted, session.Status)

	repo.AssertExpectations(t)
}

func TestProcessCheckout_MissingShippingAddress(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := activeSession()
	existing.PaymentMethod = "credit_card"
	// ShippingAddress is nil.

	repo.On("GetByID", ctx, "session-123").Return(existing, nil)

	session, err := svc.ProcessCheckout(ctx, "session-123")

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestProcessCheckout_MissingPaymentMethod(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := activeSession()
	existing.ShippingAddress = validAddress()
	// PaymentMethod is empty.

	repo.On("GetByID", ctx, "session-123").Return(existing, nil)

	session, err := svc.ProcessCheckout(ctx, "session-123")

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestProcessCheckout_TerminalSession(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := activeSession()
	existing.Status = domain.StatusCompleted

	repo.On("GetByID", ctx, "session-123").Return(existing, nil)

	session, err := svc.ProcessCheckout(ctx, "session-123")

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestProcessCheckout_ExpiredSession(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := activeSession()
	existing.ShippingAddress = validAddress()
	existing.PaymentMethod = "credit_card"
	existing.ExpiresAt = time.Now().UTC().Add(-1 * time.Hour)

	repo.On("GetByID", ctx, "session-123").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.CheckoutSession")).Return(nil)

	session, err := svc.ProcessCheckout(ctx, "session-123")

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestProcessCheckout_ExpiredSession_UpdateFails(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := activeSession()
	existing.ShippingAddress = validAddress()
	existing.PaymentMethod = "credit_card"
	existing.ExpiresAt = time.Now().UTC().Add(-1 * time.Hour)

	repo.On("GetByID", ctx, "session-123").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.CheckoutSession")).Return(fmt.Errorf("db write failed"))

	session, err := svc.ProcessCheckout(ctx, "session-123")

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update expired checkout session")

	repo.AssertExpectations(t)
}

func TestProcessCheckout_NotFound(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	session, err := svc.ProcessCheckout(ctx, "nonexistent")

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	repo.AssertExpectations(t)
}

// --- CancelCheckout Tests ---

func TestCancelCheckout_Success(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := activeSession()
	existing.Items[0].ReservationID = "res-001"

	repo.On("GetByID", ctx, "session-123").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.CheckoutSession")).Return(nil)

	session, err := svc.CancelCheckout(ctx, "session-123")

	require.NoError(t, err)
	assert.Equal(t, domain.StatusFailed, session.Status)
	assert.Equal(t, "cancelled by user", session.FailureReason)
	// Reservation IDs should be cleared.
	for _, item := range session.Items {
		assert.Empty(t, item.ReservationID)
	}

	repo.AssertExpectations(t)
}

func TestCancelCheckout_AlreadyCompleted(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := activeSession()
	existing.Status = domain.StatusCompleted

	repo.On("GetByID", ctx, "session-123").Return(existing, nil)

	session, err := svc.CancelCheckout(ctx, "session-123")

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestCancelCheckout_AlreadyFailed(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := activeSession()
	existing.Status = domain.StatusFailed

	repo.On("GetByID", ctx, "session-123").Return(existing, nil)

	session, err := svc.CancelCheckout(ctx, "session-123")

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestCancelCheckout_NotFound(t *testing.T) {
	repo := new(mockCheckoutRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	session, err := svc.CancelCheckout(ctx, "nonexistent")

	assert.Nil(t, session)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	repo.AssertExpectations(t)
}

// --- Domain Model Tests ---

func TestCheckoutSession_CalculateSubtotal(t *testing.T) {
	session := &domain.CheckoutSession{
		Items: []domain.CheckoutItem{
			{Price: 1000, Quantity: 2},
			{Price: 500, Quantity: 3},
		},
	}

	assert.Equal(t, int64(3500), session.CalculateSubtotal())
}

func TestCheckoutSession_CalculateTotal(t *testing.T) {
	session := &domain.CheckoutSession{
		SubtotalAmount: 5000,
		DiscountAmount: 500,
		ShippingAmount: 300,
	}

	assert.Equal(t, int64(4800), session.CalculateTotal())
}

func TestCheckoutSession_IsExpired(t *testing.T) {
	expired := &domain.CheckoutSession{
		ExpiresAt: time.Now().UTC().Add(-1 * time.Hour),
	}
	assert.True(t, expired.IsExpired())

	active := &domain.CheckoutSession{
		ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
	}
	assert.False(t, active.IsExpired())
}

func TestCheckoutSession_IsTerminal(t *testing.T) {
	tests := []struct {
		status   string
		terminal bool
	}{
		{domain.StatusInitiated, false},
		{domain.StatusItemsReserved, false},
		{domain.StatusPaymentPending, false},
		{domain.StatusPaymentProcessing, false},
		{domain.StatusCompleted, true},
		{domain.StatusFailed, true},
		{domain.StatusExpired, true},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			session := &domain.CheckoutSession{Status: tt.status}
			assert.Equal(t, tt.terminal, session.IsTerminal())
		})
	}
}

func TestSagaStep_Lifecycle(t *testing.T) {
	step := domain.NewSagaStep(domain.SagaStepReserveInventory)
	assert.Equal(t, domain.SagaStepPending, step.Status)
	assert.Equal(t, domain.SagaStepReserveInventory, step.Name)

	step.Complete()
	assert.Equal(t, domain.SagaStepCompleted, step.Status)
	assert.NotZero(t, step.ExecutedAt)

	step2 := domain.NewSagaStep(domain.SagaStepCreateOrder)
	step2.Fail("order service unavailable")
	assert.Equal(t, domain.SagaStepFailed, step2.Status)
	assert.Equal(t, "order service unavailable", step2.Error)

	step3 := domain.NewSagaStep(domain.SagaStepInitiatePayment)
	step3.Compensate()
	assert.Equal(t, domain.SagaStepCompensated, step3.Status)
}

func TestIsValidStatus(t *testing.T) {
	assert.True(t, domain.IsValidStatus(domain.StatusInitiated))
	assert.True(t, domain.IsValidStatus(domain.StatusCompleted))
	assert.True(t, domain.IsValidStatus(domain.StatusFailed))
	assert.False(t, domain.IsValidStatus("unknown"))
	assert.False(t, domain.IsValidStatus(""))
}
