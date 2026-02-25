package provider

import (
	"context"
)

// ChargeInput holds the parameters for charging a payment.
type ChargeInput struct {
	Amount      int64
	Currency    string
	Method      string
	Description string
	Metadata    map[string]any
}

// ChargeResult holds the result of a charge operation from the payment provider.
type ChargeResult struct {
	ProviderPaymentID string
	Status            string // "succeeded" or "failed"
	FailureReason     string
}

// RefundInput holds the parameters for refunding a payment.
type RefundInput struct {
	ProviderPaymentID string
	Amount            int64
	Currency          string
	Reason            string
}

// RefundResult holds the result of a refund operation from the payment provider.
type RefundResult struct {
	ProviderRefundID string
	Status           string
	FailureReason    string
}

// Provider defines the interface for payment provider integrations.
type Provider interface {
	// Name returns the provider name (e.g., "mock", "stripe").
	Name() string

	// Charge processes a payment charge through the provider.
	Charge(ctx context.Context, input *ChargeInput) (*ChargeResult, error)

	// Refund processes a refund through the provider.
	Refund(ctx context.Context, input *RefundInput) (*RefundResult, error)
}
