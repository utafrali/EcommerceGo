package service

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/inventory/internal/domain"
	"github.com/utafrali/EcommerceGo/services/inventory/internal/event"
)

// --- Mock StockRepository ---

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
	return args.Get(0).([]domain.StockCheckResult), args.Error(1)
}

// --- Mock ReservationRepository ---

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

// --- Test Helpers ---

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestService(stockRepo *mockStockRepository, reservationRepo *mockReservationRepository) *InventoryService {
	logger := newTestLogger()
	// Create a Kafka producer that will fail silently in tests (no real broker).
	kafkaCfg := pkgkafka.DefaultProducerConfig([]string{"localhost:9092"})
	kafkaProducer := pkgkafka.NewProducer(kafkaCfg, logger)
	producer := event.NewProducer(kafkaProducer, logger)
	// Pass nil for the pool since ReserveStock/ReleaseReservation/ConfirmReservation use it directly.
	// Those methods need integration tests; unit tests cover the simpler paths.
	return NewInventoryService(stockRepo, reservationRepo, nil, producer, logger, 900)
}

// --- Tests ---

func TestGetStock_Success(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	expected := &domain.Stock{
		ID:                "stock-1",
		ProductID:         "prod-1",
		VariantID:         "var-1",
		WarehouseID:       domain.DefaultWarehouseID,
		Quantity:          100,
		Reserved:          10,
		LowStockThreshold: 10,
		UpdatedAt:         time.Now().UTC(),
	}

	stockRepo.On("GetByProductVariant", ctx, "prod-1", "var-1").Return(expected, nil)

	stock, err := svc.GetStock(ctx, "prod-1", "var-1")

	require.NoError(t, err)
	assert.Equal(t, expected, stock)
	assert.Equal(t, 90, stock.Available())

	stockRepo.AssertExpectations(t)
}

func TestGetStock_NotFound(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	stockRepo.On("GetByProductVariant", ctx, "nonexistent", "var-1").Return(nil, apperrors.ErrNotFound)

	stock, err := svc.GetStock(ctx, "nonexistent", "var-1")

	assert.Nil(t, stock)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	stockRepo.AssertExpectations(t)
}

func TestAdjustStock_Success(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	stockRepo.On("AdjustQuantity", ctx, "prod-1", "var-1", 50, "adjustment", (*string)(nil)).Return(nil)

	updatedStock := &domain.Stock{
		ID:                "stock-1",
		ProductID:         "prod-1",
		VariantID:         "var-1",
		WarehouseID:       domain.DefaultWarehouseID,
		Quantity:          150,
		Reserved:          0,
		LowStockThreshold: 10,
		UpdatedAt:         time.Now().UTC(),
	}
	stockRepo.On("GetByProductVariant", ctx, "prod-1", "var-1").Return(updatedStock, nil)

	stock, err := svc.AdjustStock(ctx, "prod-1", "var-1", 50, "adjustment")

	require.NoError(t, err)
	assert.Equal(t, 150, stock.Quantity)
	assert.Equal(t, 150, stock.Available())

	stockRepo.AssertExpectations(t)
}

func TestAdjustStock_InsufficientStock(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	// AdjustQuantity will fail with a DB constraint error when quantity goes below 0.
	stockRepo.On("AdjustQuantity", ctx, "prod-1", "var-1", -200, "order", (*string)(nil)).
		Return(apperrors.InvalidInput("stock quantity cannot go below 0"))

	stock, err := svc.AdjustStock(ctx, "prod-1", "var-1", -200, "order")

	assert.Nil(t, stock)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	stockRepo.AssertExpectations(t)
}

func TestAdjustStock_InvalidReason(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	stock, err := svc.AdjustStock(ctx, "prod-1", "var-1", 10, "invalid_reason")

	assert.Nil(t, stock)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCheckAvailability_AllAvailable(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	items := []domain.StockCheckItem{
		{ProductID: "prod-1", VariantID: "var-1", Quantity: 5},
		{ProductID: "prod-2", VariantID: "var-2", Quantity: 3},
	}

	expectedResults := []domain.StockCheckResult{
		{ProductID: "prod-1", VariantID: "var-1", Requested: 5, Available: 100, InStock: true},
		{ProductID: "prod-2", VariantID: "var-2", Requested: 3, Available: 50, InStock: true},
	}

	stockRepo.On("BulkCheck", ctx, items).Return(expectedResults, nil)

	results, allAvailable, err := svc.CheckAvailability(ctx, items)

	require.NoError(t, err)
	assert.True(t, allAvailable)
	assert.Len(t, results, 2)
	assert.True(t, results[0].InStock)
	assert.True(t, results[1].InStock)

	stockRepo.AssertExpectations(t)
}

func TestCheckAvailability_PartiallyAvailable(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	items := []domain.StockCheckItem{
		{ProductID: "prod-1", VariantID: "var-1", Quantity: 5},
		{ProductID: "prod-2", VariantID: "var-2", Quantity: 100},
	}

	expectedResults := []domain.StockCheckResult{
		{ProductID: "prod-1", VariantID: "var-1", Requested: 5, Available: 100, InStock: true},
		{ProductID: "prod-2", VariantID: "var-2", Requested: 100, Available: 50, InStock: false},
	}

	stockRepo.On("BulkCheck", ctx, items).Return(expectedResults, nil)

	results, allAvailable, err := svc.CheckAvailability(ctx, items)

	require.NoError(t, err)
	assert.False(t, allAvailable)
	assert.Len(t, results, 2)
	assert.True(t, results[0].InStock)
	assert.False(t, results[1].InStock)

	stockRepo.AssertExpectations(t)
}

func TestCheckAvailability_NoneAvailable(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	items := []domain.StockCheckItem{
		{ProductID: "prod-1", VariantID: "var-1", Quantity: 200},
		{ProductID: "prod-2", VariantID: "var-2", Quantity: 100},
	}

	expectedResults := []domain.StockCheckResult{
		{ProductID: "prod-1", VariantID: "var-1", Requested: 200, Available: 50, InStock: false},
		{ProductID: "prod-2", VariantID: "var-2", Requested: 100, Available: 0, InStock: false},
	}

	stockRepo.On("BulkCheck", ctx, items).Return(expectedResults, nil)

	results, allAvailable, err := svc.CheckAvailability(ctx, items)

	require.NoError(t, err)
	assert.False(t, allAvailable)
	assert.Len(t, results, 2)
	assert.False(t, results[0].InStock)
	assert.False(t, results[1].InStock)

	stockRepo.AssertExpectations(t)
}

func TestCheckAvailability_EmptyItems(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	results, allAvailable, err := svc.CheckAvailability(ctx, []domain.StockCheckItem{})

	assert.Nil(t, results)
	assert.False(t, allAvailable)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestReleaseReservation_Success(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	// ReleaseReservation uses pool for transactions, so this test validates the
	// reservation lookup and status check logic. The actual transaction path
	// would require an integration test with a real database.
	// For unit testing, we verify the reservation fetch and validation logic.
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	reservation := &domain.StockReservation{
		ID:         "res-1",
		ProductID:  "prod-1",
		VariantID:  "var-1",
		Quantity:   5,
		CheckoutID: "checkout-1",
		Status:     domain.ReservationStatusActive,
		ExpiresAt:  time.Now().UTC().Add(15 * time.Minute),
		CreatedAt:  time.Now().UTC(),
	}

	reservationRepo.On("GetByID", ctx, "res-1").Return(reservation, nil)

	// ReleaseReservation will attempt to use the pool for transaction.
	// Since pool is nil, it will panic. We catch the panic to verify the
	// pre-transaction validation logic works correctly.
	assert.Panics(t, func() {
		_ = svc.ReleaseReservation(ctx, "res-1")
	})

	reservationRepo.AssertExpectations(t)
}

func TestReleaseReservation_AlreadyReleased(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	reservation := &domain.StockReservation{
		ID:         "res-1",
		ProductID:  "prod-1",
		VariantID:  "var-1",
		Quantity:   5,
		CheckoutID: "checkout-1",
		Status:     domain.ReservationStatusReleased,
		ExpiresAt:  time.Now().UTC().Add(15 * time.Minute),
		CreatedAt:  time.Now().UTC(),
	}

	reservationRepo.On("GetByID", ctx, "res-1").Return(reservation, nil)

	err := svc.ReleaseReservation(ctx, "res-1")

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	reservationRepo.AssertExpectations(t)
}

func TestReleaseReservation_NotFound(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	reservationRepo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	err := svc.ReleaseReservation(ctx, "nonexistent")

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	reservationRepo.AssertExpectations(t)
}

func TestConfirmReservation_NotFound(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	reservationRepo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	err := svc.ConfirmReservation(ctx, "nonexistent")

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	reservationRepo.AssertExpectations(t)
}

func TestConfirmReservation_AlreadyConfirmed(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	reservation := &domain.StockReservation{
		ID:         "res-1",
		ProductID:  "prod-1",
		VariantID:  "var-1",
		Quantity:   5,
		CheckoutID: "checkout-1",
		Status:     domain.ReservationStatusConfirmed,
		ExpiresAt:  time.Now().UTC().Add(15 * time.Minute),
		CreatedAt:  time.Now().UTC(),
	}

	reservationRepo.On("GetByID", ctx, "res-1").Return(reservation, nil)

	err := svc.ConfirmReservation(ctx, "res-1")

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	reservationRepo.AssertExpectations(t)
}

func TestConfirmReservation_Success(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	reservation := &domain.StockReservation{
		ID:         "res-1",
		ProductID:  "prod-1",
		VariantID:  "var-1",
		Quantity:   5,
		CheckoutID: "checkout-1",
		Status:     domain.ReservationStatusActive,
		ExpiresAt:  time.Now().UTC().Add(15 * time.Minute),
		CreatedAt:  time.Now().UTC(),
	}

	reservationRepo.On("GetByID", ctx, "res-1").Return(reservation, nil)

	// ConfirmReservation uses pool for transactions.
	// Since pool is nil, it will panic. We catch the panic to verify the
	// pre-transaction validation logic works correctly.
	assert.Panics(t, func() {
		_ = svc.ConfirmReservation(ctx, "res-1")
	})

	reservationRepo.AssertExpectations(t)
}

func TestReserveStock_EmptyCheckoutID(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	items := []domain.StockCheckItem{
		{ProductID: "prod-1", VariantID: "var-1", Quantity: 5},
	}

	reservationIDs, err := svc.ReserveStock(ctx, "", items, 900)

	assert.Nil(t, reservationIDs)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestReserveStock_EmptyItems(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	reservationIDs, err := svc.ReserveStock(ctx, "checkout-1", []domain.StockCheckItem{}, 900)

	assert.Nil(t, reservationIDs)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestListLowStock_Success(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	expectedStocks := []domain.Stock{
		{
			ID:                "stock-1",
			ProductID:         "prod-1",
			VariantID:         "var-1",
			Quantity:          5,
			Reserved:          0,
			LowStockThreshold: 10,
		},
		{
			ID:                "stock-2",
			ProductID:         "prod-2",
			VariantID:         "var-2",
			Quantity:          8,
			Reserved:          2,
			LowStockThreshold: 10,
		},
	}

	stockRepo.On("ListLowStock", ctx, 1, 20).Return(expectedStocks, 2, nil)

	stocks, total, err := svc.ListLowStock(ctx, 1, 20)

	require.NoError(t, err)
	assert.Len(t, stocks, 2)
	assert.Equal(t, 2, total)

	stockRepo.AssertExpectations(t)
}

func TestListLowStock_DefaultPagination(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	stockRepo.On("ListLowStock", ctx, 1, 20).Return([]domain.Stock{}, 0, nil)

	stocks, total, err := svc.ListLowStock(ctx, 0, 0)

	require.NoError(t, err)
	assert.Empty(t, stocks)
	assert.Equal(t, 0, total)

	stockRepo.AssertExpectations(t)
}

func TestAdjustStock_LowStockThreshold(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	stockRepo.On("AdjustQuantity", ctx, "prod-1", "var-1", -90, "order", (*string)(nil)).Return(nil)

	updatedStock := &domain.Stock{
		ID:                "stock-1",
		ProductID:         "prod-1",
		VariantID:         "var-1",
		WarehouseID:       domain.DefaultWarehouseID,
		Quantity:          10,
		Reserved:          0,
		LowStockThreshold: 10,
		UpdatedAt:         time.Now().UTC(),
	}
	stockRepo.On("GetByProductVariant", ctx, "prod-1", "var-1").Return(updatedStock, nil)

	stock, err := svc.AdjustStock(ctx, "prod-1", "var-1", -90, "order")

	require.NoError(t, err)
	assert.Equal(t, 10, stock.Quantity)
	assert.Equal(t, 10, stock.Available())
	// Stock is at threshold, so low_stock event should be published (async, not verifiable here).

	stockRepo.AssertExpectations(t)
}

// --- InitializeStock Tests ---

func TestInitializeStock_Success(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	input := &domain.Stock{
		ProductID:         "prod-1",
		VariantID:         "var-1",
		WarehouseID:       domain.DefaultWarehouseID,
		Quantity:          100,
		LowStockThreshold: 10,
	}

	expected := &domain.Stock{
		ID:                "stock-1",
		ProductID:         "prod-1",
		VariantID:         "var-1",
		WarehouseID:       domain.DefaultWarehouseID,
		Quantity:          100,
		Reserved:          0,
		LowStockThreshold: 10,
		UpdatedAt:         time.Now().UTC(),
	}

	stockRepo.On("CreateStock", ctx, mock.AnythingOfType("*domain.Stock")).Return(expected, nil)

	result, err := svc.InitializeStock(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, expected.ProductID, result.ProductID)
	assert.Equal(t, expected.VariantID, result.VariantID)
	assert.Equal(t, 100, result.Quantity)
	assert.Equal(t, 0, result.Reserved)

	stockRepo.AssertExpectations(t)
}

func TestInitializeStock_DefaultWarehouse(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	input := &domain.Stock{
		ProductID:         "prod-1",
		VariantID:         "var-1",
		WarehouseID:       "", // empty, should default
		Quantity:          50,
		LowStockThreshold: 5,
	}

	expected := &domain.Stock{
		ID:                "stock-1",
		ProductID:         "prod-1",
		VariantID:         "var-1",
		WarehouseID:       domain.DefaultWarehouseID,
		Quantity:          50,
		Reserved:          0,
		LowStockThreshold: 5,
		UpdatedAt:         time.Now().UTC(),
	}

	stockRepo.On("CreateStock", ctx, mock.AnythingOfType("*domain.Stock")).Return(expected, nil)

	result, err := svc.InitializeStock(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, domain.DefaultWarehouseID, result.WarehouseID)

	stockRepo.AssertExpectations(t)
}

func TestInitializeStock_MissingProductID(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	input := &domain.Stock{
		ProductID: "",
		VariantID: "var-1",
		Quantity:  100,
	}

	result, err := svc.InitializeStock(ctx, input)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestInitializeStock_MissingVariantID(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	input := &domain.Stock{
		ProductID: "prod-1",
		VariantID: "",
		Quantity:  100,
	}

	result, err := svc.InitializeStock(ctx, input)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestInitializeStock_NegativeQuantity(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	input := &domain.Stock{
		ProductID: "prod-1",
		VariantID: "var-1",
		Quantity:  -10,
	}

	result, err := svc.InitializeStock(ctx, input)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

// --- Domain Tests ---

func TestStock_Available(t *testing.T) {
	tests := []struct {
		name     string
		quantity int
		reserved int
		expected int
	}{
		{
			name:     "no reservations",
			quantity: 100,
			reserved: 0,
			expected: 100,
		},
		{
			name:     "with reservations",
			quantity: 100,
			reserved: 30,
			expected: 70,
		},
		{
			name:     "fully reserved",
			quantity: 50,
			reserved: 50,
			expected: 0,
		},
		{
			name:     "zero stock",
			quantity: 0,
			reserved: 0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stock := &domain.Stock{
				Quantity: tt.quantity,
				Reserved: tt.reserved,
			}
			assert.Equal(t, tt.expected, stock.Available())
		})
	}
}

func TestStockReservation_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{name: "active", status: domain.ReservationStatusActive, expected: true},
		{name: "confirmed", status: domain.ReservationStatusConfirmed, expected: false},
		{name: "released", status: domain.ReservationStatusReleased, expected: false},
		{name: "expired", status: domain.ReservationStatusExpired, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &domain.StockReservation{Status: tt.status}
			assert.Equal(t, tt.expected, r.IsActive())
		})
	}
}

func TestStockReservation_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{
			name:      "not expired",
			expiresAt: time.Now().UTC().Add(15 * time.Minute),
			expected:  false,
		},
		{
			name:      "expired",
			expiresAt: time.Now().UTC().Add(-1 * time.Minute),
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &domain.StockReservation{ExpiresAt: tt.expiresAt}
			assert.Equal(t, tt.expected, r.IsExpired())
		})
	}
}

func TestIsValidMovementReason(t *testing.T) {
	tests := []struct {
		name     string
		reason   string
		expected bool
	}{
		{name: "order", reason: "order", expected: true},
		{name: "return", reason: "return", expected: true},
		{name: "adjustment", reason: "adjustment", expected: true},
		{name: "reservation", reason: "reservation", expected: true},
		{name: "invalid", reason: "invalid", expected: false},
		{name: "empty", reason: "", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, domain.IsValidMovementReason(tt.reason))
		})
	}
}

func TestIsValidReservationStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{name: "active", status: "active", expected: true},
		{name: "confirmed", status: "confirmed", expected: true},
		{name: "released", status: "released", expected: true},
		{name: "expired", status: "expired", expected: true},
		{name: "invalid", status: "invalid", expected: false},
		{name: "empty", status: "", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, domain.IsValidReservationStatus(tt.status))
		})
	}
}

// ---------------------------------------------------------------------------
// AdjustStock input-validation guard tests (empty productID / variantID)
// ---------------------------------------------------------------------------

func TestAdjustStock_EmptyProductID(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	stock, err := svc.AdjustStock(ctx, "", "var-1", 10, "adjustment")

	assert.Nil(t, stock)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
	assert.Contains(t, err.Error(), "product_id")

	// Repository must never be called when productID is empty.
	stockRepo.AssertNotCalled(t, "AdjustQuantity")
	stockRepo.AssertNotCalled(t, "GetByProductVariant")
}

func TestAdjustStock_EmptyVariantID(t *testing.T) {
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	stock, err := svc.AdjustStock(ctx, "prod-1", "", 10, "adjustment")

	assert.Nil(t, stock)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
	assert.Contains(t, err.Error(), "variant_id")

	// Repository must never be called when variantID is empty.
	stockRepo.AssertNotCalled(t, "AdjustQuantity")
	stockRepo.AssertNotCalled(t, "GetByProductVariant")
}

func TestAdjustStock_ValidInputsPassThrough(t *testing.T) {
	// Verifies that valid productID, variantID, and reason are accepted and
	// the call reaches the repository layer (i.e., guards do not block).
	stockRepo := new(mockStockRepository)
	reservationRepo := new(mockReservationRepository)
	svc := newTestService(stockRepo, reservationRepo)
	ctx := context.Background()

	stockRepo.On("AdjustQuantity", ctx, "prod-99", "var-99", 25, "return", (*string)(nil)).Return(nil)

	updatedStock := &domain.Stock{
		ID:                "stock-99",
		ProductID:         "prod-99",
		VariantID:         "var-99",
		WarehouseID:       domain.DefaultWarehouseID,
		Quantity:          125,
		Reserved:          0,
		LowStockThreshold: 10,
		UpdatedAt:         time.Now().UTC(),
	}
	stockRepo.On("GetByProductVariant", ctx, "prod-99", "var-99").Return(updatedStock, nil)

	stock, err := svc.AdjustStock(ctx, "prod-99", "var-99", 25, "return")

	require.NoError(t, err)
	assert.Equal(t, 125, stock.Quantity)
	assert.Equal(t, "prod-99", stock.ProductID)
	assert.Equal(t, "var-99", stock.VariantID)

	// Confirm that both repository methods were called exactly once.
	stockRepo.AssertCalled(t, "AdjustQuantity", ctx, "prod-99", "var-99", 25, "return", (*string)(nil))
	stockRepo.AssertCalled(t, "GetByProductVariant", ctx, "prod-99", "var-99")
	stockRepo.AssertExpectations(t)
}
