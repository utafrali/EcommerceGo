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
	"github.com/utafrali/EcommerceGo/services/cart/internal/domain"
	"github.com/utafrali/EcommerceGo/services/cart/internal/event"
)

// --- Mock Repository ---

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

// --- Test Helpers ---

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestService(repo *mockCartRepository) *CartService {
	logger := newTestLogger()
	// Create a Kafka producer that will fail silently in tests (no real broker).
	kafkaCfg := pkgkafka.DefaultProducerConfig([]string{"localhost:9092"})
	kafkaProducer := pkgkafka.NewProducer(kafkaCfg, logger)
	producer := event.NewProducer(kafkaProducer, logger)
	return NewCartService(repo, producer, logger, 7*24*time.Hour)
}

func newCartWithItem(userID string) *domain.Cart {
	now := time.Now().UTC()
	return &domain.Cart{
		ID:     "cart-123",
		UserID: userID,
		Items: []domain.CartItem{
			{
				ProductID: "prod-1",
				VariantID: "var-1",
				Name:      "Test Product",
				SKU:       "TP-001",
				Price:     1999,
				Quantity:  2,
			},
		},
		Currency:  "USD",
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
}

// --- Tests ---

func TestGetCart_Empty(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Get", ctx, "user-1").Return(nil, apperrors.NotFound("cart", "user-1"))

	cart, err := svc.GetCart(ctx, "user-1")

	require.NoError(t, err)
	assert.NotEmpty(t, cart.ID)
	assert.Equal(t, "user-1", cart.UserID)
	assert.Empty(t, cart.Items)
	assert.Equal(t, "USD", cart.Currency)
	assert.Equal(t, 0, cart.Version)
	assert.NotZero(t, cart.CreatedAt)
	assert.NotZero(t, cart.UpdatedAt)
	assert.NotZero(t, cart.ExpiresAt)

	repo.AssertExpectations(t)
}

func TestGetCart_Existing(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	expected := newCartWithItem("user-1")
	repo.On("Get", ctx, "user-1").Return(expected, nil)

	cart, err := svc.GetCart(ctx, "user-1")

	require.NoError(t, err)
	assert.Equal(t, expected, cart)
	assert.Len(t, cart.Items, 1)

	repo.AssertExpectations(t)
}

func TestAddItem_NewItem(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	// Cart does not exist yet, returns not found.
	repo.On("Get", ctx, "user-1").Return(nil, apperrors.NotFound("cart", "user-1"))
	repo.On("SaveIfVersion", ctx, mock.AnythingOfType("*domain.Cart"), 0).Return(true, nil)

	input := AddItemInput{
		ProductID: "prod-1",
		VariantID: "var-1",
		Name:      "Test Product",
		SKU:       "TP-001",
		Price:     1999,
		Quantity:  1,
		ImageURL:  "https://example.com/img.jpg",
	}

	cart, err := svc.AddItem(ctx, "user-1", input)

	require.NoError(t, err)
	assert.NotEmpty(t, cart.ID)
	assert.Equal(t, "user-1", cart.UserID)
	require.Len(t, cart.Items, 1)
	assert.Equal(t, "prod-1", cart.Items[0].ProductID)
	assert.Equal(t, "var-1", cart.Items[0].VariantID)
	assert.Equal(t, "Test Product", cart.Items[0].Name)
	assert.Equal(t, "TP-001", cart.Items[0].SKU)
	assert.Equal(t, int64(1999), cart.Items[0].Price)
	assert.Equal(t, 1, cart.Items[0].Quantity)
	assert.Equal(t, "https://example.com/img.jpg", cart.Items[0].ImageURL)

	repo.AssertExpectations(t)
}

func TestAddItem_MergeExisting(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := newCartWithItem("user-1")
	repo.On("Get", ctx, "user-1").Return(existing, nil)
	repo.On("SaveIfVersion", ctx, mock.AnythingOfType("*domain.Cart"), 1).Return(true, nil)

	// Add the same product+variant again.
	input := AddItemInput{
		ProductID: "prod-1",
		VariantID: "var-1",
		Name:      "Test Product",
		SKU:       "TP-001",
		Price:     1999,
		Quantity:  3,
	}

	cart, err := svc.AddItem(ctx, "user-1", input)

	require.NoError(t, err)
	require.Len(t, cart.Items, 1)
	// Quantity should be merged: 2 (existing) + 3 (new) = 5.
	assert.Equal(t, 5, cart.Items[0].Quantity)

	repo.AssertExpectations(t)
}

func TestAddItem_DifferentVariant(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := newCartWithItem("user-1")
	repo.On("Get", ctx, "user-1").Return(existing, nil)
	repo.On("SaveIfVersion", ctx, mock.AnythingOfType("*domain.Cart"), 1).Return(true, nil)

	// Add a different variant of the same product.
	input := AddItemInput{
		ProductID: "prod-1",
		VariantID: "var-2",
		Name:      "Test Product (Large)",
		SKU:       "TP-002",
		Price:     2499,
		Quantity:  1,
	}

	cart, err := svc.AddItem(ctx, "user-1", input)

	require.NoError(t, err)
	assert.Len(t, cart.Items, 2)

	repo.AssertExpectations(t)
}

func TestAddItem_InvalidQuantity(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := AddItemInput{
		ProductID: "prod-1",
		VariantID: "var-1",
		Name:      "Test Product",
		SKU:       "TP-001",
		Price:     1999,
		Quantity:  0,
	}

	cart, err := svc.AddItem(ctx, "user-1", input)

	assert.Nil(t, cart)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestAddItem_NegativeQuantity(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := AddItemInput{
		ProductID: "prod-1",
		VariantID: "var-1",
		Name:      "Test Product",
		SKU:       "TP-001",
		Price:     1999,
		Quantity:  -1,
	}

	cart, err := svc.AddItem(ctx, "user-1", input)

	assert.Nil(t, cart)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestAddItem_NegativePrice(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := AddItemInput{
		ProductID: "prod-1",
		VariantID: "var-1",
		Name:      "Test Product",
		SKU:       "TP-001",
		Price:     -100,
		Quantity:  1,
	}

	cart, err := svc.AddItem(ctx, "user-1", input)

	assert.Nil(t, cart)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestAddItem_EmptyUserID(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := AddItemInput{
		ProductID: "prod-1",
		VariantID: "var-1",
		Name:      "Test Product",
		SKU:       "TP-001",
		Price:     1999,
		Quantity:  1,
	}

	cart, err := svc.AddItem(ctx, "", input)

	assert.Nil(t, cart)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestAddItem_EmptyProductID(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := AddItemInput{
		ProductID: "",
		VariantID: "var-1",
		Name:      "Test Product",
		SKU:       "TP-001",
		Price:     1999,
		Quantity:  1,
	}

	cart, err := svc.AddItem(ctx, "user-1", input)

	assert.Nil(t, cart)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestAddItem_ConcurrentModificationConflict(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := newCartWithItem("user-1")
	repo.On("Get", ctx, "user-1").Return(existing, nil)
	// SaveIfVersion returns false to simulate a version conflict.
	repo.On("SaveIfVersion", ctx, mock.AnythingOfType("*domain.Cart"), 1).Return(false, nil)

	input := AddItemInput{
		ProductID: "prod-1",
		VariantID: "var-2",
		Name:      "Another Product",
		SKU:       "AP-001",
		Price:     999,
		Quantity:  1,
	}

	cart, err := svc.AddItem(ctx, "user-1", input)

	assert.Nil(t, cart)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrConflict)
	assert.Contains(t, err.Error(), "cart was modified concurrently")

	repo.AssertExpectations(t)
}

func TestUpdateItemQuantity_Success(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := newCartWithItem("user-1")
	repo.On("Get", ctx, "user-1").Return(existing, nil)
	repo.On("SaveIfVersion", ctx, mock.AnythingOfType("*domain.Cart"), 1).Return(true, nil)

	cart, err := svc.UpdateItemQuantity(ctx, "user-1", "prod-1", "var-1", 5)

	require.NoError(t, err)
	require.Len(t, cart.Items, 1)
	assert.Equal(t, 5, cart.Items[0].Quantity)

	repo.AssertExpectations(t)
}

func TestUpdateItemQuantity_ZeroRemoves(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := newCartWithItem("user-1")
	repo.On("Get", ctx, "user-1").Return(existing, nil)
	repo.On("SaveIfVersion", ctx, mock.AnythingOfType("*domain.Cart"), 1).Return(true, nil)

	cart, err := svc.UpdateItemQuantity(ctx, "user-1", "prod-1", "var-1", 0)

	require.NoError(t, err)
	assert.Empty(t, cart.Items)

	repo.AssertExpectations(t)
}

func TestUpdateItemQuantity_ItemNotFound(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := newCartWithItem("user-1")
	repo.On("Get", ctx, "user-1").Return(existing, nil)

	cart, err := svc.UpdateItemQuantity(ctx, "user-1", "prod-999", "var-999", 5)

	assert.Nil(t, cart)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	repo.AssertExpectations(t)
}

func TestUpdateItemQuantity_NegativeQuantity(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	cart, err := svc.UpdateItemQuantity(ctx, "user-1", "prod-1", "var-1", -1)

	assert.Nil(t, cart)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestUpdateItemQuantity_CartNotFound(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Get", ctx, "user-1").Return(nil, apperrors.NotFound("cart", "user-1"))

	cart, err := svc.UpdateItemQuantity(ctx, "user-1", "prod-1", "var-1", 5)

	assert.Nil(t, cart)
	assert.Error(t, err)

	repo.AssertExpectations(t)
}

func TestUpdateItemQuantity_ConcurrentModificationConflict(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := newCartWithItem("user-1")
	repo.On("Get", ctx, "user-1").Return(existing, nil)
	repo.On("SaveIfVersion", ctx, mock.AnythingOfType("*domain.Cart"), 1).Return(false, nil)

	cart, err := svc.UpdateItemQuantity(ctx, "user-1", "prod-1", "var-1", 5)

	assert.Nil(t, cart)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrConflict)
	assert.Contains(t, err.Error(), "cart was modified concurrently")

	repo.AssertExpectations(t)
}

func TestRemoveItem_Success(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := newCartWithItem("user-1")
	repo.On("Get", ctx, "user-1").Return(existing, nil)
	repo.On("SaveIfVersion", ctx, mock.AnythingOfType("*domain.Cart"), 1).Return(true, nil)

	cart, err := svc.RemoveItem(ctx, "user-1", "prod-1", "var-1")

	require.NoError(t, err)
	assert.Empty(t, cart.Items)

	repo.AssertExpectations(t)
}

func TestRemoveItem_NotFound(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := newCartWithItem("user-1")
	repo.On("Get", ctx, "user-1").Return(existing, nil)

	cart, err := svc.RemoveItem(ctx, "user-1", "prod-999", "var-999")

	assert.Nil(t, cart)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	repo.AssertExpectations(t)
}

func TestRemoveItem_CartNotFound(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Get", ctx, "user-1").Return(nil, apperrors.NotFound("cart", "user-1"))

	cart, err := svc.RemoveItem(ctx, "user-1", "prod-1", "var-1")

	assert.Nil(t, cart)
	assert.Error(t, err)

	repo.AssertExpectations(t)
}

func TestRemoveItem_ConcurrentModificationConflict(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	existing := newCartWithItem("user-1")
	repo.On("Get", ctx, "user-1").Return(existing, nil)
	repo.On("SaveIfVersion", ctx, mock.AnythingOfType("*domain.Cart"), 1).Return(false, nil)

	cart, err := svc.RemoveItem(ctx, "user-1", "prod-1", "var-1")

	assert.Nil(t, cart)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrConflict)
	assert.Contains(t, err.Error(), "cart was modified concurrently")

	repo.AssertExpectations(t)
}

func TestClearCart_Success(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	repo.On("Delete", ctx, "user-1").Return(nil)

	err := svc.ClearCart(ctx, "user-1")

	require.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestClearCart_EmptyUserID(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	err := svc.ClearCart(ctx, "")

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestCartTotalAmount(t *testing.T) {
	cart := &domain.Cart{
		Items: []domain.CartItem{
			{Price: 1000, Quantity: 2},
			{Price: 500, Quantity: 3},
		},
	}

	assert.Equal(t, int64(3500), cart.TotalAmount())
}

func TestCartTotalAmount_Empty(t *testing.T) {
	cart := &domain.Cart{
		Items: []domain.CartItem{},
	}

	assert.Equal(t, int64(0), cart.TotalAmount())
}

func TestCartItemCount(t *testing.T) {
	cart := &domain.Cart{
		Items: []domain.CartItem{
			{Quantity: 2},
			{Quantity: 3},
		},
	}

	assert.Equal(t, 5, cart.ItemCount())
}

func TestCartItemCount_Empty(t *testing.T) {
	cart := &domain.Cart{
		Items: []domain.CartItem{},
	}

	assert.Equal(t, 0, cart.ItemCount())
}

func TestGetCart_EmptyUserID(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	cart, err := svc.GetCart(ctx, "")

	assert.Nil(t, cart)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

// --- Validation limit tests ---

func TestAddItem_ExceedsMaxQuantity(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := AddItemInput{
		ProductID: "prod-1",
		VariantID: "var-1",
		Name:      "Test Product",
		SKU:       "TP-001",
		Price:     1999,
		Quantity:  101, // exceeds MaxQuantityPerItem (100)
	}

	cart, err := svc.AddItem(ctx, "user-1", input)

	assert.Nil(t, cart)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
	assert.Contains(t, err.Error(), "quantity must not exceed 100")
}

func TestAddItem_ExceedsMaxPrice(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	input := AddItemInput{
		ProductID: "prod-1",
		VariantID: "var-1",
		Name:      "Expensive Product",
		SKU:       "EP-001",
		Price:     MaxPriceCents + 1, // exceeds MaxPriceCents (10_000_000)
		Quantity:  1,
	}

	cart, err := svc.AddItem(ctx, "user-1", input)

	assert.Nil(t, cart)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
	assert.Contains(t, err.Error(), "price must not exceed")
}

func TestAddItem_ExceedsMaxItemsPerCart(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	// Build a cart that already has MaxItemsPerCart (50) distinct items.
	now := time.Now().UTC()
	fullCart := &domain.Cart{
		ID:        "cart-full",
		UserID:    "user-1",
		Items:     make([]domain.CartItem, MaxItemsPerCart),
		Currency:  "USD",
		Version:   3,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	for i := 0; i < MaxItemsPerCart; i++ {
		fullCart.Items[i] = domain.CartItem{
			ProductID: fmt.Sprintf("prod-%d", i),
			VariantID: fmt.Sprintf("var-%d", i),
			Name:      fmt.Sprintf("Product %d", i),
			SKU:       fmt.Sprintf("SKU-%d", i),
			Price:     999,
			Quantity:  1,
		}
	}

	repo.On("Get", ctx, "user-1").Return(fullCart, nil)

	// Try to add the 51st distinct item.
	input := AddItemInput{
		ProductID: "prod-new",
		VariantID: "var-new",
		Name:      "New Product",
		SKU:       "NP-001",
		Price:     999,
		Quantity:  1,
	}

	cart, err := svc.AddItem(ctx, "user-1", input)

	assert.Nil(t, cart)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
	assert.Contains(t, err.Error(), "cart must not contain more than 50 items")

	repo.AssertExpectations(t)
}

func TestAddItem_MergeExceedsMaxQuantity(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	// Existing cart with an item at quantity 60.
	now := time.Now().UTC()
	existing := &domain.Cart{
		ID:     "cart-123",
		UserID: "user-1",
		Items: []domain.CartItem{
			{
				ProductID: "prod-1",
				VariantID: "var-1",
				Name:      "Test Product",
				SKU:       "TP-001",
				Price:     1999,
				Quantity:  60,
			},
		},
		Currency:  "USD",
		Version:   2,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}

	repo.On("Get", ctx, "user-1").Return(existing, nil)

	// Adding 50 more to the same product+variant: 60 + 50 = 110 > MaxQuantityPerItem (100).
	input := AddItemInput{
		ProductID: "prod-1",
		VariantID: "var-1",
		Name:      "Test Product",
		SKU:       "TP-001",
		Price:     1999,
		Quantity:  50,
	}

	cart, err := svc.AddItem(ctx, "user-1", input)

	assert.Nil(t, cart)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
	assert.Contains(t, err.Error(), "combined quantity must not exceed 100")

	repo.AssertExpectations(t)
}

func TestUpdateItemQuantity_ExceedsMaxQuantity(t *testing.T) {
	repo := new(mockCartRepository)
	svc := newTestService(repo)
	ctx := context.Background()

	// quantity=101 should be rejected before the repo is even consulted.
	cart, err := svc.UpdateItemQuantity(ctx, "user-1", "prod-1", "var-1", 101)

	assert.Nil(t, cart)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
	assert.Contains(t, err.Error(), "quantity must not exceed 100")
}
