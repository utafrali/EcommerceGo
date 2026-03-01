package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// CheckoutSession.CalculateSubtotal Tests
// ============================================================================

func TestCalculateSubtotal_SingleItem(t *testing.T) {
	s := &CheckoutSession{
		Items: []CheckoutItem{
			{Price: 1000, Quantity: 2},
		},
	}
	assert.Equal(t, int64(2000), s.CalculateSubtotal())
}

func TestCalculateSubtotal_MultipleItems(t *testing.T) {
	s := &CheckoutSession{
		Items: []CheckoutItem{
			{Price: 1000, Quantity: 2},
			{Price: 500, Quantity: 3},
		},
	}
	assert.Equal(t, int64(3500), s.CalculateSubtotal())
}

func TestCalculateSubtotal_EmptyItems(t *testing.T) {
	s := &CheckoutSession{Items: []CheckoutItem{}}
	assert.Equal(t, int64(0), s.CalculateSubtotal())
}

func TestCalculateSubtotal_NilItems(t *testing.T) {
	s := &CheckoutSession{}
	assert.Equal(t, int64(0), s.CalculateSubtotal())
}

func TestCalculateSubtotal_ZeroPrice(t *testing.T) {
	s := &CheckoutSession{
		Items: []CheckoutItem{
			{Price: 0, Quantity: 5},
		},
	}
	assert.Equal(t, int64(0), s.CalculateSubtotal())
}

// ============================================================================
// CheckoutSession.CalculateTotal Tests
// ============================================================================

func TestCalculateTotal_Basic(t *testing.T) {
	s := &CheckoutSession{
		SubtotalAmount: 5000,
		DiscountAmount: 500,
		ShippingAmount: 300,
	}
	// 5000 - 500 + 300 = 4800
	assert.Equal(t, int64(4800), s.CalculateTotal())
}

func TestCalculateTotal_NoDiscount(t *testing.T) {
	s := &CheckoutSession{
		SubtotalAmount: 5000,
		DiscountAmount: 0,
		ShippingAmount: 0,
	}
	assert.Equal(t, int64(5000), s.CalculateTotal())
}

func TestCalculateTotal_FreeShipping(t *testing.T) {
	s := &CheckoutSession{
		SubtotalAmount: 10000,
		DiscountAmount: 1000,
		ShippingAmount: 0,
	}
	assert.Equal(t, int64(9000), s.CalculateTotal())
}

// ============================================================================
// CheckoutSession.IsExpired Tests
// ============================================================================

func TestIsExpired_PastExpiry(t *testing.T) {
	s := &CheckoutSession{
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	assert.True(t, s.IsExpired())
}

func TestIsExpired_FutureExpiry(t *testing.T) {
	s := &CheckoutSession{
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	assert.False(t, s.IsExpired())
}

func TestIsExpired_ZeroExpiry(t *testing.T) {
	s := &CheckoutSession{}
	// Zero time is in the past
	assert.True(t, s.IsExpired())
}

// ============================================================================
// CheckoutSession.IsTerminal Tests
// ============================================================================

func TestIsTerminal_Completed(t *testing.T) {
	s := &CheckoutSession{Status: StatusCompleted}
	assert.True(t, s.IsTerminal())
}

func TestIsTerminal_Failed(t *testing.T) {
	s := &CheckoutSession{Status: StatusFailed}
	assert.True(t, s.IsTerminal())
}

func TestIsTerminal_Expired(t *testing.T) {
	s := &CheckoutSession{Status: StatusExpired}
	assert.True(t, s.IsTerminal())
}

func TestIsTerminal_NonTerminal(t *testing.T) {
	nonTerminal := []string{StatusInitiated, StatusItemsReserved, StatusPaymentPending, StatusPaymentProcessing}
	for _, status := range nonTerminal {
		s := &CheckoutSession{Status: status}
		assert.False(t, s.IsTerminal(), "expected %q to be non-terminal", status)
	}
}

// ============================================================================
// Checkout Status Validation Tests
// ============================================================================

func TestValidStatuses_ContainsAll(t *testing.T) {
	statuses := ValidStatuses()
	expected := []string{
		StatusInitiated, StatusItemsReserved, StatusPaymentPending,
		StatusPaymentProcessing, StatusCompleted, StatusFailed, StatusExpired,
	}
	assert.ElementsMatch(t, expected, statuses)
}

func TestIsValidStatus_Valid(t *testing.T) {
	for _, s := range ValidStatuses() {
		assert.True(t, IsValidStatus(s), "expected %q to be valid", s)
	}
}

func TestIsValidStatus_Invalid(t *testing.T) {
	assert.False(t, IsValidStatus("unknown"))
	assert.False(t, IsValidStatus(""))
}

// ============================================================================
// SagaStep Tests
// ============================================================================

func TestNewSagaStep_CreatesWithPendingStatus(t *testing.T) {
	step := NewSagaStep(SagaStepReserveInventory)
	assert.Equal(t, SagaStepReserveInventory, step.Name)
	assert.Equal(t, SagaStepPending, step.Status)
	assert.Empty(t, step.Error)
	assert.True(t, step.ExecutedAt.IsZero())
}

func TestSagaStep_Complete(t *testing.T) {
	step := NewSagaStep(SagaStepCreateOrder)
	step.Complete()
	assert.Equal(t, SagaStepCompleted, step.Status)
	assert.False(t, step.ExecutedAt.IsZero())
}

func TestSagaStep_Fail(t *testing.T) {
	step := NewSagaStep(SagaStepInitiatePayment)
	step.Fail("payment provider timeout")
	assert.Equal(t, SagaStepFailed, step.Status)
	assert.Equal(t, "payment provider timeout", step.Error)
	assert.False(t, step.ExecutedAt.IsZero())
}

func TestSagaStep_Compensate(t *testing.T) {
	step := NewSagaStep(SagaStepReserveInventory)
	step.Complete()
	step.Compensate()
	assert.Equal(t, SagaStepCompensated, step.Status)
}

func TestSagaStep_StepNames(t *testing.T) {
	assert.Equal(t, "reserve_inventory", SagaStepReserveInventory)
	assert.Equal(t, "create_order", SagaStepCreateOrder)
	assert.Equal(t, "initiate_payment", SagaStepInitiatePayment)
}
