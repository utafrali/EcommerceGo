package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/payment/internal/domain"
	"github.com/utafrali/EcommerceGo/services/payment/internal/event"
	"github.com/utafrali/EcommerceGo/services/payment/internal/provider"
	"github.com/utafrali/EcommerceGo/services/payment/internal/repository"
)

// PaymentService implements the business logic for payment operations.
type PaymentService struct {
	repo     repository.PaymentRepository
	provider provider.Provider
	producer *event.Producer
	logger   *slog.Logger
}

// NewPaymentService creates a new payment service.
func NewPaymentService(
	repo repository.PaymentRepository,
	prov provider.Provider,
	producer *event.Producer,
	logger *slog.Logger,
) *PaymentService {
	return &PaymentService{
		repo:     repo,
		provider: prov,
		producer: producer,
		logger:   logger,
	}
}

// CreatePaymentInput holds the parameters for creating a payment.
type CreatePaymentInput struct {
	CheckoutID string `json:"checkout_id" validate:"required,uuid"`
	OrderID    string `json:"order_id" validate:"required,uuid"`
	UserID     string `json:"user_id" validate:"required,uuid"`
	Amount     int64  `json:"amount" validate:"required,gt=0"`
	Currency   string `json:"currency" validate:"required,len=3"`
	Method     string `json:"method" validate:"required,oneof=credit_card debit_card bank_transfer wallet"`
}

// RefundPaymentInput holds the parameters for refunding a payment.
type RefundPaymentInput struct {
	Amount int64  `json:"amount" validate:"required,gt=0"`
	Reason string `json:"reason" validate:"required,min=3"`
}

// CreatePayment creates a new payment record in the pending state.
func (s *PaymentService) CreatePayment(ctx context.Context, input *CreatePaymentInput) (*domain.Payment, error) {
	if input.CheckoutID == "" {
		return nil, apperrors.InvalidInput("checkout_id is required")
	}
	if input.OrderID == "" {
		return nil, apperrors.InvalidInput("order_id is required")
	}
	if input.UserID == "" {
		return nil, apperrors.InvalidInput("user_id is required")
	}
	if input.Amount <= 0 {
		return nil, apperrors.InvalidInput("amount must be greater than zero")
	}
	if len(input.Currency) != 3 {
		return nil, apperrors.InvalidInput("currency must be a 3-letter ISO code")
	}
	if !domain.IsValidPaymentMethod(input.Method) {
		return nil, apperrors.InvalidInput(fmt.Sprintf("invalid payment method %q", input.Method))
	}

	now := time.Now().UTC()
	payment := &domain.Payment{
		ID:           uuid.New().String(),
		CheckoutID:   input.CheckoutID,
		OrderID:      input.OrderID,
		UserID:       input.UserID,
		Amount:       input.Amount,
		Currency:     strings.ToUpper(input.Currency),
		Status:       domain.PaymentStatusPending,
		Method:       input.Method,
		ProviderName: s.provider.Name(),
		Metadata:     make(map[string]any),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	s.logger.InfoContext(ctx, "payment created",
		slog.String("payment_id", payment.ID),
		slog.String("checkout_id", payment.CheckoutID),
		slog.String("status", payment.Status),
	)

	return payment, nil
}

// ProcessPayment processes a pending payment by calling the provider.
func (s *PaymentService) ProcessPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("get payment for processing: %w", err)
	}

	if payment.Status != domain.PaymentStatusPending {
		return nil, apperrors.InvalidInput(fmt.Sprintf("payment cannot be processed in status %q", payment.Status))
	}

	// Mark as processing.
	payment.Status = domain.PaymentStatusProcessing
	if err := s.repo.Update(ctx, payment); err != nil {
		return nil, fmt.Errorf("update payment to processing: %w", err)
	}

	// Call the provider to charge.
	chargeInput := &provider.ChargeInput{
		Amount:      payment.Amount,
		Currency:    payment.Currency,
		Method:      payment.Method,
		Description: fmt.Sprintf("Payment for order %s", payment.OrderID),
		Metadata:    payment.Metadata,
	}

	result, err := s.provider.Charge(ctx, chargeInput)
	if err != nil {
		// Provider error: mark payment as failed.
		payment.Status = domain.PaymentStatusFailed
		payment.FailureReason = err.Error()
		if updateErr := s.repo.Update(ctx, payment); updateErr != nil {
			s.logger.ErrorContext(ctx, "failed to update payment after provider error",
				slog.String("payment_id", payment.ID),
				slog.String("error", updateErr.Error()),
			)
		}

		if s.producer != nil {
			if pubErr := s.producer.PublishPaymentFailed(ctx, payment); pubErr != nil {
				s.logger.ErrorContext(ctx, "failed to publish payment.failed event",
					slog.String("payment_id", payment.ID),
					slog.String("error", pubErr.Error()),
				)
			}
		}

		return nil, apperrors.PaymentFailed(fmt.Sprintf("provider charge failed: %s", err.Error()))
	}

	// Update payment based on charge result.
	payment.ProviderPayID = result.ProviderPaymentID

	if result.Status == "succeeded" {
		payment.Status = domain.PaymentStatusSucceeded
	} else {
		payment.Status = domain.PaymentStatusFailed
		payment.FailureReason = result.FailureReason
	}

	if err := s.repo.Update(ctx, payment); err != nil {
		return nil, fmt.Errorf("update payment after charge: %w", err)
	}

	// Publish the appropriate event.
	if s.producer != nil {
		if payment.Status == domain.PaymentStatusSucceeded {
			if pubErr := s.producer.PublishPaymentSucceeded(ctx, payment); pubErr != nil {
				s.logger.ErrorContext(ctx, "failed to publish payment.succeeded event",
					slog.String("payment_id", payment.ID),
					slog.String("error", pubErr.Error()),
				)
			}
		} else {
			if pubErr := s.producer.PublishPaymentFailed(ctx, payment); pubErr != nil {
				s.logger.ErrorContext(ctx, "failed to publish payment.failed event",
					slog.String("payment_id", payment.ID),
					slog.String("error", pubErr.Error()),
				)
			}
		}
	}

	s.logger.InfoContext(ctx, "payment processed",
		slog.String("payment_id", payment.ID),
		slog.String("status", payment.Status),
	)

	if payment.Status == domain.PaymentStatusFailed {
		return nil, apperrors.PaymentFailed(fmt.Sprintf("charge declined: %s", payment.FailureReason))
	}

	return payment, nil
}

// GetPayment retrieves a payment by its ID.
func (s *PaymentService) GetPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("get payment by id: %w", err)
	}
	return payment, nil
}

// GetPaymentByCheckoutID retrieves a payment by the associated checkout ID.
func (s *PaymentService) GetPaymentByCheckoutID(ctx context.Context, checkoutID string) (*domain.Payment, error) {
	payment, err := s.repo.GetByCheckoutID(ctx, checkoutID)
	if err != nil {
		return nil, fmt.Errorf("get payment by checkout id: %w", err)
	}
	return payment, nil
}

// RefundPayment processes a refund for a payment.
func (s *PaymentService) RefundPayment(ctx context.Context, paymentID string, input *RefundPaymentInput) (*domain.Refund, error) {
	if input.Amount <= 0 {
		return nil, apperrors.InvalidInput("refund amount must be greater than zero")
	}
	if len(input.Reason) < 3 {
		return nil, apperrors.InvalidInput("refund reason must be at least 3 characters")
	}

	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("get payment for refund: %w", err)
	}

	// Only succeeded or partially refunded payments can be refunded.
	if payment.Status != domain.PaymentStatusSucceeded && payment.Status != domain.PaymentStatusPartiallyRefunded {
		return nil, apperrors.InvalidInput(fmt.Sprintf("payment cannot be refunded in status %q", payment.Status))
	}

	// Calculate total already-refunded amount.
	existingRefunds, err := s.repo.ListRefundsByPaymentID(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("list refunds for payment: %w", err)
	}

	var totalRefunded int64
	for _, r := range existingRefunds {
		if r.Status == domain.RefundStatusSucceeded || r.Status == domain.RefundStatusPending || r.Status == domain.RefundStatusProcessing {
			totalRefunded += r.Amount
		}
	}

	if totalRefunded+input.Amount > payment.Amount {
		return nil, apperrors.InvalidInput(fmt.Sprintf(
			"refund amount %d exceeds available amount %d (already refunded: %d)",
			input.Amount, payment.Amount-totalRefunded, totalRefunded,
		))
	}

	// Create refund record.
	now := time.Now().UTC()
	refund := &domain.Refund{
		ID:        uuid.New().String(),
		PaymentID: paymentID,
		Amount:    input.Amount,
		Currency:  payment.Currency,
		Status:    domain.RefundStatusProcessing,
		Reason:    input.Reason,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.CreateRefund(ctx, refund); err != nil {
		return nil, fmt.Errorf("create refund: %w", err)
	}

	// Call the provider to process the refund.
	refundInput := &provider.RefundInput{
		ProviderPaymentID: payment.ProviderPayID,
		Amount:            input.Amount,
		Currency:          payment.Currency,
		Reason:            input.Reason,
	}

	result, err := s.provider.Refund(ctx, refundInput)
	if err != nil {
		refund.Status = domain.RefundStatusFailed
		if updateErr := s.repo.UpdateRefund(ctx, refund); updateErr != nil {
			s.logger.ErrorContext(ctx, "failed to update refund after provider error",
				slog.String("refund_id", refund.ID),
				slog.String("error", updateErr.Error()),
			)
		}
		return nil, fmt.Errorf("provider refund: %w", err)
	}

	// Update refund with provider result.
	refund.ProviderRefID = result.ProviderRefundID
	if result.Status == "succeeded" {
		refund.Status = domain.RefundStatusSucceeded
	} else {
		refund.Status = domain.RefundStatusFailed
	}

	if err := s.repo.UpdateRefund(ctx, refund); err != nil {
		return nil, fmt.Errorf("update refund after provider response: %w", err)
	}

	// Update payment status based on total refunded amount.
	if refund.Status == domain.RefundStatusSucceeded {
		newTotalRefunded := totalRefunded + input.Amount
		if newTotalRefunded >= payment.Amount {
			payment.Status = domain.PaymentStatusRefunded
		} else {
			payment.Status = domain.PaymentStatusPartiallyRefunded
		}

		if err := s.repo.Update(ctx, payment); err != nil {
			return nil, fmt.Errorf("update payment after refund: %w", err)
		}

		if s.producer != nil {
			if pubErr := s.producer.PublishPaymentRefunded(ctx, payment, refund); pubErr != nil {
				s.logger.ErrorContext(ctx, "failed to publish payment.refunded event",
					slog.String("payment_id", payment.ID),
					slog.String("refund_id", refund.ID),
					slog.String("error", pubErr.Error()),
				)
			}
		}
	}

	s.logger.InfoContext(ctx, "payment refunded",
		slog.String("payment_id", paymentID),
		slog.String("refund_id", refund.ID),
		slog.String("refund_status", refund.Status),
		slog.Int64("refund_amount", refund.Amount),
	)

	return refund, nil
}

// ListPaymentsByUser returns a paginated list of payments for a user.
func (s *PaymentService) ListPaymentsByUser(ctx context.Context, userID string, page, perPage int) ([]domain.Payment, int, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	offset := (page - 1) * perPage

	payments, total, err := s.repo.ListByUserID(ctx, userID, offset, perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("list payments by user: %w", err)
	}

	return payments, total, nil
}
