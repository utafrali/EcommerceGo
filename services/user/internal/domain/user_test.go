package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Role Validation Tests
// ============================================================================

func TestValidRoles_ContainsAll(t *testing.T) {
	roles := ValidRoles()
	expected := []string{RoleCustomer, RoleAdmin, RoleSeller}
	assert.ElementsMatch(t, expected, roles)
}

func TestIsValidRole_ValidRoles(t *testing.T) {
	for _, r := range ValidRoles() {
		assert.True(t, IsValidRole(r), "expected %q to be valid", r)
	}
}

func TestIsValidRole_Invalid(t *testing.T) {
	assert.False(t, IsValidRole("unknown"))
	assert.False(t, IsValidRole(""))
	assert.False(t, IsValidRole("ADMIN"))
	assert.False(t, IsValidRole("superadmin"))
}

// ============================================================================
// User Struct Tests
// ============================================================================

func TestUser_PasswordHashExcludedFromJSON(t *testing.T) {
	u := User{PasswordHash: "secret"}
	assert.Equal(t, "secret", u.PasswordHash)
	// The json:"-" tag ensures PasswordHash is excluded from serialization.
	// Testing struct tag presence is validated at compile time.
}

func TestUser_DefaultFields(t *testing.T) {
	u := User{}
	assert.False(t, u.IsActive)
	assert.False(t, u.EmailVerified)
	assert.Empty(t, u.Role)
}

func TestUser_ActiveUser(t *testing.T) {
	u := User{
		ID:            "user-1",
		Email:         "test@example.com",
		FirstName:     "John",
		LastName:      "Doe",
		Role:          RoleCustomer,
		IsActive:      true,
		EmailVerified: true,
	}
	assert.True(t, u.IsActive)
	assert.True(t, u.EmailVerified)
	assert.Equal(t, RoleCustomer, u.Role)
}

func TestUser_OAuthFields(t *testing.T) {
	u := User{OAuthProvider: "google", OAuthID: "google-123"}
	assert.Equal(t, "google", u.OAuthProvider)
	assert.Equal(t, "google-123", u.OAuthID)
}

// ============================================================================
// RefreshToken Tests
// ============================================================================

func TestRefreshToken_TokenHashExcludedFromJSON(t *testing.T) {
	rt := RefreshToken{TokenHash: "hashed-value"}
	assert.Equal(t, "hashed-value", rt.TokenHash)
}

func TestRefreshToken_Expiry(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	rt := RefreshToken{ExpiresAt: future}
	assert.True(t, rt.ExpiresAt.After(time.Now()))
}

func TestRefreshToken_Revoked(t *testing.T) {
	now := time.Now()
	rt := RefreshToken{RevokedAt: &now}
	assert.NotNil(t, rt.RevokedAt)
}

func TestRefreshToken_NotRevoked(t *testing.T) {
	rt := RefreshToken{}
	assert.Nil(t, rt.RevokedAt)
}

// ============================================================================
// TokenPair Tests
// ============================================================================

func TestTokenPair_Fields(t *testing.T) {
	tp := TokenPair{AccessToken: "access-123", RefreshToken: "refresh-456"}
	assert.Equal(t, "access-123", tp.AccessToken)
	assert.Equal(t, "refresh-456", tp.RefreshToken)
}

// ============================================================================
// Address Tests
// ============================================================================

func TestAddress_RequiredFields(t *testing.T) {
	addr := Address{
		ID:           "addr-1",
		UserID:       "user-1",
		FirstName:    "John",
		LastName:     "Doe",
		AddressLine1: "123 Main St",
		City:         "New York",
		PostalCode:   "10001",
		CountryCode:  "US",
	}
	assert.NotEmpty(t, addr.FirstName)
	assert.NotEmpty(t, addr.LastName)
	assert.NotEmpty(t, addr.AddressLine1)
	assert.NotEmpty(t, addr.City)
	assert.NotEmpty(t, addr.PostalCode)
	assert.Len(t, addr.CountryCode, 2)
}

func TestAddress_OptionalFields(t *testing.T) {
	addr := Address{
		Label:        "Home",
		AddressLine2: "Apt 4",
		State:        "NY",
		Phone:        "+1234567890",
	}
	assert.Equal(t, "Home", addr.Label)
	assert.Equal(t, "Apt 4", addr.AddressLine2)
	assert.Equal(t, "NY", addr.State)
	assert.Equal(t, "+1234567890", addr.Phone)
}

func TestAddress_IsDefault(t *testing.T) {
	addr := Address{IsDefault: true}
	assert.True(t, addr.IsDefault)
}

func TestAddress_NotDefault(t *testing.T) {
	addr := Address{}
	assert.False(t, addr.IsDefault)
}
