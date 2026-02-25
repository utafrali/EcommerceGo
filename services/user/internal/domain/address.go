package domain

import (
	"time"
)

// Address represents a shipping or billing address for a user.
type Address struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Label        string    `json:"label,omitempty"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	AddressLine1 string    `json:"address_line1"`
	AddressLine2 string    `json:"address_line2,omitempty"`
	City         string    `json:"city"`
	State        string    `json:"state,omitempty"`
	PostalCode   string    `json:"postal_code"`
	CountryCode  string    `json:"country_code"`
	Phone        string    `json:"phone,omitempty"`
	IsDefault    bool      `json:"is_default"`
	CreatedAt    time.Time `json:"created_at"`
}
