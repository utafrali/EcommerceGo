package integration

import (
	"net/http"
	"testing"
	"time"
)

const gatewayPort = 8080

// TestGatewayPublicRoutes verifies that public endpoints (products, search)
// are accessible through the gateway without authentication.
func TestGatewayPublicRoutes(t *testing.T) {
	skipIfNotRunning(t, gatewayPort)

	tests := []struct {
		name string
		url  string
	}{
		{"products_list", "/api/v1/products"},
		{"search", "/api/v1/search?q=test"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, _ := httpGet(t, baseURL(gatewayPort)+tc.url)
			if status != 200 {
				t.Fatalf("expected 200 for public route %s, got %d", tc.url, status)
			}
		})
	}
}

// TestGatewayProtectedRouteNoAuth verifies that protected endpoints (cart)
// return 401 when no JWT is provided.
func TestGatewayProtectedRouteNoAuth(t *testing.T) {
	skipIfNotRunning(t, gatewayPort)

	status, data := httpGet(t, baseURL(gatewayPort)+"/api/v1/cart")

	if status != 401 {
		t.Fatalf("expected 401 for cart without auth, got %d; body: %v", status, data)
	}
}

// TestGatewayProtectedRouteWithAuth verifies that protected endpoints are
// accessible when a valid JWT is provided.
func TestGatewayProtectedRouteWithAuth(t *testing.T) {
	skipIfNotRunning(t, gatewayPort)
	skipIfNotRunning(t, userPort)

	// Register and login to get a JWT.
	_, token := registerAndLogin(t)

	// Access cart through the gateway with the JWT.
	status, _ := httpGetWithAuth(t, baseURL(gatewayPort)+"/api/v1/cart", token)

	// Should get 200 (empty cart) instead of 401.
	if status != 200 {
		t.Fatalf("expected 200 for cart with valid auth, got %d", status)
	}
}

// TestGatewayJWTForwarding verifies that the gateway extracts user context from the JWT
// and forwards it to downstream services via X-User-ID header.
func TestGatewayJWTForwarding(t *testing.T) {
	skipIfNotRunning(t, gatewayPort)
	skipIfNotRunning(t, userPort)
	skipIfNotRunning(t, cartPort)

	// Register, login, and get user ID + JWT.
	userID, token := registerAndLogin(t)

	// Add an item to cart through the gateway.
	addBody := map[string]interface{}{
		"product_id": uniqueUUID(),
		"variant_id": uniqueUUID(),
		"name":       "Gateway Forward Widget",
		"sku":        "GFW-001",
		"price":      3500,
		"quantity":   1,
	}
	addStatus, _ := httpPostWithAuth(t, baseURL(gatewayPort)+"/api/v1/cart/items", addBody, token)
	requireStatus(t, addStatus, 200)

	// Verify the cart is associated with the correct user by reading it back.
	getStatus, getData := httpGetWithAuth(t, baseURL(gatewayPort)+"/api/v1/cart", token)
	requireStatus(t, getStatus, 200)

	// The cart response should contain items for the authenticated user.
	items := extractField(getData, "data.items")
	if items == nil {
		t.Fatal("expected data.items in cart response through gateway")
	}

	arr, ok := items.([]interface{})
	if !ok {
		t.Fatalf("expected items to be an array, got %T", items)
	}
	if len(arr) == 0 {
		t.Fatal("expected at least 1 item in cart after adding through gateway")
	}

	t.Logf("gateway correctly forwarded JWT for user %s, cart has %d items", userID, len(arr))
}

// TestGatewaySpoofedHeaderRejected verifies that the gateway strips incoming
// X-User-ID headers and rejects requests that try to spoof user identity
// without a valid JWT.
func TestGatewaySpoofedHeaderRejected(t *testing.T) {
	skipIfNotRunning(t, gatewayPort)

	// Try to access a protected route with a spoofed X-User-ID but no JWT.
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, baseURL(gatewayPort)+"/api/v1/cart", nil)
	if err != nil {
		t.Fatalf("creating request failed: %v", err)
	}
	// Set X-User-ID directly without a JWT — the gateway should strip it and reject.
	req.Header.Set("X-User-ID", uniqueUUID())

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Fatalf("expected 401 for spoofed X-User-ID without JWT, got %d", resp.StatusCode)
	}

	t.Log("spoofed X-User-ID header correctly rejected without JWT")
}

// TestGatewayHealthCheck verifies the gateway's own health endpoint.
func TestGatewayHealthCheck(t *testing.T) {
	skipIfNotRunning(t, gatewayPort)

	status, _ := httpGet(t, baseURL(gatewayPort)+"/health/live")
	requireStatus(t, status, 200)
}

// TestGatewayAuthPassthrough verifies that auth endpoints (register/login)
// are accessible through the gateway as public routes.
func TestGatewayAuthPassthrough(t *testing.T) {
	skipIfNotRunning(t, gatewayPort)
	skipIfNotRunning(t, userPort)

	email := uniqueEmail("gw-auth")
	regBody := map[string]interface{}{
		"email":      email,
		"password":   "TestPass123!",
		"first_name": "GW",
		"last_name":  "Auth",
	}

	// Register through gateway — /api/v1/auth/register is a public POST route.
	status, data := httpPost(t, baseURL(gatewayPort)+"/api/v1/auth/register", regBody)

	// The gateway proxies to the user service.
	if status != 201 {
		t.Fatalf("expected 201 for registration through gateway, got %d; body: %v", status, data)
	}

	t.Logf("registered user %s through gateway", email)
}
