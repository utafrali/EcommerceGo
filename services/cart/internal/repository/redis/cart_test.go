package redis

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/cart/internal/domain"
)

func setupTestRedis(t *testing.T) (*CartRepository, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	repo := NewCartRepository(client, 24*time.Hour)
	return repo, mr
}

func sampleCart() *domain.Cart {
	now := time.Now().UTC().Truncate(time.Millisecond)
	return &domain.Cart{
		ID:     "cart-001",
		UserID: "user-001",
		Items: []domain.CartItem{
			{
				ProductID: "prod-1",
				VariantID: "var-1",
				Name:      "Widget",
				SKU:       "WDG-1",
				Price:     1990,
				Quantity:  2,
				ImageURL:  "https://img.example.com/w.jpg",
			},
		},
		Currency:  "USD",
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestCartRepository_Get_Success(t *testing.T) {
	repo, mr := setupTestRedis(t)

	cart := sampleCart()
	data, err := json.Marshal(cart)
	require.NoError(t, err)

	// Set data directly in miniredis.
	require.NoError(t, mr.Set("cart:"+cart.UserID, string(data)))

	got, err := repo.Get(context.Background(), cart.UserID)
	require.NoError(t, err)
	assert.Equal(t, cart.ID, got.ID)
	assert.Equal(t, cart.UserID, got.UserID)
	assert.Equal(t, cart.Currency, got.Currency)
	assert.Equal(t, cart.Version, got.Version)
	require.Len(t, got.Items, 1)
	assert.Equal(t, "prod-1", got.Items[0].ProductID)
	assert.Equal(t, "var-1", got.Items[0].VariantID)
	assert.Equal(t, "Widget", got.Items[0].Name)
	assert.Equal(t, int64(1990), got.Items[0].Price)
	assert.Equal(t, 2, got.Items[0].Quantity)
}

func TestCartRepository_Get_NotFound(t *testing.T) {
	repo, _ := setupTestRedis(t)

	got, err := repo.Get(context.Background(), "nonexistent-user")
	assert.Nil(t, got)
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

func TestCartRepository_Get_InvalidJSON(t *testing.T) {
	repo, mr := setupTestRedis(t)

	// Set corrupted JSON data.
	require.NoError(t, mr.Set("cart:user-bad", "{{not-valid-json"))

	got, err := repo.Get(context.Background(), "user-bad")
	assert.Nil(t, got)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal cart")
}

// ---------------------------------------------------------------------------
// Save
// ---------------------------------------------------------------------------

func TestCartRepository_Save_Success(t *testing.T) {
	repo, mr := setupTestRedis(t)

	cart := sampleCart()
	err := repo.Save(context.Background(), cart)
	require.NoError(t, err)

	// Verify key exists in Redis.
	assert.True(t, mr.Exists("cart:"+cart.UserID))

	// Verify JSON content.
	raw, err := mr.Get("cart:" + cart.UserID)
	require.NoError(t, err)

	var stored domain.Cart
	require.NoError(t, json.Unmarshal([]byte(raw), &stored))
	assert.Equal(t, cart.ID, stored.ID)
	assert.Equal(t, cart.UserID, stored.UserID)
	assert.Equal(t, cart.Currency, stored.Currency)
	assert.Equal(t, cart.Version, stored.Version)
	require.Len(t, stored.Items, 1)
	assert.Equal(t, "prod-1", stored.Items[0].ProductID)
}

func TestCartRepository_Save_TTL(t *testing.T) {
	repo, mr := setupTestRedis(t)

	cart := sampleCart()
	err := repo.Save(context.Background(), cart)
	require.NoError(t, err)

	ttl := mr.TTL("cart:" + cart.UserID)
	// TTL should be approximately 24 hours (allow some margin for test execution).
	assert.True(t, ttl > 23*time.Hour, "expected TTL > 23h, got %v", ttl)
	assert.True(t, ttl <= 24*time.Hour, "expected TTL <= 24h, got %v", ttl)
}

// ---------------------------------------------------------------------------
// SaveIfVersion
// ---------------------------------------------------------------------------

func TestCartRepository_SaveIfVersion_Success(t *testing.T) {
	repo, _ := setupTestRedis(t)

	cart := sampleCart()
	cart.Version = 1

	// First, save the cart normally.
	err := repo.Save(context.Background(), cart)
	require.NoError(t, err)

	// SaveIfVersion with correct expected version.
	cart.Items = append(cart.Items, domain.CartItem{
		ProductID: "prod-2",
		VariantID: "var-2",
		Name:      "Gadget",
		SKU:       "GDG-1",
		Price:     4500,
		Quantity:  1,
	})

	ok, err := repo.SaveIfVersion(context.Background(), cart, 1)
	require.NoError(t, err)
	assert.True(t, ok)

	// Verify version was incremented.
	got, err := repo.Get(context.Background(), cart.UserID)
	require.NoError(t, err)
	assert.Equal(t, 2, got.Version)
	assert.Len(t, got.Items, 2)
}

func TestCartRepository_SaveIfVersion_VersionMismatch(t *testing.T) {
	repo, _ := setupTestRedis(t)

	cart := sampleCart()
	cart.Version = 1

	// Save the cart.
	err := repo.Save(context.Background(), cart)
	require.NoError(t, err)

	// Attempt SaveIfVersion with wrong expected version.
	ok, err := repo.SaveIfVersion(context.Background(), cart, 99)
	require.NoError(t, err)
	assert.False(t, ok)

	// Verify original data unchanged.
	got, err := repo.Get(context.Background(), cart.UserID)
	require.NoError(t, err)
	assert.Equal(t, 1, got.Version)
}

func TestCartRepository_SaveIfVersion_NewCart(t *testing.T) {
	repo, _ := setupTestRedis(t)

	cart := sampleCart()
	cart.Version = 0

	// SaveIfVersion with expectedVersion=0 when key doesn't exist should succeed.
	ok, err := repo.SaveIfVersion(context.Background(), cart, 0)
	require.NoError(t, err)
	assert.True(t, ok)

	// Verify version was set to 1.
	got, err := repo.Get(context.Background(), cart.UserID)
	require.NoError(t, err)
	assert.Equal(t, 1, got.Version)
}

func TestCartRepository_SaveIfVersion_NewCartVersionMismatch(t *testing.T) {
	repo, _ := setupTestRedis(t)

	cart := sampleCart()

	// SaveIfVersion with expectedVersion=5 when key doesn't exist should fail.
	ok, err := repo.SaveIfVersion(context.Background(), cart, 5)
	require.NoError(t, err)
	assert.False(t, ok)

	// Verify key was not created.
	_, err = repo.Get(context.Background(), cart.UserID)
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestCartRepository_Delete_Success(t *testing.T) {
	repo, mr := setupTestRedis(t)

	cart := sampleCart()
	err := repo.Save(context.Background(), cart)
	require.NoError(t, err)
	assert.True(t, mr.Exists("cart:"+cart.UserID))

	err = repo.Delete(context.Background(), cart.UserID)
	require.NoError(t, err)

	// Verify key was removed.
	assert.False(t, mr.Exists("cart:"+cart.UserID))
}

func TestCartRepository_Delete_NonExistent(t *testing.T) {
	repo, _ := setupTestRedis(t)

	// Deleting a key that doesn't exist should not return an error.
	err := repo.Delete(context.Background(), "nonexistent-user")
	assert.NoError(t, err)
}
