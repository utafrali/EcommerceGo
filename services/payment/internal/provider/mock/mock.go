package mock

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/utafrali/EcommerceGo/services/payment/internal/provider"
)

// Provider is a mock payment provider that always succeeds.
// It is intended for development and testing purposes.
type Provider struct{}

// NewProvider creates a new mock payment provider.
func NewProvider() *Provider {
	return &Provider{}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "mock"
}

// Charge simulates a payment charge that always succeeds.
func (p *Provider) Charge(_ context.Context, _ *provider.ChargeInput) (*provider.ChargeResult, error) {
	// Simulate a small processing delay.
	time.Sleep(50 * time.Millisecond)

	return &provider.ChargeResult{
		ProviderPaymentID: "mock_pay_" + uuid.New().String(),
		Status:            "succeeded",
		FailureReason:     "",
	}, nil
}

// Refund simulates a payment refund that always succeeds.
func (p *Provider) Refund(_ context.Context, _ *provider.RefundInput) (*provider.RefundResult, error) {
	// Simulate a small processing delay.
	time.Sleep(50 * time.Millisecond)

	return &provider.RefundResult{
		ProviderRefundID: "mock_ref_" + uuid.New().String(),
		Status:           "succeeded",
		FailureReason:    "",
	}, nil
}
