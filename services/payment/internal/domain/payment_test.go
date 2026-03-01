package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Payment Status Validation Tests
// ============================================================================

func TestValidPaymentStatuses_ContainsAll(t *testing.T) {
	statuses := ValidPaymentStatuses()
	expected := []string{
		PaymentStatusPending, PaymentStatusProcessing, PaymentStatusSucceeded,
		PaymentStatusFailed, PaymentStatusCanceled, PaymentStatusRefunded,
		PaymentStatusPartiallyRefunded,
	}
	assert.ElementsMatch(t, expected, statuses)
}

func TestIsValidPaymentStatus_ValidStatuses(t *testing.T) {
	for _, s := range ValidPaymentStatuses() {
		assert.True(t, IsValidPaymentStatus(s), "expected %q to be valid", s)
	}
}

func TestIsValidPaymentStatus_Invalid(t *testing.T) {
	assert.False(t, IsValidPaymentStatus("unknown"))
	assert.False(t, IsValidPaymentStatus(""))
	assert.False(t, IsValidPaymentStatus("PENDING"))
}

// ============================================================================
// Payment Method Validation Tests
// ============================================================================

func TestValidPaymentMethods_ContainsAll(t *testing.T) {
	methods := ValidPaymentMethods()
	expected := []string{
		PaymentMethodCreditCard, PaymentMethodDebitCard,
		PaymentMethodBankTransfer, PaymentMethodWallet,
	}
	assert.ElementsMatch(t, expected, methods)
}

func TestIsValidPaymentMethod_ValidMethods(t *testing.T) {
	for _, m := range ValidPaymentMethods() {
		assert.True(t, IsValidPaymentMethod(m), "expected %q to be valid", m)
	}
}

func TestIsValidPaymentMethod_Invalid(t *testing.T) {
	assert.False(t, IsValidPaymentMethod("cash"))
	assert.False(t, IsValidPaymentMethod(""))
	assert.False(t, IsValidPaymentMethod("CREDIT_CARD"))
}

// ============================================================================
// Payment Struct Tests
// ============================================================================

func TestPayment_AmountInCents(t *testing.T) {
	p := Payment{Amount: 9999, Currency: "USD"}
	assert.Equal(t, int64(9999), p.Amount)
	assert.Equal(t, "USD", p.Currency)
}

func TestPayment_ZeroAmount(t *testing.T) {
	p := Payment{Amount: 0}
	assert.Equal(t, int64(0), p.Amount)
}

// ============================================================================
// Refund Status Validation Tests
// ============================================================================

func TestValidRefundStatuses_ContainsAll(t *testing.T) {
	statuses := ValidRefundStatuses()
	expected := []string{
		RefundStatusPending, RefundStatusProcessing,
		RefundStatusSucceeded, RefundStatusFailed,
	}
	assert.ElementsMatch(t, expected, statuses)
}

func TestIsValidRefundStatus_ValidStatuses(t *testing.T) {
	for _, s := range ValidRefundStatuses() {
		assert.True(t, IsValidRefundStatus(s), "expected %q to be valid", s)
	}
}

func TestIsValidRefundStatus_Invalid(t *testing.T) {
	assert.False(t, IsValidRefundStatus("unknown"))
	assert.False(t, IsValidRefundStatus(""))
}

// ============================================================================
// Refund Amount Tests
// ============================================================================

func TestRefund_AmountInCents(t *testing.T) {
	r := Refund{Amount: 5000, Currency: "EUR"}
	assert.Equal(t, int64(5000), r.Amount)
	assert.Equal(t, "EUR", r.Currency)
}
