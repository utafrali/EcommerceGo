package domain

import (
	"time"
)

// Saga step status constants.
const (
	SagaStepPending     = "pending"
	SagaStepCompleted   = "completed"
	SagaStepFailed      = "failed"
	SagaStepCompensated = "compensated"
)

// SagaStep tracks the execution status of a single step in the checkout saga.
type SagaStep struct {
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	Error      string    `json:"error,omitempty"`
	ExecutedAt time.Time `json:"executed_at,omitempty"`
}

// NewSagaStep creates a new saga step in the pending state.
func NewSagaStep(name string) SagaStep {
	return SagaStep{
		Name:   name,
		Status: SagaStepPending,
	}
}

// Complete marks the saga step as successfully completed.
func (s *SagaStep) Complete() {
	s.Status = SagaStepCompleted
	s.ExecutedAt = time.Now().UTC()
}

// Fail marks the saga step as failed with the given error message.
func (s *SagaStep) Fail(err string) {
	s.Status = SagaStepFailed
	s.Error = err
	s.ExecutedAt = time.Now().UTC()
}

// Compensate marks the saga step as compensated (rolled back).
func (s *SagaStep) Compensate() {
	s.Status = SagaStepCompensated
	s.ExecutedAt = time.Now().UTC()
}

// Saga step name constants for the checkout process.
const (
	SagaStepReserveInventory = "reserve_inventory"
	SagaStepCreateOrder      = "create_order"
	SagaStepInitiatePayment  = "initiate_payment"
)
