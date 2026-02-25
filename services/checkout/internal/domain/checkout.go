package domain

import (
	"time"
)

// Checkout session status constants.
const (
	StatusInitiated         = "initiated"
	StatusItemsReserved     = "items_reserved"
	StatusPaymentPending    = "payment_pending"
	StatusPaymentProcessing = "payment_processing"
	StatusCompleted         = "completed"
	StatusFailed            = "failed"
	StatusExpired           = "expired"
)

// CheckoutSession represents an ongoing checkout.
type CheckoutSession struct {
	ID              string         `json:"id"`
	UserID          string         `json:"user_id"`
	Status          string         `json:"status"`
	Items           []CheckoutItem `json:"items"`
	SubtotalAmount  int64          `json:"subtotal_amount"`
	DiscountAmount  int64          `json:"discount_amount"`
	ShippingAmount  int64          `json:"shipping_amount"`
	TotalAmount     int64          `json:"total_amount"`
	Currency        string         `json:"currency"`
	ShippingAddress *Address       `json:"shipping_address,omitempty"`
	BillingAddress  *Address       `json:"billing_address,omitempty"`
	PaymentMethod   string         `json:"payment_method,omitempty"`
	PaymentID       string         `json:"payment_id,omitempty"`
	OrderID         string         `json:"order_id,omitempty"`
	FailureReason   string         `json:"failure_reason,omitempty"`
	ExpiresAt       time.Time      `json:"expires_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// CheckoutItem represents a single item in a checkout session.
type CheckoutItem struct {
	ProductID     string `json:"product_id"`
	VariantID     string `json:"variant_id"`
	Name          string `json:"name"`
	SKU           string `json:"sku"`
	Price         int64  `json:"price"`
	Quantity      int    `json:"quantity"`
	ReservationID string `json:"reservation_id,omitempty"`
}

// Address represents a shipping or billing address.
type Address struct {
	FullName    string `json:"full_name"`
	AddressLine string `json:"address_line"`
	City        string `json:"city"`
	State       string `json:"state"`
	PostalCode  string `json:"postal_code"`
	Country     string `json:"country"`
	Phone       string `json:"phone,omitempty"`
}

// CalculateSubtotal computes the subtotal from the items (price * quantity for each).
func (s *CheckoutSession) CalculateSubtotal() int64 {
	var subtotal int64
	for _, item := range s.Items {
		subtotal += item.Price * int64(item.Quantity)
	}
	return subtotal
}

// CalculateTotal computes the total: subtotal - discount + shipping.
func (s *CheckoutSession) CalculateTotal() int64 {
	return s.SubtotalAmount - s.DiscountAmount + s.ShippingAmount
}

// IsExpired checks whether the session has passed its expiry time.
func (s *CheckoutSession) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

// IsTerminal returns true if the session is in a final state.
func (s *CheckoutSession) IsTerminal() bool {
	return s.Status == StatusCompleted || s.Status == StatusFailed || s.Status == StatusExpired
}

// ValidStatuses returns the set of valid checkout session statuses.
func ValidStatuses() []string {
	return []string{
		StatusInitiated,
		StatusItemsReserved,
		StatusPaymentPending,
		StatusPaymentProcessing,
		StatusCompleted,
		StatusFailed,
		StatusExpired,
	}
}

// IsValidStatus checks whether the given status string is a valid checkout status.
func IsValidStatus(status string) bool {
	for _, s := range ValidStatuses() {
		if s == status {
			return true
		}
	}
	return false
}
