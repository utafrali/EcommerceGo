package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
)

// Kafka topics consumed by the inventory service.
const (
	TopicOrderConfirmed = "ecommerce.order.confirmed"
	TopicOrderCanceled  = "ecommerce.order.canceled"
)

// InventoryService defines the interface required by the event consumer.
type InventoryService interface {
	ConfirmReservationByCheckoutID(ctx context.Context, checkoutID string) error
	ReleaseReservationByCheckoutID(ctx context.Context, checkoutID string) error
}

// OrderConfirmedData is the expected payload of an order.confirmed event.
type OrderConfirmedData struct {
	OrderID    string `json:"order_id"`
	CheckoutID string `json:"checkout_id"`
}

// OrderCanceledData is the expected payload of an order.canceled event.
type OrderCanceledData struct {
	OrderID    string `json:"order_id"`
	CheckoutID string `json:"checkout_id"`
}

// Consumer processes incoming Kafka events for the inventory service.
type Consumer struct {
	logger  *slog.Logger
	service InventoryService
}

// NewConsumer creates a new event consumer for the inventory service.
func NewConsumer(service InventoryService, logger *slog.Logger) *Consumer {
	return &Consumer{
		service: service,
		logger:  logger,
	}
}

// HandleOrderConfirmed processes order.confirmed events by confirming the stock reservation.
func (c *Consumer) HandleOrderConfirmed(ctx context.Context, event *pkgkafka.Event) error {
	var data OrderConfirmedData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal order.confirmed data: %w", err)
	}

	c.logger.InfoContext(ctx, "processing order.confirmed event",
		slog.String("order_id", data.OrderID),
		slog.String("checkout_id", data.CheckoutID),
	)

	if err := c.service.ConfirmReservationByCheckoutID(ctx, data.CheckoutID); err != nil {
		return fmt.Errorf("confirm reservation for checkout %s: %w", data.CheckoutID, err)
	}

	c.logger.InfoContext(ctx, "stock confirmed for order",
		slog.String("order_id", data.OrderID),
		slog.String("checkout_id", data.CheckoutID),
	)

	return nil
}

// HandleOrderCanceled processes order.canceled events by releasing the stock reservation.
func (c *Consumer) HandleOrderCanceled(ctx context.Context, event *pkgkafka.Event) error {
	var data OrderCanceledData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal order.canceled data: %w", err)
	}

	c.logger.InfoContext(ctx, "processing order.canceled event",
		slog.String("order_id", data.OrderID),
		slog.String("checkout_id", data.CheckoutID),
	)

	if err := c.service.ReleaseReservationByCheckoutID(ctx, data.CheckoutID); err != nil {
		return fmt.Errorf("release reservation for checkout %s: %w", data.CheckoutID, err)
	}

	c.logger.InfoContext(ctx, "stock released for canceled order",
		slog.String("order_id", data.OrderID),
		slog.String("checkout_id", data.CheckoutID),
	)

	return nil
}
