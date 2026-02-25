package domain

import (
	"time"
)

// User represents a registered user in the system.
type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	PasswordHash  string    `json:"-"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Phone         string    `json:"phone,omitempty"`
	Role          string    `json:"role"`
	IsActive      bool      `json:"is_active"`
	EmailVerified bool      `json:"email_verified"`
	OAuthProvider string    `json:"oauth_provider,omitempty"`
	OAuthID       string    `json:"oauth_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// RefreshToken represents a stored refresh token for a user session.
type RefreshToken struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// TokenPair holds an access and refresh token pair.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
