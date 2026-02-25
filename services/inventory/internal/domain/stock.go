package domain

import (
	"time"
)

// DefaultWarehouseID is the default warehouse identifier used when no specific warehouse is specified.
const DefaultWarehouseID = "00000000-0000-0000-0000-000000000001"

// Stock represents the inventory level for a specific product variant in a warehouse.
type Stock struct {
	ID                string    `json:"id"`
	ProductID         string    `json:"product_id"`
	VariantID         string    `json:"variant_id"`
	WarehouseID       string    `json:"warehouse_id"`
	Quantity          int       `json:"quantity"`
	Reserved          int       `json:"reserved"`
	LowStockThreshold int       `json:"low_stock_threshold"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Available returns the quantity available for purchase (Quantity - Reserved).
func (s *Stock) Available() int {
	return s.Quantity - s.Reserved
}

// StockCheckItem represents a single item to check availability for.
type StockCheckItem struct {
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id"`
	Quantity  int    `json:"quantity"`
}

// StockCheckResult represents the availability check result for a single item.
type StockCheckResult struct {
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id"`
	Requested int    `json:"requested"`
	Available int    `json:"available"`
	InStock   bool   `json:"in_stock"`
}

// StockMovement records a change in stock quantity.
type StockMovement struct {
	ID             string    `json:"id"`
	ProductID      string    `json:"product_id"`
	VariantID      string    `json:"variant_id"`
	QuantityChange int       `json:"quantity_change"`
	Reason         string    `json:"reason"`
	ReferenceID    *string   `json:"reference_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// Stock movement reasons.
const (
	MovementReasonOrder       = "order"
	MovementReasonReturn      = "return"
	MovementReasonAdjustment  = "adjustment"
	MovementReasonReservation = "reservation"
)

// ValidMovementReasons returns the set of valid movement reasons.
func ValidMovementReasons() []string {
	return []string{MovementReasonOrder, MovementReasonReturn, MovementReasonAdjustment, MovementReasonReservation}
}

// IsValidMovementReason checks whether the given reason is a valid stock movement reason.
func IsValidMovementReason(reason string) bool {
	for _, r := range ValidMovementReasons() {
		if r == reason {
			return true
		}
	}
	return false
}
