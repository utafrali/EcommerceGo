package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Stock.Available Tests
// ============================================================================

func TestAvailable_Normal(t *testing.T) {
	s := &Stock{Quantity: 100, Reserved: 30}
	assert.Equal(t, 70, s.Available())
}

func TestAvailable_AllReserved(t *testing.T) {
	s := &Stock{Quantity: 50, Reserved: 50}
	assert.Equal(t, 0, s.Available())
}

func TestAvailable_NoneReserved(t *testing.T) {
	s := &Stock{Quantity: 100, Reserved: 0}
	assert.Equal(t, 100, s.Available())
}

func TestAvailable_ReservedExceedsQuantity(t *testing.T) {
	// Guards against negative: should return 0
	s := &Stock{Quantity: 10, Reserved: 20}
	assert.Equal(t, 0, s.Available())
}

func TestAvailable_ZeroStock(t *testing.T) {
	s := &Stock{Quantity: 0, Reserved: 0}
	assert.Equal(t, 0, s.Available())
}

// ============================================================================
// Movement Reason Validation Tests
// ============================================================================

func TestValidMovementReasons_ContainsAll(t *testing.T) {
	reasons := ValidMovementReasons()
	expected := []string{
		MovementReasonOrder, MovementReasonReturn,
		MovementReasonAdjustment, MovementReasonReservation,
	}
	assert.ElementsMatch(t, expected, reasons)
}

func TestIsValidMovementReason_Valid(t *testing.T) {
	for _, r := range ValidMovementReasons() {
		assert.True(t, IsValidMovementReason(r), "expected %q to be valid", r)
	}
}

func TestIsValidMovementReason_Invalid(t *testing.T) {
	assert.False(t, IsValidMovementReason("unknown"))
	assert.False(t, IsValidMovementReason(""))
	assert.False(t, IsValidMovementReason("ORDER"))
}

// ============================================================================
// Stock Struct Tests
// ============================================================================

func TestStock_DefaultWarehouseID(t *testing.T) {
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", DefaultWarehouseID)
}

func TestStock_LowStockThreshold(t *testing.T) {
	s := &Stock{Quantity: 5, Reserved: 0, LowStockThreshold: 10}
	assert.True(t, s.Available() < s.LowStockThreshold)
}

func TestStockMovement_NegativeQuantityChange(t *testing.T) {
	m := StockMovement{QuantityChange: -5, Reason: MovementReasonOrder}
	assert.Equal(t, -5, m.QuantityChange)
	assert.Equal(t, MovementReasonOrder, m.Reason)
}

func TestStockMovement_PositiveQuantityChange(t *testing.T) {
	m := StockMovement{QuantityChange: 10, Reason: MovementReasonReturn}
	assert.Equal(t, 10, m.QuantityChange)
	assert.Equal(t, MovementReasonReturn, m.Reason)
}

// ============================================================================
// StockCheckResult Tests
// ============================================================================

func TestStockCheckResult_InStock(t *testing.T) {
	r := StockCheckResult{Requested: 5, Available: 10, InStock: true}
	assert.True(t, r.InStock)
	assert.Equal(t, 5, r.Requested)
}

func TestStockCheckResult_OutOfStock(t *testing.T) {
	r := StockCheckResult{Requested: 15, Available: 10, InStock: false}
	assert.False(t, r.InStock)
	assert.Equal(t, 15, r.Requested)
}

// ============================================================================
// Reservation Tests
// ============================================================================

func TestReservation_IsActive(t *testing.T) {
	r := &StockReservation{Status: ReservationStatusActive}
	assert.True(t, r.IsActive())
}

func TestReservation_IsNotActive(t *testing.T) {
	for _, status := range []string{ReservationStatusConfirmed, ReservationStatusReleased, ReservationStatusExpired} {
		r := &StockReservation{Status: status}
		assert.False(t, r.IsActive(), "expected %q to not be active", status)
	}
}

func TestReservation_IsExpired(t *testing.T) {
	r := &StockReservation{ExpiresAt: time.Now().Add(-1 * time.Hour)}
	assert.True(t, r.IsExpired())
}

func TestReservation_IsNotExpired(t *testing.T) {
	r := &StockReservation{ExpiresAt: time.Now().Add(1 * time.Hour)}
	assert.False(t, r.IsExpired())
}

func TestValidReservationStatuses_ContainsAll(t *testing.T) {
	statuses := ValidReservationStatuses()
	expected := []string{
		ReservationStatusActive, ReservationStatusConfirmed,
		ReservationStatusReleased, ReservationStatusExpired,
	}
	assert.ElementsMatch(t, expected, statuses)
}

func TestIsValidReservationStatus_Valid(t *testing.T) {
	for _, s := range ValidReservationStatuses() {
		assert.True(t, IsValidReservationStatus(s), "expected %q to be valid", s)
	}
}

func TestIsValidReservationStatus_Invalid(t *testing.T) {
	assert.False(t, IsValidReservationStatus("unknown"))
	assert.False(t, IsValidReservationStatus(""))
}
