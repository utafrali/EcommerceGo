package domain

import "time"

// Order status constants.
const (
	OrderStatusPending    = "pending"
	OrderStatusConfirmed  = "confirmed"
	OrderStatusProcessing = "processing"
	OrderStatusShipped    = "shipped"
	OrderStatusDelivered  = "delivered"
	OrderStatusCanceled   = "canceled"
	OrderStatusRefunded   = "refunded"
)

// Order represents a customer order.
type Order struct {
	ID              string      `json:"id"`
	UserID          string      `json:"user_id"`
	Status          string      `json:"status"`
	Items           []OrderItem `json:"items"`
	SubtotalAmount  int64       `json:"subtotal_amount"`
	DiscountAmount  int64       `json:"discount_amount"`
	ShippingAmount  int64       `json:"shipping_amount"`
	TotalAmount     int64       `json:"total_amount"`
	Currency        string      `json:"currency"`
	ShippingAddress *Address    `json:"shipping_address,omitempty"`
	BillingAddress  *Address    `json:"billing_address,omitempty"`
	Notes           string      `json:"notes,omitempty"`
	CanceledReason  string      `json:"canceled_reason,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
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

// ValidStatuses returns all valid order statuses.
func ValidStatuses() []string {
	return []string{
		OrderStatusPending,
		OrderStatusConfirmed,
		OrderStatusProcessing,
		OrderStatusShipped,
		OrderStatusDelivered,
		OrderStatusCanceled,
		OrderStatusRefunded,
	}
}

// IsValidStatus checks if a status string is valid.
func IsValidStatus(status string) bool {
	for _, s := range ValidStatuses() {
		if s == status {
			return true
		}
	}
	return false
}

// AllowedTransitions defines which status transitions are valid.
func AllowedTransitions() map[string][]string {
	return map[string][]string{
		OrderStatusPending:    {OrderStatusConfirmed, OrderStatusCanceled},
		OrderStatusConfirmed:  {OrderStatusProcessing, OrderStatusCanceled},
		OrderStatusProcessing: {OrderStatusShipped, OrderStatusCanceled},
		OrderStatusShipped:    {OrderStatusDelivered},
		OrderStatusDelivered:  {OrderStatusRefunded},
		OrderStatusCanceled:   {},
		OrderStatusRefunded:   {},
	}
}

// CanTransitionTo checks if the order can transition to the target status.
func (o *Order) CanTransitionTo(target string) bool {
	allowed, ok := AllowedTransitions()[o.Status]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == target {
			return true
		}
	}
	return false
}
