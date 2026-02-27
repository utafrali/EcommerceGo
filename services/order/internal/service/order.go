package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/order/internal/domain"
	"github.com/utafrali/EcommerceGo/services/order/internal/event"
	"github.com/utafrali/EcommerceGo/services/order/internal/repository"
)

// OrderService implements the business logic for order operations.
type OrderService struct {
	repo     repository.OrderRepository
	producer *event.Producer
	logger   *slog.Logger
}

// NewOrderService creates a new order service.
func NewOrderService(repo repository.OrderRepository, producer *event.Producer, logger *slog.Logger) *OrderService {
	return &OrderService{
		repo:     repo,
		producer: producer,
		logger:   logger,
	}
}

// CreateOrderItemInput holds the parameters for an order line item.
type CreateOrderItemInput struct {
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id"`
	Name      string `json:"name"`
	SKU       string `json:"sku"`
	Price     int64  `json:"price"`
	Quantity  int    `json:"quantity"`
}

// CreateOrderInput holds the parameters for creating an order.
type CreateOrderInput struct {
	UserID          string
	Items           []CreateOrderItemInput
	DiscountAmount  int64
	ShippingAmount  int64
	Currency        string
	ShippingAddress *domain.Address
	BillingAddress  *domain.Address
	Notes           string
}

// CreateOrder creates a new order from the given input.
func (s *OrderService) CreateOrder(ctx context.Context, input CreateOrderInput) (*domain.Order, error) {
	if input.UserID == "" {
		return nil, apperrors.InvalidInput("user_id is required")
	}
	if len(input.Items) == 0 {
		return nil, apperrors.InvalidInput("order must contain at least one item")
	}
	if len(input.Currency) != 3 {
		return nil, apperrors.InvalidInput("currency must be a 3-letter ISO code")
	}

	now := time.Now().UTC()
	orderID := uuid.New().String()

	// Build order items and calculate subtotal.
	var subtotal int64
	items := make([]domain.OrderItem, len(input.Items))
	for i, itemInput := range input.Items {
		itemID := uuid.New().String()
		items[i] = domain.OrderItem{
			ID:        itemID,
			OrderID:   orderID,
			ProductID: itemInput.ProductID,
			VariantID: itemInput.VariantID,
			Name:      itemInput.Name,
			SKU:       itemInput.SKU,
			Price:     itemInput.Price,
			Quantity:  itemInput.Quantity,
		}
		subtotal += items[i].LineTotal()
	}

	totalAmount := subtotal - input.DiscountAmount + input.ShippingAmount
	if totalAmount < 0 {
		totalAmount = 0
	}

	order := &domain.Order{
		ID:              orderID,
		UserID:          input.UserID,
		Status:          domain.OrderStatusPending,
		Items:           items,
		SubtotalAmount:  subtotal,
		DiscountAmount:  input.DiscountAmount,
		ShippingAmount:  input.ShippingAmount,
		TotalAmount:     totalAmount,
		Currency:        strings.ToUpper(input.Currency),
		ShippingAddress: input.ShippingAddress,
		BillingAddress:  input.BillingAddress,
		Notes:           input.Notes,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	if err := s.producer.PublishOrderCreated(ctx, order); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish order.created event",
			slog.String("order_id", order.ID),
			slog.String("error", err.Error()),
		)
		// Do not fail the operation if event publishing fails.
	}

	s.logger.InfoContext(ctx, "order created",
		slog.String("order_id", order.ID),
		slog.String("user_id", order.UserID),
		slog.Int64("total_amount", order.TotalAmount),
	)

	return order, nil
}

// GetOrder retrieves an order by its ID.
func (s *OrderService) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get order by id: %w", err)
	}
	return order, nil
}

// ListOrders returns a filtered, paginated list of orders.
func (s *OrderService) ListOrders(ctx context.Context, filter repository.OrderFilter) ([]domain.Order, int, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.PerPage > 100 {
		filter.PerPage = 100
	}

	orders, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}

	return orders, total, nil
}

// UpdateOrderStatus transitions the order to a new status with validation.
func (s *OrderService) UpdateOrderStatus(ctx context.Context, id string, newStatus string, reason string) (*domain.Order, error) {
	if !domain.IsValidStatus(newStatus) {
		return nil, apperrors.InvalidInput(fmt.Sprintf("invalid status %q, must be one of: %s", newStatus, strings.Join(domain.ValidStatuses(), ", ")))
	}

	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get order for status update: %w", err)
	}

	if !order.CanTransitionTo(newStatus) {
		return nil, apperrors.InvalidInput(fmt.Sprintf("cannot transition from %q to %q", order.Status, newStatus))
	}

	oldStatus := order.Status

	if err := s.repo.UpdateStatus(ctx, id, newStatus, reason); err != nil {
		return nil, fmt.Errorf("update order status: %w", err)
	}

	if err := s.producer.PublishOrderStatusChanged(ctx, id, oldStatus, newStatus); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish order.status_changed event",
			slog.String("order_id", id),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "order status updated",
		slog.String("order_id", id),
		slog.String("old_status", oldStatus),
		slog.String("new_status", newStatus),
	)

	// Return updated order.
	order.Status = newStatus
	if reason != "" {
		order.CanceledReason = reason
	}

	return order, nil
}

// CancelOrder cancels an order with a reason, validating the transition.
func (s *OrderService) CancelOrder(ctx context.Context, id string, reason string) (*domain.Order, error) {
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get order for cancel: %w", err)
	}

	if !order.CanTransitionTo(domain.OrderStatusCanceled) {
		return nil, apperrors.InvalidInput(fmt.Sprintf("cannot cancel order in %q status", order.Status))
	}

	if err := s.repo.UpdateStatus(ctx, id, domain.OrderStatusCanceled, reason); err != nil {
		return nil, fmt.Errorf("cancel order: %w", err)
	}

	if err := s.producer.PublishOrderCanceled(ctx, id, reason); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish order.canceled event",
			slog.String("order_id", id),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "order canceled",
		slog.String("order_id", id),
		slog.String("reason", reason),
	)

	order.Status = domain.OrderStatusCanceled
	order.CanceledReason = reason

	return order, nil
}
