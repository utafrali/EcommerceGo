package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/search/internal/service"
)

// Kafka topic constants for product domain events consumed by the search service.
const (
	TopicProductCreated = "ecommerce.product.created"
	TopicProductUpdated = "ecommerce.product.updated"
	TopicProductDeleted = "ecommerce.product.deleted"
)

// ProductEventData represents the payload from product domain events.
type ProductEventData struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	BrandID     *string `json:"brand_id,omitempty"`
	CategoryID  *string `json:"category_id,omitempty"`
	Status      string  `json:"status"`
	BasePrice   int64   `json:"base_price"`
	Currency    string  `json:"currency"`
}

// ProductDeletedData represents the payload from a product.deleted event.
type ProductDeletedData struct {
	ID string `json:"id"`
}

// Consumer handles Kafka events related to product changes for search indexing.
type Consumer struct {
	searchService *service.SearchService
	logger        *slog.Logger
}

// NewConsumer creates a new event consumer for the search service.
func NewConsumer(searchService *service.SearchService, logger *slog.Logger) *Consumer {
	return &Consumer{
		searchService: searchService,
		logger:        logger,
	}
}

// Handle processes a Kafka event based on its type.
func (c *Consumer) Handle(ctx context.Context, event *pkgkafka.Event) error {
	switch event.EventType {
	case TopicProductCreated:
		return c.handleProductCreated(ctx, event)
	case TopicProductUpdated:
		return c.handleProductUpdated(ctx, event)
	case TopicProductDeleted:
		return c.handleProductDeleted(ctx, event)
	default:
		c.logger.WarnContext(ctx, "unknown event type received",
			slog.String("event_type", event.EventType),
			slog.String("event_id", event.EventID),
		)
		return nil
	}
}

// handleProductCreated indexes a newly created product.
func (c *Consumer) handleProductCreated(ctx context.Context, event *pkgkafka.Event) error {
	var data ProductEventData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal product.created data: %w", err)
	}

	input := &service.IndexProductInput{
		ID:          data.ID,
		Name:        data.Name,
		Slug:        data.Slug,
		Description: data.Description,
		BasePrice:   data.BasePrice,
		Currency:    data.Currency,
		Status:      data.Status,
	}

	if data.CategoryID != nil {
		input.CategoryID = *data.CategoryID
	}
	if data.BrandID != nil {
		input.BrandID = *data.BrandID
	}

	if err := c.searchService.IndexProduct(ctx, input); err != nil {
		return fmt.Errorf("index product from created event: %w", err)
	}

	c.logger.InfoContext(ctx, "indexed product from created event",
		slog.String("product_id", data.ID),
	)

	return nil
}

// handleProductUpdated re-indexes an updated product.
func (c *Consumer) handleProductUpdated(ctx context.Context, event *pkgkafka.Event) error {
	var data ProductEventData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal product.updated data: %w", err)
	}

	input := &service.IndexProductInput{
		ID:          data.ID,
		Name:        data.Name,
		Slug:        data.Slug,
		Description: data.Description,
		BasePrice:   data.BasePrice,
		Currency:    data.Currency,
		Status:      data.Status,
	}

	if data.CategoryID != nil {
		input.CategoryID = *data.CategoryID
	}
	if data.BrandID != nil {
		input.BrandID = *data.BrandID
	}

	if err := c.searchService.IndexProduct(ctx, input); err != nil {
		return fmt.Errorf("index product from updated event: %w", err)
	}

	c.logger.InfoContext(ctx, "re-indexed product from updated event",
		slog.String("product_id", data.ID),
	)

	return nil
}

// handleProductDeleted removes a deleted product from the index.
func (c *Consumer) handleProductDeleted(ctx context.Context, event *pkgkafka.Event) error {
	var data ProductDeletedData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal product.deleted data: %w", err)
	}

	if err := c.searchService.DeleteProduct(ctx, data.ID); err != nil {
		return fmt.Errorf("delete product from deleted event: %w", err)
	}

	c.logger.InfoContext(ctx, "deleted product from deleted event",
		slog.String("product_id", data.ID),
	)

	return nil
}
