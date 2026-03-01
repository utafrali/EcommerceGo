package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/pkg/httpclient"
	"github.com/utafrali/EcommerceGo/services/checkout/internal/domain"
	"github.com/utafrali/EcommerceGo/services/checkout/internal/event"
	"github.com/utafrali/EcommerceGo/services/checkout/internal/repository"
)

const (
	// checkoutExpiryDuration is how long a checkout session remains valid.
	checkoutExpiryDuration = 30 * time.Minute
)

// CircuitOpenFallback is a fallback function for the checkout saga's circuit breaker.
// When the circuit is open, it returns a structured error with a retry hint
// instead of letting the raw ErrCircuitOpen propagate.
func CircuitOpenFallback(_ context.Context, _ error) (*http.Response, error) {
	return nil, apperrors.ServiceUnavailable("downstream service is temporarily unavailable, please retry after 30 seconds")
}

// HTTPDoer is the interface for executing HTTP requests.
// Both httpclient.Client and httpclient.CircuitBreakerClient satisfy this.
type HTTPDoer interface {
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
}

// SagaTimeouts holds per-step timeout configuration for the checkout saga.
// A zero value means no per-step timeout (inherits the parent context timeout).
type SagaTimeouts struct {
	InventoryTimeout time.Duration
	OrderTimeout     time.Duration
	PaymentTimeout   time.Duration
}

// CheckoutService implements the business logic for checkout operations.
type CheckoutService struct {
	repo                repository.CheckoutRepository
	producer            *event.Producer
	logger              *slog.Logger
	httpClient          HTTPDoer
	inventoryServiceURL string
	orderServiceURL     string
	paymentServiceURL   string
	sagaTimeouts        SagaTimeouts
}

// NewCheckoutService creates a new checkout service.
func NewCheckoutService(
	repo repository.CheckoutRepository,
	producer *event.Producer,
	logger *slog.Logger,
	httpClient HTTPDoer,
	inventoryServiceURL, orderServiceURL, paymentServiceURL string,
	sagaTimeouts SagaTimeouts,
) *CheckoutService {
	return &CheckoutService{
		repo:                repo,
		producer:            producer,
		logger:              logger,
		httpClient:          httpClient,
		inventoryServiceURL: inventoryServiceURL,
		orderServiceURL:     orderServiceURL,
		paymentServiceURL:   paymentServiceURL,
		sagaTimeouts:        sagaTimeouts,
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

	if session.IsExpired() {
		session.Status = domain.StatusExpired
		if err := s.repo.Update(ctx, session); err != nil {
			s.logger.ErrorContext(ctx, "failed to update expired checkout session",
				slog.String("checkout_id", session.ID),
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("update expired checkout session: %w", err)
		}
		return nil, apperrors.InvalidInput("checkout session has expired")
	}

	if session.Status != domain.StatusInitiated {
		return nil, apperrors.InvalidInput("shipping address can only be set while checkout is in initiated state")
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

	if session.IsExpired() {
		session.Status = domain.StatusExpired
		if err := s.repo.Update(ctx, session); err != nil {
			s.logger.ErrorContext(ctx, "failed to update expired checkout session",
				slog.String("checkout_id", session.ID),
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("update expired checkout session: %w", err)
		}
		return nil, apperrors.InvalidInput("checkout session has expired")
	}

	if session.Status != domain.StatusInitiated {
		return nil, apperrors.InvalidInput("payment method can only be set while checkout is in initiated state")
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
		if err := s.repo.Update(ctx, session); err != nil {
			s.logger.ErrorContext(ctx, "failed to update expired checkout session",
				slog.String("checkout_id", session.ID),
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("update expired checkout session: %w", err)
		}
		return nil, apperrors.Gone("checkout session has expired")
	}

	if session.ShippingAddress == nil {
		return nil, apperrors.InvalidInput("shipping address must be set before processing")
	}

	if session.PaymentMethod == "" {
		return nil, apperrors.InvalidInput("payment method must be set before processing")
	}

	// Defense-in-depth: re-validate subtotal against embedded items.
	// Items are embedded in the session so they should not change, but verify
	// to catch any potential data corruption.
	recalculatedSubtotal := session.CalculateSubtotal()
	if recalculatedSubtotal != session.SubtotalAmount {
		s.logger.WarnContext(ctx, "subtotal mismatch detected during checkout processing",
			slog.String("checkout_id", session.ID),
			slog.Int64("stored_subtotal", session.SubtotalAmount),
			slog.Int64("recalculated_subtotal", recalculatedSubtotal),
		)
		// Update the subtotal and total to the recalculated values.
		session.SubtotalAmount = recalculatedSubtotal
		session.TotalAmount = session.CalculateTotal()
	}

	// Initialize saga steps.
	steps := []domain.SagaStep{
		domain.NewSagaStep(domain.SagaStepReserveInventory),
		domain.NewSagaStep(domain.SagaStepCreateOrder),
		domain.NewSagaStep(domain.SagaStepInitiatePayment),
	}

	// Step 1: Reserve inventory.
	reservationIDs, err := s.reserveInventory(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("reserve inventory: %w", err)
	}
	for i, reservationID := range reservationIDs {
		if i < len(session.Items) {
			session.Items[i].ReservationID = reservationID
		}
	}
	steps[0].Complete()
	session.Status = domain.StatusItemsReserved

	if err := s.repo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update checkout after inventory reservation: %w", err)
	}

	// Step 2: Create order.
	orderID, err := s.createOrder(ctx, session)
	if err != nil {
		// Compensate: release inventory reservations
		_ = s.releaseInventoryReservations(ctx, reservationIDs)
		return nil, fmt.Errorf("create order: %w", err)
	}
	session.OrderID = orderID
	steps[1].Complete()

	if err := s.repo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update checkout after order creation: %w", err)
	}

	// Step 3: Initiate payment.
	paymentID, err := s.initiatePayment(ctx, session)
	if err != nil {
		// Compensate: cancel order and release inventory
		_ = s.cancelOrder(ctx, orderID)
		_ = s.releaseInventoryReservations(ctx, reservationIDs)
		return nil, fmt.Errorf("initiate payment: %w", err)
	}
	session.PaymentID = paymentID
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

// reserveInventory calls the inventory service to reserve stock for all checkout items.
func (s *CheckoutService) reserveInventory(ctx context.Context, session *domain.CheckoutSession) ([]string, error) {
	if s.sagaTimeouts.InventoryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.sagaTimeouts.InventoryTimeout)
		defer cancel()
	}

	type reserveRequest struct {
		Items []struct {
			VariantID string `json:"variant_id"`
			Quantity  int    `json:"quantity"`
		} `json:"items"`
		CheckoutID string `json:"checkout_id"`
	}

	type reserveResponse struct {
		ReservationIDs []string `json:"reservation_ids"`
	}

	req := reserveRequest{
		Items:      make([]struct {
			VariantID string `json:"variant_id"`
			Quantity  int    `json:"quantity"`
		}, len(session.Items)),
		CheckoutID: session.ID,
	}

	for i, item := range session.Items {
		req.Items[i].VariantID = item.VariantID
		req.Items[i].Quantity = item.Quantity
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal reserve request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.inventoryServiceURL+"/api/inventory/reserve", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create reserve request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("call inventory service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, httpclient.ParseResponseError(resp, "inventory")
	}

	var reserveResp reserveResponse
	if err := json.NewDecoder(resp.Body).Decode(&reserveResp); err != nil {
		return nil, fmt.Errorf("decode reserve response: %w", err)
	}

	s.logger.InfoContext(ctx, "inventory reserved",
		slog.String("checkout_id", session.ID),
		slog.Int("items_count", len(reserveResp.ReservationIDs)),
	)

	return reserveResp.ReservationIDs, nil
}

// createOrder calls the order service to create an order from the checkout session.
func (s *CheckoutService) createOrder(ctx context.Context, session *domain.CheckoutSession) (string, error) {
	if s.sagaTimeouts.OrderTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.sagaTimeouts.OrderTimeout)
		defer cancel()
	}

	type orderItem struct {
		ProductID string `json:"product_id"`
		VariantID string `json:"variant_id"`
		Name      string `json:"name"`
		SKU       string `json:"sku"`
		Price     int64  `json:"price"`
		Quantity  int    `json:"quantity"`
	}

	type createOrderRequest struct {
		UserID          string      `json:"user_id"`
		Items           []orderItem `json:"items"`
		Currency        string      `json:"currency"`
		SubtotalAmount  int64       `json:"subtotal_amount"`
		TotalAmount     int64       `json:"total_amount"`
		ShippingAddress struct {
			FullName    string `json:"full_name"`
			AddressLine string `json:"address_line"`
			City        string `json:"city"`
			PostalCode  string `json:"postal_code"`
			Country     string `json:"country"`
		} `json:"shipping_address"`
		CheckoutID string `json:"checkout_id"`
	}

	type createOrderResponse struct {
		OrderID string `json:"order_id"`
	}

	req := createOrderRequest{
		UserID:         session.UserID,
		Items:          make([]orderItem, len(session.Items)),
		Currency:       session.Currency,
		SubtotalAmount: session.SubtotalAmount,
		TotalAmount:    session.TotalAmount,
		CheckoutID:     session.ID,
	}

	for i, item := range session.Items {
		req.Items[i] = orderItem{
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			Name:      item.Name,
			SKU:       item.SKU,
			Price:     item.Price,
			Quantity:  item.Quantity,
		}
	}

	if session.ShippingAddress != nil {
		req.ShippingAddress.FullName = session.ShippingAddress.FullName
		req.ShippingAddress.AddressLine = session.ShippingAddress.AddressLine
		req.ShippingAddress.City = session.ShippingAddress.City
		req.ShippingAddress.PostalCode = session.ShippingAddress.PostalCode
		req.ShippingAddress.Country = session.ShippingAddress.Country
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal create order request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.orderServiceURL+"/api/orders", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create order request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(ctx, httpReq)
	if err != nil {
		return "", fmt.Errorf("call order service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", httpclient.ParseResponseError(resp, "order")
	}

	var orderResp createOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResp); err != nil {
		return "", fmt.Errorf("decode order response: %w", err)
	}

	s.logger.InfoContext(ctx, "order created",
		slog.String("checkout_id", session.ID),
		slog.String("order_id", orderResp.OrderID),
	)

	return orderResp.OrderID, nil
}

// initiatePayment calls the payment service to start payment processing.
func (s *CheckoutService) initiatePayment(ctx context.Context, session *domain.CheckoutSession) (string, error) {
	if s.sagaTimeouts.PaymentTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.sagaTimeouts.PaymentTimeout)
		defer cancel()
	}

	type initiatePaymentRequest struct {
		OrderID       string `json:"order_id"`
		UserID        string `json:"user_id"`
		Amount        int64  `json:"amount"`
		Currency      string `json:"currency"`
		PaymentMethod string `json:"payment_method"`
	}

	type initiatePaymentResponse struct {
		PaymentID string `json:"payment_id"`
	}

	req := initiatePaymentRequest{
		OrderID:       session.OrderID,
		UserID:        session.UserID,
		Amount:        session.TotalAmount,
		Currency:      session.Currency,
		PaymentMethod: session.PaymentMethod,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal payment request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.paymentServiceURL+"/api/payments", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create payment request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(ctx, httpReq)
	if err != nil {
		return "", fmt.Errorf("call payment service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", httpclient.ParseResponseError(resp, "payment")
	}

	var paymentResp initiatePaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&paymentResp); err != nil {
		return "", fmt.Errorf("decode payment response: %w", err)
	}

	s.logger.InfoContext(ctx, "payment initiated",
		slog.String("checkout_id", session.ID),
		slog.String("payment_id", paymentResp.PaymentID),
	)

	return paymentResp.PaymentID, nil
}

// releaseInventoryReservations is a compensating action to release inventory holds.
func (s *CheckoutService) releaseInventoryReservations(ctx context.Context, reservationIDs []string) error {
	type releaseRequest struct {
		ReservationIDs []string `json:"reservation_ids"`
	}

	req := releaseRequest{
		ReservationIDs: reservationIDs,
	}

	body, err := json.Marshal(req)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal release request", slog.String("error", err.Error()))
		return fmt.Errorf("marshal release request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.inventoryServiceURL+"/api/inventory/release", bytes.NewReader(body))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create release request", slog.String("error", err.Error()))
		return fmt.Errorf("create release request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(ctx, httpReq)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to call inventory service for release", slog.String("error", err.Error()))
		return fmt.Errorf("call inventory service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return httpclient.ParseResponseError(resp, "inventory")
	}

	s.logger.InfoContext(ctx, "inventory reservations released",
		slog.Int("count", len(reservationIDs)),
	)

	return nil
}

// cancelOrder is a compensating action to cancel an order.
func (s *CheckoutService) cancelOrder(ctx context.Context, orderID string) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.orderServiceURL+"/api/orders/"+orderID+"/cancel", nil)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create cancel order request", slog.String("error", err.Error()))
		return fmt.Errorf("create cancel order request: %w", err)
	}

	resp, err := s.httpClient.Do(ctx, httpReq)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to call order service for cancellation", slog.String("error", err.Error()))
		return fmt.Errorf("call order service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return httpclient.ParseResponseError(resp, "order")
	}

	s.logger.InfoContext(ctx, "order cancelled",
		slog.String("order_id", orderID),
	)

	return nil
}
