package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// OrderItem.LineTotal Tests
// ============================================================================

func TestLineTotal_BasicCalculation(t *testing.T) {
	item := OrderItem{Price: 1999, Quantity: 3}
	assert.Equal(t, int64(5997), item.LineTotal())
}

func TestLineTotal_SingleItem(t *testing.T) {
	item := OrderItem{Price: 500, Quantity: 1}
	assert.Equal(t, int64(500), item.LineTotal())
}

func TestLineTotal_ZeroQuantity(t *testing.T) {
	item := OrderItem{Price: 1999, Quantity: 0}
	assert.Equal(t, int64(0), item.LineTotal())
}

func TestLineTotal_ZeroPrice(t *testing.T) {
	item := OrderItem{Price: 0, Quantity: 5}
	assert.Equal(t, int64(0), item.LineTotal())
}

func TestLineTotal_LargeValues(t *testing.T) {
	item := OrderItem{Price: 99999999, Quantity: 1000}
	assert.Equal(t, int64(99999999000), item.LineTotal())
}

// ============================================================================
// Order Status Validation Tests
// ============================================================================

func TestValidStatuses_ContainsAllStatuses(t *testing.T) {
	statuses := ValidStatuses()
	expected := []string{
		OrderStatusPending, OrderStatusConfirmed, OrderStatusProcessing,
		OrderStatusShipped, OrderStatusDelivered, OrderStatusCanceled, OrderStatusRefunded,
	}
	assert.ElementsMatch(t, expected, statuses)
}

func TestIsValidStatus_ValidStatuses(t *testing.T) {
	for _, s := range ValidStatuses() {
		assert.True(t, IsValidStatus(s), "expected %q to be valid", s)
	}
}

func TestIsValidStatus_InvalidStatus(t *testing.T) {
	assert.False(t, IsValidStatus("unknown"))
	assert.False(t, IsValidStatus(""))
	assert.False(t, IsValidStatus("PENDING")) // case-sensitive
}

// ============================================================================
// Order State Transitions Tests
// ============================================================================

func TestAllowedTransitions_PendingCanTransition(t *testing.T) {
	transitions := AllowedTransitions()
	allowed := transitions[OrderStatusPending]
	assert.Contains(t, allowed, OrderStatusConfirmed)
	assert.Contains(t, allowed, OrderStatusCanceled)
}

func TestCanTransitionTo_ValidTransition(t *testing.T) {
	order := &Order{Status: OrderStatusPending}
	assert.True(t, order.CanTransitionTo(OrderStatusConfirmed))
}

func TestCanTransitionTo_InvalidTransition(t *testing.T) {
	order := &Order{Status: OrderStatusPending}
	assert.False(t, order.CanTransitionTo(OrderStatusDelivered))
}

func TestCanTransitionTo_DeliveredIsTerminal(t *testing.T) {
	order := &Order{Status: OrderStatusDelivered}
	// Delivered can only transition to refunded
	assert.False(t, order.CanTransitionTo(OrderStatusPending))
	assert.False(t, order.CanTransitionTo(OrderStatusShipped))
}

func TestCanTransitionTo_CanceledIsTerminal(t *testing.T) {
	order := &Order{Status: OrderStatusCanceled}
	assert.False(t, order.CanTransitionTo(OrderStatusPending))
	assert.False(t, order.CanTransitionTo(OrderStatusConfirmed))
}

func TestCanTransitionTo_ShippedToDelivered(t *testing.T) {
	order := &Order{Status: OrderStatusShipped}
	assert.True(t, order.CanTransitionTo(OrderStatusDelivered))
}

func TestCanTransitionTo_SameStatus(t *testing.T) {
	order := &Order{Status: OrderStatusPending}
	assert.False(t, order.CanTransitionTo(OrderStatusPending))
}

func TestCanTransitionTo_UnknownCurrentStatus(t *testing.T) {
	order := &Order{Status: "nonexistent"}
	assert.False(t, order.CanTransitionTo(OrderStatusConfirmed))
}
