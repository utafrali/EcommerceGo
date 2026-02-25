package domain

import (
	"time"
)

// Payment status constants.
const (
	PaymentStatusPending            = "pending"
	PaymentStatusProcessing         = "processing"
	PaymentStatusSucceeded          = "succeeded"
	PaymentStatusFailed             = "failed"
	PaymentStatusCanceled           = "canceled"
	PaymentStatusRefunded           = "refunded"
	PaymentStatusPartiallyRefunded  = "partially_refunded"
)

// Payment method constants.
const (
	PaymentMethodCreditCard   = "credit_card"
	PaymentMethodDebitCard    = "debit_card"
	PaymentMethodBankTransfer = "bank_transfer"
	PaymentMethodWallet       = "wallet"
)

// Payment represents a payment transaction.
type Payment struct {
	ID            string         `json:"id"`
	CheckoutID    string         `json:"checkout_id"`
	OrderID       string         `json:"order_id"`
	UserID        string         `json:"user_id"`
	Amount        int64          `json:"amount"`
	Currency      string         `json:"currency"`
	Status        string         `json:"status"`
	Method        string         `json:"method"`
	ProviderName  string         `json:"provider_name"`
	ProviderPayID string         `json:"provider_payment_id,omitempty"`
	FailureReason string         `json:"failure_reason,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// ValidPaymentStatuses returns all valid payment statuses.
func ValidPaymentStatuses() []string {
	return []string{
		PaymentStatusPending,
		PaymentStatusProcessing,
		PaymentStatusSucceeded,
		PaymentStatusFailed,
		PaymentStatusCanceled,
		PaymentStatusRefunded,
		PaymentStatusPartiallyRefunded,
	}
}

// IsValidPaymentStatus checks whether the given status is a valid payment status.
func IsValidPaymentStatus(status string) bool {
	for _, s := range ValidPaymentStatuses() {
		if s == status {
			return true
		}
	}
	return false
}

// ValidPaymentMethods returns all valid payment methods.
func ValidPaymentMethods() []string {
	return []string{
		PaymentMethodCreditCard,
		PaymentMethodDebitCard,
		PaymentMethodBankTransfer,
		PaymentMethodWallet,
	}
}

// IsValidPaymentMethod checks whether the given method is a valid payment method.
func IsValidPaymentMethod(method string) bool {
	for _, m := range ValidPaymentMethods() {
		if m == method {
			return true
		}
	}
	return false
}
