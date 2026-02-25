package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/checkout/internal/domain"
	"github.com/utafrali/EcommerceGo/services/checkout/internal/event"
	"github.com/utafrali/EcommerceGo/services/checkout/internal/repository"
)

const (
	// checkoutExpiryDuration is how long a checkout session remains valid.
	checkoutExpiryDuration = 30 * time.Minute
)

// CheckoutService implements the business logic for checkout operations.
type CheckoutService struct {
	repo     repository.CheckoutRepository
	producer *event.Producer
	logger   *slog.Logger
}

// NewCheckoutService creates a new checkout service.
func NewCheckoutService(repo repository.CheckoutRepository, producer *event.Producer, logger *slog.Logger) *CheckoutService {
	return &CheckoutService{
		repo:     repo,
		producer: producer,
		logger:   logger,
	}
}

// InitiateCheckoutInput holds the parameters for initiating a checkout.
type InitiateCheckoutInput struct {
	Items    []CheckoutItemInput `json:"items" validate:"required,min=1,dive"`
	Currency string              `json:"currency" validate:"required,len=3"`
}

// CheckoutItemInput represents a single item in the initiate checkout request.
type CheckoutItemInput struct {
	ProductID string `json:"product_id" validate:"required,uuid"`
	VariantID string `json:"variant_id" validate:"required,uuid"`
	Name      string `json:"name" validate:"required"`
	SKU       string `json:"sku" validate:"required"`
	Price     int64  `json:"price" validate:"required,gt=0"`
	Quantity  int    `json:"quantity" validate:"required,gt=0"`
}

// InitiateCheckout creates a new checkout session from cart items with a 30-minute expiry.
func (s *CheckoutService) InitiateCheckout(ctx context.Context, userID string, input *InitiateCheckoutInput) (*domain.CheckoutSession, error) {
	if userID == "" {
		return nil, apperrors.InvalidInput("user id is required")
	}
	if input == nil {
		return nil, apperrors.InvalidInput("checkout input is required")
	}
	if len(input.Items) == 0 {
		return nil, apperrors.InvalidInput("at least one item is required")
	}
	if len(input.Currency) != 3 {
		return nil, apperrors.InvalidInput("currency must be a 3-letter ISO code")
	}

	// Validate items.
	for i, item := range input.Items {
		if item.ProductID == "" {
			return nil, apperrors.InvalidInput(fmt.Sprintf("item %d: product_id is required", i))
		}
		if item.VariantID == "" {
			return nil, apperrors.InvalidInput(fmt.Sprintf("item %d: variant_id is required", i))
		}
		if item.Name == "" {
			return nil, apperrors.InvalidInput(fmt.Sprintf("item %d: name is required", i))
		}
		if item.SKU == "" {
			return nil, apperrors.InvalidInput(fmt.Sprintf("item %d: sku is required", i))
		}
		if item.Price <= 0 {
			return nil, apperrors.InvalidInput(fmt.Sprintf("item %d: price must be greater than 0", i))
		}
		if item.Quantity <= 0 {
			return nil, apperrors.InvalidInput(fmt.Sprintf("item %d: quantity must be greater than 0", i))
		}
	}

	now := time.Now().UTC()

	// Build checkout items from input.
	items := make([]domain.CheckoutItem, len(input.Items))
	for i, item := range input.Items {
		items[i] = domain.CheckoutItem{
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			Name:      item.Name,
			SKU:       item.SKU,
			Price:     item.Price,
			Quantity:  item.Quantity,
		}
	}

	session := &domain.CheckoutSession{
		ID:        uuid.New().String(),
		UserID:    userID,
		Status:    domain.StatusInitiated,
		Items:     items,
		Currency:  strings.ToUpper(input.Currency),
		ExpiresAt: now.Add(checkoutExpiryDuration),
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Calculate amounts.
	session.SubtotalAmount = session.CalculateSubtotal()
	session.TotalAmount = session.CalculateTotal()

	if err := s.repo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create checkout session: %w", err)
	}

	// Publish event; log but do not fail on error.
	if err := s.producer.PublishCheckoutInitiated(ctx, session); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish checkout.initiated event",
			slog.String("checkout_id", session.ID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "checkout session initiated",
		slog.String("checkout_id", session.ID),
		slog.String("user_id", userID),
		slog.Int64("total_amount", session.TotalAmount),
	)

	return session, nil
}

// GetCheckout retrieves a checkout session by its ID.
func (s *CheckoutService) GetCheckout(ctx context.Context, sessionID string) (*domain.CheckoutSession, error) {
	session, err := s.repo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get checkout session: %w", err)
	}
	return session, nil
}

// SetShippingAddress sets the shipping address on a checkout session.
func (s *CheckoutService) SetShippingAddress(ctx context.Context, sessionID string, address *domain.Address) (*domain.CheckoutSession, error) {
	if address == nil {
		return nil, apperrors.InvalidInput("shipping address is required")
	}
	if address.FullName == "" {
		return nil, apperrors.InvalidInput("full_name is required")
	}
	if address.AddressLine == "" {
		return nil, apperrors.InvalidInput("address_line is required")
	}
	if address.City == "" {
		return nil, apperrors.InvalidInput("city is required")
	}
	if address.PostalCode == "" {
		return nil, apperrors.InvalidInput("postal_code is required")
	}
	if address.Country == "" {
		return nil, apperrors.InvalidInput("country is required")
	}

	session, err := s.repo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get checkout for shipping address: %w", err)
	}

	if session.IsTerminal() {
		return nil, apperrors.InvalidInput("cannot modify a completed, failed, or expired checkout")
	}

	if session.IsExpired() {
		session.Status = domain.StatusExpired
		_ = s.repo.Update(ctx, session)
		return nil, apperrors.InvalidInput("checkout session has expired")
	}

	session.ShippingAddress = address

	if err := s.repo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update checkout shipping address: %w", err)
	}

	s.logger.InfoContext(ctx, "shipping address set",
		slog.String("checkout_id", sessionID),
	)

	return session, nil
}

// SetPaymentMethod sets the payment method on a checkout session.
func (s *CheckoutService) SetPaymentMethod(ctx context.Context, sessionID, method string) (*domain.CheckoutSession, error) {
	if method == "" {
		return nil, apperrors.InvalidInput("payment method is required")
	}

	session, err := s.repo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get checkout for payment method: %w", err)
	}

	if session.IsTerminal() {
		return nil, apperrors.InvalidInput("cannot modify a completed, failed, or expired checkout")
	}

	if session.IsExpired() {
		session.Status = domain.StatusExpired
		_ = s.repo.Update(ctx, session)
		return nil, apperrors.InvalidInput("checkout session has expired")
	}

	session.PaymentMethod = method

	if err := s.repo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update checkout payment method: %w", err)
	}

	s.logger.InfoContext(ctx, "payment method set",
		slog.String("checkout_id", sessionID),
		slog.String("payment_method", method),
	)

	return session, nil
}

// ProcessCheckout orchestrates the checkout saga: reserve inventory -> create order -> initiate payment.
func (s *CheckoutService) ProcessCheckout(ctx context.Context, sessionID string) (*domain.CheckoutSession, error) {
	session, err := s.repo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get checkout for processing: %w", err)
	}

	if session.IsTerminal() {
		return nil, apperrors.InvalidInput("cannot process a completed, failed, or expired checkout")
	}

	if session.IsExpired() {
		session.Status = domain.StatusExpired
		_ = s.repo.Update(ctx, session)
		return nil, apperrors.InvalidInput("checkout session has expired")
	}

	if session.ShippingAddress == nil {
		return nil, apperrors.InvalidInput("shipping address must be set before processing")
	}

	if session.PaymentMethod == "" {
		return nil, apperrors.InvalidInput("payment method must be set before processing")
	}

	// Initialize saga steps.
	steps := []domain.SagaStep{
		domain.NewSagaStep(domain.SagaStepReserveInventory),
		domain.NewSagaStep(domain.SagaStepCreateOrder),
		domain.NewSagaStep(domain.SagaStepInitiatePayment),
	}

	// Step 1: Reserve inventory.
	// In a real system, this would call the Inventory Service via HTTP.
	// For now, we simulate success and assign reservation IDs.
	for i := range session.Items {
		session.Items[i].ReservationID = uuid.New().String()
	}
	steps[0].Complete()
	session.Status = domain.StatusItemsReserved

	if err := s.repo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update checkout after inventory reservation: %w", err)
	}

	// Step 2: Create order.
	// In a real system, this would call the Order Service via HTTP.
	session.OrderID = uuid.New().String()
	steps[1].Complete()

	if err := s.repo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update checkout after order creation: %w", err)
	}

	// Step 3: Initiate payment.
	// In a real system, this would call the Payment Service via HTTP.
	session.PaymentID = uuid.New().String()
	session.Status = domain.StatusPaymentProcessing
	steps[2].Complete()

	// Mark checkout as completed.
	session.Status = domain.StatusCompleted

	if err := s.repo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update checkout after payment: %w", err)
	}

	// Publish completed event; log but do not fail on error.
	if err := s.producer.PublishCheckoutCompleted(ctx, session); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish checkout.completed event",
			slog.String("checkout_id", session.ID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "checkout completed",
		slog.String("checkout_id", session.ID),
		slog.String("order_id", session.OrderID),
		slog.String("payment_id", session.PaymentID),
		slog.Int64("total_amount", session.TotalAmount),
	)

	return session, nil
}

// CancelCheckout cancels a checkout session and performs compensating actions.
func (s *CheckoutService) CancelCheckout(ctx context.Context, sessionID string) (*domain.CheckoutSession, error) {
	session, err := s.repo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get checkout for cancellation: %w", err)
	}

	if session.Status == domain.StatusCompleted {
		return nil, apperrors.InvalidInput("cannot cancel a completed checkout")
	}

	if session.Status == domain.StatusFailed || session.Status == domain.StatusExpired {
		return nil, apperrors.InvalidInput("checkout is already cancelled or expired")
	}

	// Compensating actions:
	// In a real system, we would release inventory reservations and cancel the order.
	// For now, we clear reservation IDs to simulate compensation.
	for i := range session.Items {
		session.Items[i].ReservationID = ""
	}

	session.Status = domain.StatusFailed
	session.FailureReason = "cancelled by user"

	if err := s.repo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update checkout for cancellation: %w", err)
	}

	// Publish failed event; log but do not fail on error.
	if err := s.producer.PublishCheckoutFailed(ctx, session); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish checkout.failed event",
			slog.String("checkout_id", session.ID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "checkout cancelled",
		slog.String("checkout_id", session.ID),
		slog.String("user_id", session.UserID),
	)

	return session, nil
}
