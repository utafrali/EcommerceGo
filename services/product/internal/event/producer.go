package event

import (
	"context"
	"fmt"
	"log/slog"

	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
)

// Kafka topic constants for product domain events.
const (
	TopicProductCreated = "ecommerce.product.created"
	TopicProductUpdated = "ecommerce.product.updated"
	TopicProductDeleted = "ecommerce.product.deleted"
)

// Aggregate type constant.
const AggregateTypeProduct = "product"

// Source identifier for events originating from the product service.
const SourceProductService = "product-service"

// ProductCreatedData is the payload for a product.created event.
type ProductCreatedData struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	Description string         `json:"description"`
	BrandID     *string        `json:"brand_id,omitempty"`
	CategoryID  *string        `json:"category_id,omitempty"`
	Status      string         `json:"status"`
	BasePrice   int64          `json:"base_price"`
	Currency    string         `json:"currency"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ProductUpdatedData is the payload for a product.updated event.
type ProductUpdatedData struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	Description string         `json:"description"`
	BrandID     *string        `json:"brand_id,omitempty"`
	CategoryID  *string        `json:"category_id,omitempty"`
	Status      string         `json:"status"`
	BasePrice   int64          `json:"base_price"`
	Currency    string         `json:"currency"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ProductDeletedData is the payload for a product.deleted event.
type ProductDeletedData struct {
	ID string `json:"id"`
}

// Producer publishes product domain events to Kafka.
type Producer struct {
	kafka  *pkgkafka.Producer
	logger *slog.Logger
}

// NewProducer creates a new event producer for the product service.
func NewProducer(kafka *pkgkafka.Producer, logger *slog.Logger) *Producer {
	return &Producer{
		kafka:  kafka,
		logger: logger,
	}
}

// PublishProductCreated publishes a product.created event.
func (p *Producer) PublishProductCreated(ctx context.Context, product *domain.Product) error {
	data := ProductCreatedData{
		ID:          product.ID,
		Name:        product.Name,
		Slug:        product.Slug,
		Description: product.Description,
		BrandID:     product.BrandID,
		CategoryID:  product.CategoryID,
		Status:      product.Status,
		BasePrice:   product.BasePrice,
		Currency:    product.Currency,
		Metadata:    product.Metadata,
	}

	event, err := pkgkafka.NewEvent(TopicProductCreated, product.ID, AggregateTypeProduct, SourceProductService, data)
	if err != nil {
		return fmt.Errorf("create product.created event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicProductCreated, event); err != nil {
		return fmt.Errorf("publish product.created event: %w", err)
	}

	p.logger.DebugContext(ctx, "published product.created event",
		slog.String("product_id", product.ID),
		slog.String("slug", product.Slug),
	)

	return nil
}

// PublishProductUpdated publishes a product.updated event.
func (p *Producer) PublishProductUpdated(ctx context.Context, product *domain.Product) error {
	data := ProductUpdatedData{
		ID:          product.ID,
		Name:        product.Name,
		Slug:        product.Slug,
		Description: product.Description,
		BrandID:     product.BrandID,
		CategoryID:  product.CategoryID,
		Status:      product.Status,
		BasePrice:   product.BasePrice,
		Currency:    product.Currency,
		Metadata:    product.Metadata,
	}

	event, err := pkgkafka.NewEvent(TopicProductUpdated, product.ID, AggregateTypeProduct, SourceProductService, data)
	if err != nil {
		return fmt.Errorf("create product.updated event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicProductUpdated, event); err != nil {
		return fmt.Errorf("publish product.updated event: %w", err)
	}

	p.logger.DebugContext(ctx, "published product.updated event",
		slog.String("product_id", product.ID),
		slog.String("slug", product.Slug),
	)

	return nil
}

// PublishProductDeleted publishes a product.deleted event.
func (p *Producer) PublishProductDeleted(ctx context.Context, id string) error {
	data := ProductDeletedData{ID: id}

	event, err := pkgkafka.NewEvent(TopicProductDeleted, id, AggregateTypeProduct, SourceProductService, data)
	if err != nil {
		return fmt.Errorf("create product.deleted event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicProductDeleted, event); err != nil {
		return fmt.Errorf("publish product.deleted event: %w", err)
	}

	p.logger.DebugContext(ctx, "published product.deleted event",
		slog.String("product_id", id),
	)

	return nil
}
