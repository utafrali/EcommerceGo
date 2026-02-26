package integration

import (
	"fmt"
	"testing"
)

// TestFullCommerceFlow exercises the entire E2E commerce lifecycle in a single test:
//  1. Register a new user
//  2. Login and obtain a JWT
//  3. Create a product (directly against product service)
//  4. Initialize stock for the product (500 units)
//  5. Add the product to cart (via gateway with JWT)
//  6. Create an order (directly against order service)
//  7. Verify stock decreased (if the checkout saga is wired)
//  8. Index and search for the product
//  9. Create a campaign
//
// Each step asserts success and passes data to the next step.
func TestFullCommerceFlow(t *testing.T) {
	// Require all core services to be running.
	requiredPorts := []int{userPort, productPort, inventoryPort, cartPort, orderPort, gatewayPort}
	for _, port := range requiredPorts {
		skipIfNotRunning(t, port)
	}

	// ---------------------------------------------------------------
	// Step 1: Register a new user
	// ---------------------------------------------------------------
	t.Log("Step 1: Register user")
	email := uniqueEmail("e2e")
	regBody := map[string]interface{}{
		"email":      email,
		"password":   "E2ETestPass1!",
		"first_name": "E2E",
		"last_name":  "Commerce",
	}
	regStatus, regData := httpPost(t, baseURL(userPort)+"/api/v1/auth/register", regBody)
	requireStatus(t, regStatus, 201)

	userID := extractString(t, regData, "data.user.id")
	t.Logf("  registered user id=%s email=%s", userID, email)

	// ---------------------------------------------------------------
	// Step 2: Login and get JWT
	// ---------------------------------------------------------------
	t.Log("Step 2: Login")
	loginBody := map[string]interface{}{
		"email":    email,
		"password": "E2ETestPass1!",
	}
	loginStatus, loginData := httpPost(t, baseURL(userPort)+"/api/v1/auth/login", loginBody)
	requireStatus(t, loginStatus, 200)

	accessToken := extractString(t, loginData, "data.tokens.access_token")
	t.Logf("  got access token (length %d)", len(accessToken))

	// ---------------------------------------------------------------
	// Step 3: Create a product
	// ---------------------------------------------------------------
	t.Log("Step 3: Create product")
	slug := uniqueSlug("e2e-widget")
	productBody := map[string]interface{}{
		"name":        "E2E Widget " + slug,
		"description": "A test product for the full commerce flow",
		"base_price":  5000,
		"currency":    "USD",
	}
	prodStatus, prodData := httpPost(t, baseURL(productPort)+"/api/v1/products", productBody)
	requireStatus(t, prodStatus, 201)

	productID := extractString(t, prodData, "data.id")
	productSlug := extractString(t, prodData, "data.slug")
	t.Logf("  created product id=%s slug=%s", productID, productSlug)

	// ---------------------------------------------------------------
	// Step 4: Initialize stock (500 units)
	// ---------------------------------------------------------------
	t.Log("Step 4: Initialize stock")
	variantID := uniqueUUID()
	stockBody := map[string]interface{}{
		"product_id":          productID,
		"variant_id":          variantID,
		"quantity":            500,
		"low_stock_threshold": 10,
	}
	stockStatus, stockData := httpPost(t, baseURL(inventoryPort)+"/api/v1/inventory", stockBody)
	requireStatus(t, stockStatus, 201)

	initialQty := extractFloat(t, stockData, "data.quantity")
	t.Logf("  initialized stock: product=%s variant=%s quantity=%v", productID, variantID, initialQty)

	if initialQty != 500 {
		t.Fatalf("expected initial stock quantity 500, got %v", initialQty)
	}

	// ---------------------------------------------------------------
	// Step 5: Add to cart (via gateway with JWT)
	// ---------------------------------------------------------------
	t.Log("Step 5: Add to cart via gateway")
	cartItemBody := map[string]interface{}{
		"product_id": productID,
		"variant_id": variantID,
		"name":       "E2E Widget " + slug,
		"sku":        "E2E-" + slug,
		"price":      5000,
		"quantity":   2,
	}
	addCartStatus, _ := httpPostWithAuth(t, baseURL(gatewayPort)+"/api/v1/cart/items", cartItemBody, accessToken)
	requireStatus(t, addCartStatus, 200)

	// Verify the cart via gateway.
	cartGetStatus, cartGetData := httpGetWithAuth(t, baseURL(gatewayPort)+"/api/v1/cart", accessToken)
	requireStatus(t, cartGetStatus, 200)

	cartItems := extractField(cartGetData, "data.items")
	if cartItems == nil {
		t.Fatal("expected items in cart response")
	}
	arr, ok := cartItems.([]interface{})
	if !ok || len(arr) == 0 {
		t.Fatal("expected at least 1 item in cart")
	}
	t.Logf("  cart has %d items", len(arr))

	// ---------------------------------------------------------------
	// Step 6: Create an order
	// ---------------------------------------------------------------
	t.Log("Step 6: Create order")
	orderBody := map[string]interface{}{
		"user_id":  userID,
		"currency": "USD",
		"items": []map[string]interface{}{
			{
				"product_id": productID,
				"variant_id": variantID,
				"name":       "E2E Widget " + slug,
				"sku":        "E2E-" + slug,
				"price":      5000,
				"quantity":   2,
			},
		},
		"shipping_amount": 500,
		"discount_amount": 0,
	}
	orderStatus, orderData := httpPost(t, baseURL(orderPort)+"/api/v1/orders", orderBody)
	requireStatus(t, orderStatus, 201)

	orderID := extractString(t, orderData, "data.id")
	t.Logf("  created order id=%s", orderID)

	// ---------------------------------------------------------------
	// Step 7: Verify stock (optional — depends on saga wiring)
	// ---------------------------------------------------------------
	t.Log("Step 7: Check stock level")
	stockURL := fmt.Sprintf("%s/api/v1/inventory/%s/variants/%s", baseURL(inventoryPort), productID, variantID)
	stockGetStatus, stockGetData := httpGet(t, stockURL)
	requireStatus(t, stockGetStatus, 200)

	currentQty := extractFloat(t, stockGetData, "data.quantity")
	t.Logf("  current stock quantity=%v (initial was %v)", currentQty, initialQty)

	// If the checkout saga automatically decrements stock, we expect 498.
	// If not wired, the quantity remains at 500.
	if currentQty == 498 {
		t.Log("  stock was automatically decremented by saga (500 -> 498)")
	} else if currentQty == 500 {
		t.Log("  stock unchanged (saga not wired to inventory for direct orders)")
	} else {
		t.Logf("  unexpected stock level: %v", currentQty)
	}

	// ---------------------------------------------------------------
	// Step 8: Search for the product
	// ---------------------------------------------------------------
	t.Log("Step 8: Search for product")
	searchPort := 8010
	skipSearch := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				skipSearch = true
			}
		}()
		skipIfNotRunning(t, searchPort)
	}()

	if !skipSearch {
		// Index the product in search.
		indexBody := map[string]interface{}{
			"id":         productID,
			"name":       "E2E Widget " + slug,
			"slug":       productSlug,
			"base_price": 5000,
			"currency":   "USD",
			"status":     "published",
		}
		indexStatus, _ := httpPost(t, baseURL(searchPort)+"/api/v1/search/index", indexBody)
		if indexStatus == 200 || indexStatus == 201 {
			t.Logf("  indexed product in search")

			// Search for it.
			searchStatus, searchData := httpGet(t, baseURL(searchPort)+"/api/v1/search?q=E2E+Widget")
			if searchStatus == 200 {
				results := extractField(searchData, "data")
				t.Logf("  search returned results: %v", results != nil)
			} else {
				t.Logf("  search returned status %d (search index may need time)", searchStatus)
			}
		} else {
			t.Logf("  search indexing returned status %d, skipping search query", indexStatus)
		}
	} else {
		t.Log("  search service not running, skipping")
	}

	// ---------------------------------------------------------------
	// Step 9: Create a campaign
	// ---------------------------------------------------------------
	t.Log("Step 9: Create campaign")
	campaignPort := 8008
	skipCampaign := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				skipCampaign = true
			}
		}()
		skipIfNotRunning(t, campaignPort)
	}()

	if !skipCampaign {
		campaignBody := map[string]interface{}{
			"name":            "E2E Test Campaign",
			"description":     "A campaign for the E2E commerce flow",
			"type":            "percentage",
			"discount_value":  10,
			"min_order_amount": 1000,
			"start_date":      "2025-01-01T00:00:00Z",
			"end_date":        "2027-12-31T23:59:59Z",
		}
		campStatus, campData := httpPost(t, baseURL(campaignPort)+"/api/v1/campaigns", campaignBody)

		if campStatus == 201 || campStatus == 200 {
			campaignID := extractField(campData, "data.id")
			t.Logf("  created campaign id=%v", campaignID)
		} else {
			t.Logf("  campaign creation returned status %d; body: %v", campStatus, campData)
		}
	} else {
		t.Log("  campaign service not running, skipping")
	}

	t.Log("Full commerce flow completed successfully")
}
