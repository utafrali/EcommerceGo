package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Cart.TotalAmount Tests
// ============================================================================

func TestTotalAmount_SingleItem(t *testing.T) {
	c := &Cart{
		Items: []CartItem{
			{Price: 1999, Quantity: 2},
		},
	}
	assert.Equal(t, int64(3998), c.TotalAmount())
}

func TestTotalAmount_MultipleItems(t *testing.T) {
	c := &Cart{
		Items: []CartItem{
			{Price: 1000, Quantity: 2},
			{Price: 500, Quantity: 3},
			{Price: 2500, Quantity: 1},
		},
	}
	// 2000 + 1500 + 2500 = 6000
	assert.Equal(t, int64(6000), c.TotalAmount())
}

func TestTotalAmount_EmptyCart(t *testing.T) {
	c := &Cart{Items: []CartItem{}}
	assert.Equal(t, int64(0), c.TotalAmount())
}

func TestTotalAmount_NilItems(t *testing.T) {
	c := &Cart{}
	assert.Equal(t, int64(0), c.TotalAmount())
}

func TestTotalAmount_ZeroPrice(t *testing.T) {
	c := &Cart{
		Items: []CartItem{
			{Price: 0, Quantity: 5},
		},
	}
	assert.Equal(t, int64(0), c.TotalAmount())
}

func TestTotalAmount_ZeroQuantity(t *testing.T) {
	c := &Cart{
		Items: []CartItem{
			{Price: 1000, Quantity: 0},
		},
	}
	assert.Equal(t, int64(0), c.TotalAmount())
}

// ============================================================================
// Cart.ItemCount Tests
// ============================================================================

func TestItemCount_MultipleItems(t *testing.T) {
	c := &Cart{
		Items: []CartItem{
			{Quantity: 2},
			{Quantity: 3},
			{Quantity: 1},
		},
	}
	assert.Equal(t, 6, c.ItemCount())
}

func TestItemCount_EmptyCart(t *testing.T) {
	c := &Cart{Items: []CartItem{}}
	assert.Equal(t, 0, c.ItemCount())
}

func TestItemCount_SingleItem(t *testing.T) {
	c := &Cart{
		Items: []CartItem{{Quantity: 5}},
	}
	assert.Equal(t, 5, c.ItemCount())
}

// ============================================================================
// Cart.FindItemIndex Tests
// ============================================================================

func TestFindItemIndex_Found(t *testing.T) {
	c := &Cart{
		Items: []CartItem{
			{ProductID: "prod-1", VariantID: "var-1"},
			{ProductID: "prod-2", VariantID: "var-2"},
		},
	}
	assert.Equal(t, 0, c.FindItemIndex("prod-1", "var-1"))
	assert.Equal(t, 1, c.FindItemIndex("prod-2", "var-2"))
}

func TestFindItemIndex_NotFound(t *testing.T) {
	c := &Cart{
		Items: []CartItem{
			{ProductID: "prod-1", VariantID: "var-1"},
		},
	}
	assert.Equal(t, -1, c.FindItemIndex("prod-999", "var-999"))
}

func TestFindItemIndex_EmptyCart(t *testing.T) {
	c := &Cart{Items: []CartItem{}}
	assert.Equal(t, -1, c.FindItemIndex("prod-1", "var-1"))
}

func TestFindItemIndex_ProductMatchVariantMismatch(t *testing.T) {
	c := &Cart{
		Items: []CartItem{
			{ProductID: "prod-1", VariantID: "var-1"},
		},
	}
	assert.Equal(t, -1, c.FindItemIndex("prod-1", "var-2"))
}

func TestFindItemIndex_VariantMatchProductMismatch(t *testing.T) {
	c := &Cart{
		Items: []CartItem{
			{ProductID: "prod-1", VariantID: "var-1"},
		},
	}
	assert.Equal(t, -1, c.FindItemIndex("prod-2", "var-1"))
}

// ============================================================================
// Cart Struct Tests
// ============================================================================

func TestCart_VersionForOptimisticLocking(t *testing.T) {
	c := &Cart{Version: 3}
	assert.Equal(t, 3, c.Version)
}

func TestCartItem_PriceInCents(t *testing.T) {
	item := CartItem{Price: 9999, Quantity: 1}
	assert.Equal(t, int64(9999), item.Price)
}
