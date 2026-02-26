package integration

import (
	"testing"
)

const inventoryPort = 8007

// TestInitializeStock verifies that stock can be initialized for a product variant.
func TestInitializeStock(t *testing.T) {
	skipIfNotRunning(t, inventoryPort)

	productID := uniqueUUID()
	variantID := uniqueUUID()

	body := map[string]interface{}{
		"product_id":          productID,
		"variant_id":          variantID,
		"quantity":            500,
		"low_stock_threshold": 10,
	}

	status, data := httpPost(t, baseURL(inventoryPort)+"/api/v1/inventory", body)
	requireStatus(t, status, 201)

	// Verify we got stock data back.
	stockProdID := extractField(data, "data.product_id")
	if stockProdID == nil {
		t.Fatal("expected data.product_id in stock response, got nil")
	}

	qty := extractField(data, "data.quantity")
	if qty == nil {
		t.Fatal("expected data.quantity in stock response, got nil")
	}

	t.Logf("initialized stock for product=%s variant=%s quantity=%v", productID, variantID, qty)
}

// TestAdjustStock verifies that stock quantity can be adjusted with a delta.
func TestAdjustStock(t *testing.T) {
	skipIfNotRunning(t, inventoryPort)

	productID := uniqueUUID()
	variantID := uniqueUUID()

	// Initialize stock.
	initBody := map[string]interface{}{
		"product_id":          productID,
		"variant_id":          variantID,
		"quantity":            100,
		"low_stock_threshold": 5,
	}
	initStatus, _ := httpPost(t, baseURL(inventoryPort)+"/api/v1/inventory", initBody)
	requireStatus(t, initStatus, 201)

	// Adjust stock down by 10.
	adjustBody := map[string]interface{}{
		"delta":  -10,
		"reason": "adjustment",
	}
	adjustURL := baseURL(inventoryPort) + "/api/v1/inventory/" + productID + "/variants/" + variantID
	adjustStatus, adjustData := httpPut(t, adjustURL, adjustBody)
	requireStatus(t, adjustStatus, 200)

	// Verify the quantity is reduced.
	newQty := extractFloat(t, adjustData, "data.quantity")
	if newQty != 90 {
		t.Fatalf("expected quantity 90 after adjustment, got %v", newQty)
	}

	t.Logf("adjusted stock: quantity is now %v", newQty)
}

// TestGetStock verifies that stock levels can be queried.
func TestGetStock(t *testing.T) {
	skipIfNotRunning(t, inventoryPort)

	productID := uniqueUUID()
	variantID := uniqueUUID()

	// Initialize stock.
	initBody := map[string]interface{}{
		"product_id":          productID,
		"variant_id":          variantID,
		"quantity":            250,
		"low_stock_threshold": 20,
	}
	initStatus, _ := httpPost(t, baseURL(inventoryPort)+"/api/v1/inventory", initBody)
	requireStatus(t, initStatus, 201)

	// Get stock.
	getURL := baseURL(inventoryPort) + "/api/v1/inventory/" + productID + "/variants/" + variantID
	getStatus, getData := httpGet(t, getURL)
	requireStatus(t, getStatus, 200)

	qty := extractFloat(t, getData, "data.quantity")
	if qty != 250 {
		t.Fatalf("expected quantity 250, got %v", qty)
	}
}

// TestAdjustStockUpsert verifies that adjusting stock on a non-existent record
// either creates it (upsert) or returns a meaningful error. The behavior depends
// on the service implementation.
func TestAdjustStockUpsert(t *testing.T) {
	skipIfNotRunning(t, inventoryPort)

	productID := uniqueUUID()
	variantID := uniqueUUID()

	// Try to adjust a product that has no stock record.
	adjustBody := map[string]interface{}{
		"delta":  50,
		"reason": "adjustment",
	}
	adjustURL := baseURL(inventoryPort) + "/api/v1/inventory/" + productID + "/variants/" + variantID
	status, data := httpPut(t, adjustURL, adjustBody)

	// Accept either 200 (upsert succeeded) or 404 (not found).
	if status != 200 && status != 404 {
		t.Fatalf("expected status 200 (upsert) or 404 (not found) for adjust on nonexistent stock, got %d; body: %v", status, data)
	}

	if status == 200 {
		// If upserted, verify the quantity.
		qty := extractFloat(t, data, "data.quantity")
		if qty != 50 {
			t.Fatalf("expected upserted quantity 50, got %v", qty)
		}
		t.Log("stock was upserted successfully")
	} else {
		t.Log("adjust on nonexistent stock correctly returned 404")
	}
}

// TestInitializeStockValidation verifies that initializing stock with invalid data
// returns a 400 error.
func TestInitializeStockValidation(t *testing.T) {
	skipIfNotRunning(t, inventoryPort)

	// Missing product_id and variant_id (required UUIDs).
	body := map[string]interface{}{
		"quantity": 100,
	}

	status, data := httpPost(t, baseURL(inventoryPort)+"/api/v1/inventory", body)
	if status != 400 {
		t.Fatalf("expected status 400 for invalid inventory init, got %d; body: %v", status, data)
	}
}
