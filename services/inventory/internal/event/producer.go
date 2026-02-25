package event

import (
	"context"
	"fmt"
	"log/slog"

	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/inventory/internal/domain"
)

// Kafka topic constants for inventory domain events.
const (
	TopicInventoryUpdated  = "ecommerce.inventory.updated"
	TopicInventoryReserved = "ecommerce.inventory.reserved"
	TopicInventoryReleased = "ecommerce.inventory.released"
	TopicInventoryLowStock = "ecommerce.inventory.low_stock"
)

// Aggregate type constant.
const AggregateTypeInventory = "inventory"

// Source identifier for events originating from the inventory service.
const SourceInventoryService = "inventory-service"

// InventoryUpdatedData is the payload for an inventory.updated event.
type InventoryUpdatedData struct {
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id"`
	Quantity  int    `json:"quantity"`
	Reserved  int    `json:"reserved"`
	Available int    `json:"available"`
}

// InventoryReservedData is the payload for an inventory.reserved event.
type InventoryReservedData struct {
	ReservationID string `json:"reservation_id"`
	CheckoutID    string `json:"checkout_id"`
	ProductID     string `json:"product_id"`
	VariantID     string `json:"variant_id"`
	Quantity      int    `json:"quantity"`
}

// InventoryReleasedData is the payload for an inventory.released event.
type InventoryReleasedData struct {
	ReservationID string `json:"reservation_id"`
	ProductID     string `json:"product_id"`
	VariantID     string `json:"variant_id"`
	Quantity      int    `json:"quantity"`
}

// InventoryLowStockData is the payload for an inventory.low_stock event.
type InventoryLowStockData struct {
	ProductID         string `json:"product_id"`
	VariantID         string `json:"variant_id"`
	Available         int    `json:"available"`
	LowStockThreshold int    `json:"low_stock_threshold"`
}

// Producer publishes inventory domain events to Kafka.
type Producer struct {
	kafka  *pkgkafka.Producer
	logger *slog.Logger
}

// NewProducer creates a new event producer for the inventory service.
func NewProducer(kafka *pkgkafka.Producer, logger *slog.Logger) *Producer {
	return &Producer{
		kafka:  kafka,
		logger: logger,
	}
}

// PublishInventoryUpdated publishes an inventory.updated event.
func (p *Producer) PublishInventoryUpdated(ctx context.Context, stock *domain.Stock) error {
	data := InventoryUpdatedData{
		ProductID: stock.ProductID,
		VariantID: stock.VariantID,
		Quantity:  stock.Quantity,
		Reserved:  stock.Reserved,
		Available: stock.Available(),
	}

	event, err := pkgkafka.NewEvent(TopicInventoryUpdated, stock.ProductID, AggregateTypeInventory, SourceInventoryService, data)
	if err != nil {
		return fmt.Errorf("create inventory.updated event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicInventoryUpdated, event); err != nil {
		return fmt.Errorf("publish inventory.updated event: %w", err)
	}

	p.logger.DebugContext(ctx, "published inventory.updated event",
		slog.String("product_id", stock.ProductID),
		slog.String("variant_id", stock.VariantID),
	)

	return nil
}

// PublishInventoryReserved publishes an inventory.reserved event.
func (p *Producer) PublishInventoryReserved(ctx context.Context, reservation *domain.StockReservation) error {
	data := InventoryReservedData{
		ReservationID: reservation.ID,
		CheckoutID:    reservation.CheckoutID,
		ProductID:     reservation.ProductID,
		VariantID:     reservation.VariantID,
		Quantity:      reservation.Quantity,
	}

	event, err := pkgkafka.NewEvent(TopicInventoryReserved, reservation.ProductID, AggregateTypeInventory, SourceInventoryService, data)
	if err != nil {
		return fmt.Errorf("create inventory.reserved event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicInventoryReserved, event); err != nil {
		return fmt.Errorf("publish inventory.reserved event: %w", err)
	}

	p.logger.DebugContext(ctx, "published inventory.reserved event",
		slog.String("reservation_id", reservation.ID),
		slog.String("checkout_id", reservation.CheckoutID),
	)

	return nil
}

// PublishInventoryReleased publishes an inventory.released event.
func (p *Producer) PublishInventoryReleased(ctx context.Context, reservation *domain.StockReservation) error {
	data := InventoryReleasedData{
		ReservationID: reservation.ID,
		ProductID:     reservation.ProductID,
		VariantID:     reservation.VariantID,
		Quantity:      reservation.Quantity,
	}

	event, err := pkgkafka.NewEvent(TopicInventoryReleased, reservation.ProductID, AggregateTypeInventory, SourceInventoryService, data)
	if err != nil {
		return fmt.Errorf("create inventory.released event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicInventoryReleased, event); err != nil {
		return fmt.Errorf("publish inventory.released event: %w", err)
	}

	p.logger.DebugContext(ctx, "published inventory.released event",
		slog.String("reservation_id", reservation.ID),
	)

	return nil
}

// PublishInventoryLowStock publishes an inventory.low_stock event.
func (p *Producer) PublishInventoryLowStock(ctx context.Context, stock *domain.Stock) error {
	data := InventoryLowStockData{
		ProductID:         stock.ProductID,
		VariantID:         stock.VariantID,
		Available:         stock.Available(),
		LowStockThreshold: stock.LowStockThreshold,
	}

	event, err := pkgkafka.NewEvent(TopicInventoryLowStock, stock.ProductID, AggregateTypeInventory, SourceInventoryService, data)
	if err != nil {
		return fmt.Errorf("create inventory.low_stock event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicInventoryLowStock, event); err != nil {
		return fmt.Errorf("publish inventory.low_stock event: %w", err)
	}

	p.logger.DebugContext(ctx, "published inventory.low_stock event",
		slog.String("product_id", stock.ProductID),
		slog.String("variant_id", stock.VariantID),
		slog.Int("available", stock.Available()),
	)

	return nil
}
