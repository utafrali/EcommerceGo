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
	"github.com/utafrali/EcommerceGo/services/order/internal/domain"
	"github.com/utafrali/EcommerceGo/services/order/internal/event"
	"github.com/utafrali/EcommerceGo/services/order/internal/repository"
)

// --- Mock Repository ---

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

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestService(repo *mockOrderRepository) *OrderService {
	logger := newTestLogger()
	// Create a Kafka producer that will fail silently in tests (no real broker).
	kafkaCfg := pkgkafka.DefaultProducerConfig([]string{"localhost:9092"})
	kafkaProducer := pkgkafka.NewProducer(kafkaCfg, logger)
	producer := event.NewProducer(kafkaProducer, logger)
	return NewOrderService(repo, producer, logger)
}

func strPtr(s string) *string {
	return &s
}

// --- Tests ---

func TestCreateOrder_Success(t *testing.T) {
	repo := new(mockOrderRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Order")).Return(nil)

	input := CreateOrderInput{
		UserID: "user-123",
		Items: []CreateOrderItemInput{
			{
				ProductID: "prod-1",
				VariantID: "var-1",
				Name:      "Widget",
				SKU:       "WDG-001",
				Price:     1000,
				Quantity:  2,
			},
			{
				ProductID: "prod-2",
				VariantID: "var-2",
				Name:      "Gadget",
				SKU:       "GDG-001",
				Price:     2500,
				Quantity:  1,
			},
		},
		DiscountAmount: 500,
		ShippingAmount: 300,
		Currency:       "USD",
		ShippingAddress: &domain.Address{
			FullName:    "John Doe",
			AddressLine: "123 Main St",
			City:        "Springfield",
			State:       "IL",
			PostalCode:  "62704",
			Country:     "US",
		},
		Notes: "Please deliver before noon",
	}

	order, err := svc.CreateOrder(ctx, input)

	require.NoError(t, err)
	assert.NotEmpty(t, order.ID)
	assert.Equal(t, "user-123", order.UserID)
	assert.Equal(t, domain.OrderStatusPending, order.Status)
	assert.Len(t, order.Items, 2)
	assert.Equal(t, int64(4500), order.SubtotalAmount) // 1000*2 + 2500*1
	assert.Equal(t, int64(500), order.DiscountAmount)
	assert.Equal(t, int64(300), order.ShippingAmount)
	assert.Equal(t, int64(4300), order.TotalAmount) // 4500 - 500 + 300
	assert.Equal(t, "USD", order.Currency)
	assert.NotNil(t, order.ShippingAddress)
	assert.Equal(t, "Please deliver before noon", order.Notes)
	assert.NotZero(t, order.CreatedAt)
	assert.NotZero(t, order.UpdatedAt)

	// Check that items have proper order_id set.
	for _, item := range order.Items {
		assert.Equal(t, order.ID, item.OrderID)
		assert.NotEmpty(t, item.ID)
	}

	repo.AssertExpectations(t)
}

func TestCreateOrder_EmptyItems(t *testing.T) {
	repo := new(mockOrderRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := CreateOrderInput{
		UserID:   "user-123",
		Items:    []CreateOrderItemInput{},
		Currency: "USD",
	}

	order, err := svc.CreateOrder(ctx, input)

	assert.Nil(t, order)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCreateOrder_InvalidCurrency(t *testing.T) {
	repo := new(mockOrderRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := CreateOrderInput{
		UserID: "user-123",
		Items: []CreateOrderItemInput{
			{ProductID: "prod-1", Name: "Widget", Price: 1000, Quantity: 1},
		},
		Currency: "US", // Invalid: must be 3 characters
	}

	order, err := svc.CreateOrder(ctx, input)

	assert.Nil(t, order)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCreateOrder_MissingUserID(t *testing.T) {
	repo := new(mockOrderRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := CreateOrderInput{
		UserID: "",
		Items: []CreateOrderItemInput{
			{ProductID: "prod-1", Name: "Widget", Price: 1000, Quantity: 1},
		},
		Currency: "USD",
	}

	order, err := svc.CreateOrder(ctx, input)

	assert.Nil(t, order)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCreateOrder_CurrencyUppercased(t *testing.T) {
	repo := new(mockOrderRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Order")).Return(nil)

	input := CreateOrderInput{
		UserID: "user-123",
		Items: []CreateOrderItemInput{
			{ProductID: "prod-1", Name: "Widget", Price: 1000, Quantity: 1},
		},
		Currency: "usd",
	}

	order, err := svc.CreateOrder(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, "USD", order.Currency)

	repo.AssertExpectations(t)
}

func TestGetOrder_Success(t *testing.T) {
	repo := new(mockOrderRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	expected := &domain.Order{
		ID:     "order-123",
		UserID: "user-123",
		Status: domain.OrderStatusPending,
		Items: []domain.OrderItem{
			{ID: "item-1", OrderID: "order-123", ProductID: "prod-1", Name: "Widget", Price: 1000, Quantity: 1},
		},
		TotalAmount: 1000,
		Currency:    "USD",
	}

	repo.On("GetByID", ctx, "order-123").Return(expected, nil)

	order, err := svc.GetOrder(ctx, "order-123")

	require.NoError(t, err)
	assert.Equal(t, expected, order)

	repo.AssertExpectations(t)
}

func TestGetOrder_NotFound(t *testing.T) {
	repo := new(mockOrderRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	order, err := svc.GetOrder(ctx, "nonexistent")

	assert.Nil(t, order)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	repo.AssertExpectations(t)
}

func TestListOrders_Success(t *testing.T) {
	repo := new(mockOrderRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	expectedOrders := []domain.Order{
		{ID: "order-1", UserID: "user-123", Status: domain.OrderStatusPending},
		{ID: "order-2", UserID: "user-123", Status: domain.OrderStatusConfirmed},
	}

	filter := repository.OrderFilter{
		UserID:  strPtr("user-123"),
		Page:    1,
		PerPage: 20,
	}

	repo.On("List", ctx, filter).Return(expectedOrders, 2, nil)

	orders, total, err := svc.ListOrders(ctx, filter)

	require.NoError(t, err)
	assert.Len(t, orders, 2)
	assert.Equal(t, 2, total)

	repo.AssertExpectations(t)
}

func TestListOrders_DefaultPagination(t *testing.T) {
	repo := new(mockOrderRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	// The service should clamp these to defaults.
	filter := repository.OrderFilter{
		Page:    0,
		PerPage: 0,
	}

	expectedFilter := repository.OrderFilter{
		Page:    1,
		PerPage: 20,
	}

	repo.On("List", ctx, expectedFilter).Return([]domain.Order{}, 0, nil)

	orders, total, err := svc.ListOrders(ctx, filter)

	require.NoError(t, err)
	assert.Empty(t, orders)
	assert.Equal(t, 0, total)

	repo.AssertExpectations(t)
}

func TestUpdateOrderStatus_Success(t *testing.T) {
	repo := new(mockOrderRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := &domain.Order{
		ID:     "order-123",
		UserID: "user-123",
		Status: domain.OrderStatusPending,
		Items:  []domain.OrderItem{},
	}

	repo.On("GetByID", ctx, "order-123").Return(existing, nil)
	repo.On("UpdateStatus", ctx, "order-123", domain.OrderStatusConfirmed, "").Return(nil)

	order, err := svc.UpdateOrderStatus(ctx, "order-123", domain.OrderStatusConfirmed, "")

	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusConfirmed, order.Status)

	repo.AssertExpectations(t)
}

func TestUpdateOrderStatus_InvalidTransition(t *testing.T) {
	repo := new(mockOrderRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := &domain.Order{
		ID:     "order-123",
		UserID: "user-123",
		Status: domain.OrderStatusPending,
		Items:  []domain.OrderItem{},
	}

	repo.On("GetByID", ctx, "order-123").Return(existing, nil)

	// Pending cannot transition directly to shipped.
	order, err := svc.UpdateOrderStatus(ctx, "order-123", domain.OrderStatusShipped, "")

	assert.Nil(t, order)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestUpdateOrderStatus_OrderNotFound(t *testing.T) {
	repo := new(mockOrderRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	order, err := svc.UpdateOrderStatus(ctx, "nonexistent", domain.OrderStatusConfirmed, "")

	assert.Nil(t, order)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	repo.AssertExpectations(t)
}

func TestUpdateOrderStatus_InvalidStatus(t *testing.T) {
	repo := new(mockOrderRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	order, err := svc.UpdateOrderStatus(ctx, "order-123", "invalid_status", "")

	assert.Nil(t, order)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCancelOrder_Success(t *testing.T) {
	repo := new(mockOrderRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := &domain.Order{
		ID:     "order-123",
		UserID: "user-123",
		Status: domain.OrderStatusPending,
		Items:  []domain.OrderItem{},
	}

	repo.On("GetByID", ctx, "order-123").Return(existing, nil)
	repo.On("UpdateStatus", ctx, "order-123", domain.OrderStatusCanceled, "customer request").Return(nil)

	order, err := svc.CancelOrder(ctx, "order-123", "customer request")

	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusCanceled, order.Status)
	assert.Equal(t, "customer request", order.CanceledReason)

	repo.AssertExpectations(t)
}

func TestCancelOrder_AlreadyCanceled(t *testing.T) {
	repo := new(mockOrderRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := &domain.Order{
		ID:     "order-123",
		UserID: "user-123",
		Status: domain.OrderStatusCanceled,
		Items:  []domain.OrderItem{},
	}

	repo.On("GetByID", ctx, "order-123").Return(existing, nil)

	order, err := svc.CancelOrder(ctx, "order-123", "duplicate request")

	assert.Nil(t, order)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestCancelOrder_CannotCancelDelivered(t *testing.T) {
	repo := new(mockOrderRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := &domain.Order{
		ID:     "order-123",
		UserID: "user-123",
		Status: domain.OrderStatusDelivered,
		Items:  []domain.OrderItem{},
	}

	repo.On("GetByID", ctx, "order-123").Return(existing, nil)

	order, err := svc.CancelOrder(ctx, "order-123", "too late")

	assert.Nil(t, order)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestCanTransitionTo(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		expected bool
	}{
		// From pending
		{"pending to confirmed", domain.OrderStatusPending, domain.OrderStatusConfirmed, true},
		{"pending to canceled", domain.OrderStatusPending, domain.OrderStatusCanceled, true},
		{"pending to shipped", domain.OrderStatusPending, domain.OrderStatusShipped, false},
		{"pending to delivered", domain.OrderStatusPending, domain.OrderStatusDelivered, false},
		{"pending to refunded", domain.OrderStatusPending, domain.OrderStatusRefunded, false},
		{"pending to processing", domain.OrderStatusPending, domain.OrderStatusProcessing, false},

		// From confirmed
		{"confirmed to processing", domain.OrderStatusConfirmed, domain.OrderStatusProcessing, true},
		{"confirmed to canceled", domain.OrderStatusConfirmed, domain.OrderStatusCanceled, true},
		{"confirmed to shipped", domain.OrderStatusConfirmed, domain.OrderStatusShipped, false},

		// From processing
		{"processing to shipped", domain.OrderStatusProcessing, domain.OrderStatusShipped, true},
		{"processing to canceled", domain.OrderStatusProcessing, domain.OrderStatusCanceled, true},
		{"processing to delivered", domain.OrderStatusProcessing, domain.OrderStatusDelivered, false},

		// From shipped
		{"shipped to delivered", domain.OrderStatusShipped, domain.OrderStatusDelivered, true},
		{"shipped to canceled", domain.OrderStatusShipped, domain.OrderStatusCanceled, false},

		// From delivered
		{"delivered to refunded", domain.OrderStatusDelivered, domain.OrderStatusRefunded, true},
		{"delivered to canceled", domain.OrderStatusDelivered, domain.OrderStatusCanceled, false},

		// Terminal states
		{"canceled to anything", domain.OrderStatusCanceled, domain.OrderStatusPending, false},
		{"refunded to anything", domain.OrderStatusRefunded, domain.OrderStatusPending, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := &domain.Order{Status: tt.from}
			result := order.CanTransitionTo(tt.to)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateOrder_NegativeTotalClampedToZero(t *testing.T) {
	repo := new(mockOrderRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Order")).Return(nil)

	input := CreateOrderInput{
		UserID: "user-456",
		Items: []CreateOrderItemInput{
			{
				ProductID: "prod-1",
				VariantID: "var-1",
				Name:      "Widget",
				SKU:       "WDG-001",
				Price:     1000,
				Quantity:  1,
			},
		},
		DiscountAmount: 5000, // discount (5000) > subtotal (1000) + shipping (200) = 1200
		ShippingAmount: 200,
		Currency:       "USD",
	}

	order, err := svc.CreateOrder(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, int64(1000), order.SubtotalAmount)
	assert.Equal(t, int64(5000), order.DiscountAmount)
	assert.Equal(t, int64(200), order.ShippingAmount)
	assert.Equal(t, int64(0), order.TotalAmount) // 1000 - 5000 + 200 = -3800, clamped to 0

	repo.AssertExpectations(t)
}

func TestCreateOrder_ExactZeroTotal(t *testing.T) {
	repo := new(mockOrderRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Order")).Return(nil)

	input := CreateOrderInput{
		UserID: "user-789",
		Items: []CreateOrderItemInput{
			{
				ProductID: "prod-1",
				VariantID: "var-1",
				Name:      "Widget",
				SKU:       "WDG-001",
				Price:     1000,
				Quantity:  2,
			},
		},
		DiscountAmount: 2500, // discount (2500) == subtotal (2000) + shipping (500)
		ShippingAmount: 500,
		Currency:       "USD",
	}

	order, err := svc.CreateOrder(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, int64(2000), order.SubtotalAmount) // 1000 * 2
	assert.Equal(t, int64(2500), order.DiscountAmount)
	assert.Equal(t, int64(500), order.ShippingAmount)
	assert.Equal(t, int64(0), order.TotalAmount) // 2000 - 2500 + 500 = 0

	repo.AssertExpectations(t)
}

func TestLineTotal(t *testing.T) {
	item := domain.OrderItem{
		Price:    1500,
		Quantity: 3,
	}

	assert.Equal(t, int64(4500), item.LineTotal())
}
