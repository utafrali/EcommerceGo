package event

import (
	"context"
	"fmt"
	"log/slog"

	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/order/internal/domain"
)

// Kafka topic constants for order domain events.
const (
	TopicOrderCreated       = "ecommerce.order.created"
	TopicOrderStatusChanged = "ecommerce.order.status_changed"
	TopicOrderCanceled      = "ecommerce.order.canceled"
)

// Aggregate type constant.
const AggregateTypeOrder = "order"

// Source identifier for events originating from the order service.
const SourceOrderService = "order-service"

// OrderCreatedData is the payload for an order.created event (full order snapshot).
type OrderCreatedData struct {
	ID              string             `json:"id"`
	UserID          string             `json:"user_id"`
	Status          string             `json:"status"`
	Items           []OrderItemData    `json:"items"`
	SubtotalAmount  int64              `json:"subtotal_amount"`
	DiscountAmount  int64              `json:"discount_amount"`
	ShippingAmount  int64              `json:"shipping_amount"`
	TotalAmount     int64              `json:"total_amount"`
	Currency        string             `json:"currency"`
	ShippingAddress *domain.Address    `json:"shipping_address,omitempty"`
	BillingAddress  *domain.Address    `json:"billing_address,omitempty"`
	Notes           string             `json:"notes,omitempty"`
}

// OrderItemData is the event payload for an order item.
type OrderItemData struct {
	ID        string `json:"id"`
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id"`
	Name      string `json:"name"`
	SKU       string `json:"sku"`
	Price     int64  `json:"price"`
	Quantity  int    `json:"quantity"`
}

// OrderStatusChangedData is the payload for an order.status_changed event.
type OrderStatusChangedData struct {
	OrderID   string `json:"order_id"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
}

// OrderCanceledData is the payload for an order.canceled event.
type OrderCanceledData struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

// Producer publishes order domain events to Kafka.
type Producer struct {
	kafka  *pkgkafka.Producer
	logger *slog.Logger
}

// NewProducer creates a new event producer for the order service.
func NewProducer(kafka *pkgkafka.Producer, logger *slog.Logger) *Producer {
	return &Producer{
		kafka:  kafka,
		logger: logger,
	}
}

// PublishOrderCreated publishes an order.created event with the full order snapshot.
func (p *Producer) PublishOrderCreated(ctx context.Context, order *domain.Order) error {
	items := make([]OrderItemData, len(order.Items))
	for i, item := range order.Items {
		items[i] = OrderItemData{
			ID:        item.ID,
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			Name:      item.Name,
			SKU:       item.SKU,
			Price:     item.Price,
			Quantity:  item.Quantity,
		}
	}

	data := OrderCreatedData{
		ID:              order.ID,
		UserID:          order.UserID,
		Status:          order.Status,
		Items:           items,
		SubtotalAmount:  order.SubtotalAmount,
		DiscountAmount:  order.DiscountAmount,
		ShippingAmount:  order.ShippingAmount,
		TotalAmount:     order.TotalAmount,
		Currency:        order.Currency,
		ShippingAddress: order.ShippingAddress,
		BillingAddress:  order.BillingAddress,
		Notes:           order.Notes,
	}

	event, err := pkgkafka.NewEvent(TopicOrderCreated, order.ID, AggregateTypeOrder, SourceOrderService, data)
	if err != nil {
		return fmt.Errorf("create order.created event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicOrderCreated, event); err != nil {
		return fmt.Errorf("publish order.created event: %w", err)
	}

	p.logger.DebugContext(ctx, "published order.created event",
		slog.String("order_id", order.ID),
		slog.String("user_id", order.UserID),
	)

	return nil
}

// PublishOrderStatusChanged publishes an order.status_changed event.
func (p *Producer) PublishOrderStatusChanged(ctx context.Context, orderID, oldStatus, newStatus string) error {
	data := OrderStatusChangedData{
		OrderID:   orderID,
		OldStatus: oldStatus,
		NewStatus: newStatus,
	}

	event, err := pkgkafka.NewEvent(TopicOrderStatusChanged, orderID, AggregateTypeOrder, SourceOrderService, data)
	if err != nil {
		return fmt.Errorf("create order.status_changed event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicOrderStatusChanged, event); err != nil {
		return fmt.Errorf("publish order.status_changed event: %w", err)
	}

	p.logger.DebugContext(ctx, "published order.status_changed event",
		slog.String("order_id", orderID),
		slog.String("old_status", oldStatus),
		slog.String("new_status", newStatus),
	)

	return nil
}

// PublishOrderCanceled publishes an order.canceled event.
func (p *Producer) PublishOrderCanceled(ctx context.Context, orderID, reason string) error {
	data := OrderCanceledData{
		OrderID: orderID,
		Reason:  reason,
	}

	event, err := pkgkafka.NewEvent(TopicOrderCanceled, orderID, AggregateTypeOrder, SourceOrderService, data)
	if err != nil {
		return fmt.Errorf("create order.canceled event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicOrderCanceled, event); err != nil {
		return fmt.Errorf("publish order.canceled event: %w", err)
	}

	p.logger.DebugContext(ctx, "published order.canceled event",
		slog.String("order_id", orderID),
		slog.String("reason", reason),
	)

	return nil
}
