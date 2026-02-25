package domain

import (
	"time"
)

// Refund status constants.
const (
	RefundStatusPending    = "pending"
	RefundStatusProcessing = "processing"
	RefundStatusSucceeded  = "succeeded"
	RefundStatusFailed     = "failed"
)

// Refund represents a refund against a payment.
type Refund struct {
	ID            string    `json:"id"`
	PaymentID     string    `json:"payment_id"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	Reason        string    `json:"reason"`
	ProviderRefID string    `json:"provider_refund_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ValidRefundStatuses returns all valid refund statuses.
func ValidRefundStatuses() []string {
	return []string{
		RefundStatusPending,
		RefundStatusProcessing,
		RefundStatusSucceeded,
		RefundStatusFailed,
	}
}

// IsValidRefundStatus checks whether the given status is a valid refund status.
func IsValidRefundStatus(status string) bool {
	for _, s := range ValidRefundStatuses() {
		if s == status {
			return true
		}
	}
	return false
}
