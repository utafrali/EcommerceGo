package integration

import (
	"testing"
)

const productPort = 8001

// TestCreateProduct verifies that a product can be created via POST.
func TestCreateProduct(t *testing.T) {
	skipIfNotRunning(t, productPort)

	slug := uniqueSlug("test-product")
	body := map[string]interface{}{
		"name":        "Integration Test Product " + slug,
		"description": "A product created by integration tests",
		"base_price":  4999,
		"currency":    "USD",
	}

	status, data := httpPost(t, baseURL(productPort)+"/api/v1/products", body)

	// The product service returns 201 on creation.
	requireStatus(t, status, 201)

	productID := extractField(data, "data.id")
	if productID == nil {
		t.Fatal("expected data.id in create product response, got nil")
	}

	productSlug := extractField(data, "data.slug")
	if productSlug == nil {
		t.Fatal("expected data.slug in create product response, got nil")
	}

	t.Logf("created product id=%v slug=%v", productID, productSlug)
}

// TestGetProductByUUID verifies that a product can be retrieved by its UUID.
func TestGetProductByUUID(t *testing.T) {
	skipIfNotRunning(t, productPort)

	// Create a product first.
	slug := uniqueSlug("get-uuid")
	body := map[string]interface{}{
		"name":        "Get By UUID Product " + slug,
		"description": "Test product for UUID lookup",
		"base_price":  2999,
		"currency":    "USD",
	}
	createStatus, createData := httpPost(t, baseURL(productPort)+"/api/v1/products", body)
	requireStatus(t, createStatus, 201)

	productID := extractString(t, createData, "data.id")

	// Retrieve by UUID.
	getStatus, getData := httpGet(t, baseURL(productPort)+"/api/v1/products/"+productID)
	requireStatus(t, getStatus, 200)

	retrievedID := extractString(t, getData, "data.id")
	if retrievedID != productID {
		t.Fatalf("expected product id %s, got %s", productID, retrievedID)
	}
}

// TestGetProductBySlug verifies that a product can be retrieved by its slug.
func TestGetProductBySlug(t *testing.T) {
	skipIfNotRunning(t, productPort)

	// Create a product first.
	slug := uniqueSlug("get-slug")
	body := map[string]interface{}{
		"name":        "Get By Slug Product " + slug,
		"description": "Test product for slug lookup",
		"base_price":  1999,
		"currency":    "USD",
	}
	createStatus, createData := httpPost(t, baseURL(productPort)+"/api/v1/products", body)
	requireStatus(t, createStatus, 201)

	productSlug := extractString(t, createData, "data.slug")

	// Retrieve by slug.
	getStatus, getData := httpGet(t, baseURL(productPort)+"/api/v1/products/"+productSlug)
	requireStatus(t, getStatus, 200)

	retrievedSlug := extractString(t, getData, "data.slug")
	if retrievedSlug != productSlug {
		t.Fatalf("expected product slug %s, got %s", productSlug, retrievedSlug)
	}
}

// TestListProducts verifies that the product listing endpoint returns data.
func TestListProducts(t *testing.T) {
	skipIfNotRunning(t, productPort)

	// Ensure at least one product exists.
	slug := uniqueSlug("list")
	body := map[string]interface{}{
		"name":        "List Product " + slug,
		"description": "Product for listing test",
		"base_price":  999,
		"currency":    "USD",
	}
	createStatus, _ := httpPost(t, baseURL(productPort)+"/api/v1/products", body)
	requireStatus(t, createStatus, 201)

	// List products.
	status, data := httpGet(t, baseURL(productPort)+"/api/v1/products")
	requireStatus(t, status, 200)

	// The list response uses a top-level "data" array and "total_count".
	products := extractField(data, "data")
	if products == nil {
		t.Fatal("expected data field in list products response, got nil")
	}

	arr, ok := products.([]interface{})
	if !ok {
		t.Fatalf("expected data to be an array, got %T", products)
	}
	if len(arr) == 0 {
		t.Fatal("expected at least one product in list, got empty array")
	}

	t.Logf("listed %d products", len(arr))
}

// TestCreateProductValidation verifies that creating a product with missing required
// fields returns a 400 error.
func TestCreateProductValidation(t *testing.T) {
	skipIfNotRunning(t, productPort)

	// Missing name and currency.
	body := map[string]interface{}{
		"base_price": 100,
	}

	status, data := httpPost(t, baseURL(productPort)+"/api/v1/products", body)

	if status != 400 {
		t.Fatalf("expected status 400 for invalid product, got %d; body: %v", status, data)
	}
}

// createTestProduct is a helper that creates a product and returns its ID and slug.
func createTestProduct(t *testing.T) (productID, productSlug string) {
	t.Helper()
	skipIfNotRunning(t, productPort)

	slug := uniqueSlug("helper")
	body := map[string]interface{}{
		"name":        "Helper Product " + slug,
		"description": "Created by test helper",
		"base_price":  5000,
		"currency":    "USD",
	}

	status, data := httpPost(t, baseURL(productPort)+"/api/v1/products", body)
	requireStatus(t, status, 201)

	productID = extractString(t, data, "data.id")
	productSlug = extractString(t, data, "data.slug")
	return productID, productSlug
}
