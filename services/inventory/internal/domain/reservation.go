package domain

import (
	"time"
)

// Reservation status constants.
const (
	ReservationStatusActive    = "active"
	ReservationStatusConfirmed = "confirmed"
	ReservationStatusReleased  = "released"
	ReservationStatusExpired   = "expired"
)

// StockReservation represents a temporary hold on inventory for a checkout session.
type StockReservation struct {
	ID         string    `json:"id"`
	ProductID  string    `json:"product_id"`
	VariantID  string    `json:"variant_id"`
	Quantity   int       `json:"quantity"`
	CheckoutID string    `json:"checkout_id"`
	Status     string    `json:"status"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// IsActive returns true if the reservation is still active.
func (r *StockReservation) IsActive() bool {
	return r.Status == ReservationStatusActive
}

// IsExpired returns true if the reservation has passed its expiration time.
func (r *StockReservation) IsExpired() bool {
	return time.Now().UTC().After(r.ExpiresAt)
}

// ValidReservationStatuses returns the set of valid reservation statuses.
func ValidReservationStatuses() []string {
	return []string{ReservationStatusActive, ReservationStatusConfirmed, ReservationStatusReleased, ReservationStatusExpired}
}

// IsValidReservationStatus checks whether the given status is a valid reservation status.
func IsValidReservationStatus(status string) bool {
	for _, s := range ValidReservationStatuses() {
		if s == status {
			return true
		}
	}
	return false
}
