package integration

import (
	"testing"
)

const userPort = 8006

// TestUserRegistration verifies that a new user can register successfully.
// It expects a 201 response with user data and tokens in the body.
func TestUserRegistration(t *testing.T) {
	skipIfNotRunning(t, userPort)

	email := uniqueEmail("register")
	body := map[string]interface{}{
		"email":      email,
		"password":   "TestPass123!",
		"first_name": "Integration",
		"last_name":  "Test",
	}

	status, data := httpPost(t, baseURL(userPort)+"/api/v1/auth/register", body)
	requireStatus(t, status, 201)

	// Verify we got user data back.
	userID := extractField(data, "data.user.id")
	if userID == nil {
		t.Fatal("expected data.user.id in registration response, got nil")
	}

	tokens := extractField(data, "data.tokens")
	if tokens == nil {
		t.Fatal("expected data.tokens in registration response, got nil")
	}

	t.Logf("registered user %s with id %v", email, userID)
}

// TestUserLogin verifies that a registered user can log in and receive tokens.
func TestUserLogin(t *testing.T) {
	skipIfNotRunning(t, userPort)

	// First register a user.
	email := uniqueEmail("login")
	regBody := map[string]interface{}{
		"email":      email,
		"password":   "TestPass123!",
		"first_name": "Login",
		"last_name":  "Test",
	}
	regStatus, _ := httpPost(t, baseURL(userPort)+"/api/v1/auth/register", regBody)
	requireStatus(t, regStatus, 201)

	// Now login.
	loginBody := map[string]interface{}{
		"email":    email,
		"password": "TestPass123!",
	}
	status, data := httpPost(t, baseURL(userPort)+"/api/v1/auth/login", loginBody)
	requireStatus(t, status, 200)

	accessToken := extractField(data, "data.tokens.access_token")
	if accessToken == nil {
		t.Fatal("expected data.tokens.access_token in login response, got nil")
	}

	userID := extractField(data, "data.user.id")
	if userID == nil {
		t.Fatal("expected data.user.id in login response, got nil")
	}

	t.Logf("logged in user %s, got access_token (length %d)", email, len(accessToken.(string)))
}

// TestUserLoginInvalidPassword verifies that login with wrong password returns 401.
func TestUserLoginInvalidPassword(t *testing.T) {
	skipIfNotRunning(t, userPort)

	// Register a user first.
	email := uniqueEmail("badpw")
	regBody := map[string]interface{}{
		"email":      email,
		"password":   "TestPass123!",
		"first_name": "BadPW",
		"last_name":  "Test",
	}
	regStatus, _ := httpPost(t, baseURL(userPort)+"/api/v1/auth/register", regBody)
	requireStatus(t, regStatus, 201)

	// Try to login with wrong password.
	loginBody := map[string]interface{}{
		"email":    email,
		"password": "WrongPassword999",
	}
	status, data := httpPost(t, baseURL(userPort)+"/api/v1/auth/login", loginBody)

	// Expect 401 Unauthorized.
	if status != 401 {
		t.Fatalf("expected status 401 for wrong password, got %d; body: %v", status, data)
	}

	// Verify there is an error in the response.
	errField := extractField(data, "error")
	if errField == nil {
		t.Fatal("expected error field in response for invalid password")
	}
}

// TestUserDuplicateRegistration verifies that registering with an already-used email
// returns a conflict or error status (409 or 400).
func TestUserDuplicateRegistration(t *testing.T) {
	skipIfNotRunning(t, userPort)

	email := uniqueEmail("dup")
	body := map[string]interface{}{
		"email":      email,
		"password":   "TestPass123!",
		"first_name": "Dup",
		"last_name":  "Test",
	}

	// Register the first time — should succeed.
	status1, _ := httpPost(t, baseURL(userPort)+"/api/v1/auth/register", body)
	requireStatus(t, status1, 201)

	// Register the second time — should fail.
	status2, data2 := httpPost(t, baseURL(userPort)+"/api/v1/auth/register", body)

	// Accept either 409 Conflict or 400 Bad Request, depending on the service implementation.
	if status2 != 409 && status2 != 400 {
		t.Fatalf("expected status 409 or 400 for duplicate registration, got %d; body: %v", status2, data2)
	}

	t.Logf("duplicate registration correctly returned status %d", status2)
}

// TestUserRegistrationValidation verifies that missing required fields return 400.
func TestUserRegistrationValidation(t *testing.T) {
	skipIfNotRunning(t, userPort)

	// Missing all required fields.
	body := map[string]interface{}{}
	status, data := httpPost(t, baseURL(userPort)+"/api/v1/auth/register", body)

	if status != 400 {
		t.Fatalf("expected status 400 for empty registration, got %d; body: %v", status, data)
	}

	// Missing password.
	body2 := map[string]interface{}{
		"email":      uniqueEmail("val"),
		"first_name": "Val",
		"last_name":  "Test",
	}
	status2, data2 := httpPost(t, baseURL(userPort)+"/api/v1/auth/register", body2)
	if status2 != 400 {
		t.Fatalf("expected status 400 for missing password, got %d; body: %v", status2, data2)
	}
}

// registerAndLogin is a test helper that registers a new user and logs in,
// returning the user ID and access token. Intended for use by other test files
// that need an authenticated user.
func registerAndLogin(t *testing.T) (userID, accessToken string) {
	t.Helper()
	skipIfNotRunning(t, userPort)

	email := uniqueEmail("helper")
	regBody := map[string]interface{}{
		"email":      email,
		"password":   "TestPass123!",
		"first_name": "Helper",
		"last_name":  "User",
	}
	regStatus, _ := httpPost(t, baseURL(userPort)+"/api/v1/auth/register", regBody)
	requireStatus(t, regStatus, 201)

	loginBody := map[string]interface{}{
		"email":    email,
		"password": "TestPass123!",
	}
	loginStatus, loginData := httpPost(t, baseURL(userPort)+"/api/v1/auth/login", loginBody)
	requireStatus(t, loginStatus, 200)

	userID = extractString(t, loginData, "data.user.id")
	accessToken = extractString(t, loginData, "data.tokens.access_token")
	return userID, accessToken
}
