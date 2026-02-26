package integration

import (
	"testing"
)

const cartPort = 8002

// TestAddItemToCart verifies that an item can be added to a cart.
func TestAddItemToCart(t *testing.T) {
	skipIfNotRunning(t, cartPort)

	userID := uniqueUUID()
	productID := uniqueUUID()
	variantID := uniqueUUID()

	body := map[string]interface{}{
		"product_id": productID,
		"variant_id": variantID,
		"name":       "Test Widget",
		"sku":        "TST-WIDGET-001",
		"price":      2999,
		"quantity":   2,
		"image_url":  "https://example.com/widget.png",
	}

	headers := map[string]string{
		"X-User-ID": userID,
	}

	status, data := httpPostWithHeaders(t, baseURL(cartPort)+"/api/v1/cart/items", body, headers)
	requireStatus(t, status, 200)

	// Verify we got cart data back.
	cartData := extractField(data, "data")
	if cartData == nil {
		t.Fatal("expected data in add-to-cart response, got nil")
	}

	t.Logf("added item to cart for user %s", userID)
}

// TestViewCart verifies that a cart can be retrieved.
func TestViewCart(t *testing.T) {
	skipIfNotRunning(t, cartPort)

	userID := uniqueUUID()
	productID := uniqueUUID()
	variantID := uniqueUUID()

	// First add an item.
	addBody := map[string]interface{}{
		"product_id": productID,
		"variant_id": variantID,
		"name":       "View Cart Widget",
		"sku":        "VCW-001",
		"price":      1999,
		"quantity":   1,
	}
	addHeaders := map[string]string{"X-User-ID": userID}
	addStatus, _ := httpPostWithHeaders(t, baseURL(cartPort)+"/api/v1/cart/items", addBody, addHeaders)
	requireStatus(t, addStatus, 200)

	// Now view the cart.
	getHeaders := map[string]string{"X-User-ID": userID}
	getStatus, getData := httpGetWithHeaders(t, baseURL(cartPort)+"/api/v1/cart", getHeaders)
	requireStatus(t, getStatus, 200)

	cartData := extractField(getData, "data")
	if cartData == nil {
		t.Fatal("expected data in get-cart response, got nil")
	}

	t.Logf("viewed cart for user %s: %v", userID, cartData)
}

// TestAddMultipleItems verifies that multiple items can be added to a cart.
func TestAddMultipleItems(t *testing.T) {
	skipIfNotRunning(t, cartPort)

	userID := uniqueUUID()
	headers := map[string]string{"X-User-ID": userID}

	// Add first item.
	item1 := map[string]interface{}{
		"product_id": uniqueUUID(),
		"variant_id": uniqueUUID(),
		"name":       "Widget A",
		"sku":        "WA-001",
		"price":      1000,
		"quantity":   1,
	}
	s1, _ := httpPostWithHeaders(t, baseURL(cartPort)+"/api/v1/cart/items", item1, headers)
	requireStatus(t, s1, 200)

	// Add second item.
	item2 := map[string]interface{}{
		"product_id": uniqueUUID(),
		"variant_id": uniqueUUID(),
		"name":       "Widget B",
		"sku":        "WB-001",
		"price":      2000,
		"quantity":   3,
	}
	s2, _ := httpPostWithHeaders(t, baseURL(cartPort)+"/api/v1/cart/items", item2, headers)
	requireStatus(t, s2, 200)

	// View the cart.
	getStatus, getData := httpGetWithHeaders(t, baseURL(cartPort)+"/api/v1/cart", headers)
	requireStatus(t, getStatus, 200)

	// Verify cart has items.
	items := extractField(getData, "data.items")
	if items == nil {
		t.Fatal("expected data.items in cart response, got nil")
	}
	arr, ok := items.([]interface{})
	if !ok {
		t.Fatalf("expected items to be an array, got %T", items)
	}
	if len(arr) < 2 {
		t.Fatalf("expected at least 2 items in cart, got %d", len(arr))
	}

	t.Logf("cart has %d items after adding 2", len(arr))
}

// TestCartEmptyInitially verifies that a new user's cart is empty.
func TestCartEmptyInitially(t *testing.T) {
	skipIfNotRunning(t, cartPort)

	// Use a fresh user ID that has never had items added.
	userID := uniqueUUID()
	headers := map[string]string{"X-User-ID": userID}

	status, data := httpGetWithHeaders(t, baseURL(cartPort)+"/api/v1/cart", headers)
	requireStatus(t, status, 200)

	// The cart should exist but have no items (or an empty items array).
	items := extractField(data, "data.items")
	if items != nil {
		arr, ok := items.([]interface{})
		if ok && len(arr) > 0 {
			t.Fatalf("expected empty cart for new user, got %d items", len(arr))
		}
	}

	t.Logf("new user %s has an empty cart as expected", userID)
}

// TestCartRequiresUserID verifies that cart endpoints require the X-User-ID header.
func TestCartRequiresUserID(t *testing.T) {
	skipIfNotRunning(t, cartPort)

	// Try to get cart without X-User-ID.
	status, data := httpGet(t, baseURL(cartPort)+"/api/v1/cart")

	if status != 400 {
		t.Fatalf("expected status 400 when X-User-ID is missing, got %d; body: %v", status, data)
	}
}
