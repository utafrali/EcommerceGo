package event

import (
	"context"
	"fmt"
	"log/slog"

	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/cart/internal/domain"
)

// Kafka topic constants for cart domain events.
const (
	TopicCartUpdated = "ecommerce.cart.updated"
	TopicCartCleared = "ecommerce.cart.cleared"
)

// Aggregate type constant.
const AggregateTypeCart = "cart"

// Source identifier for events originating from the cart service.
const SourceCartService = "cart-service"

// CartUpdatedData is the payload for a cart.updated event.
type CartUpdatedData struct {
	UserID      string            `json:"user_id"`
	Items       []CartItemData    `json:"items"`
	ItemCount   int               `json:"item_count"`
	TotalAmount int64             `json:"total_amount"`
	Currency    string            `json:"currency"`
}

// CartItemData is the item payload within cart events.
type CartItemData struct {
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id"`
	Name      string `json:"name"`
	SKU       string `json:"sku"`
	Price     int64  `json:"price"`
	Quantity  int    `json:"quantity"`
}

// CartClearedData is the payload for a cart.cleared event.
type CartClearedData struct {
	UserID string `json:"user_id"`
}

// Producer publishes cart domain events to Kafka.
type Producer struct {
	kafka  *pkgkafka.Producer
	logger *slog.Logger
}

// NewProducer creates a new event producer for the cart service.
func NewProducer(kafka *pkgkafka.Producer, logger *slog.Logger) *Producer {
	return &Producer{
		kafka:  kafka,
		logger: logger,
	}
}

// PublishCartUpdated publishes a cart.updated event.
func (p *Producer) PublishCartUpdated(ctx context.Context, cart *domain.Cart) error {
	items := make([]CartItemData, len(cart.Items))
	for i, item := range cart.Items {
		items[i] = CartItemData{
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			Name:      item.Name,
			SKU:       item.SKU,
			Price:     item.Price,
			Quantity:  item.Quantity,
		}
	}

	data := CartUpdatedData{
		UserID:      cart.UserID,
		Items:       items,
		ItemCount:   cart.ItemCount(),
		TotalAmount: cart.TotalAmount(),
		Currency:    cart.Currency,
	}

	event, err := pkgkafka.NewEvent(TopicCartUpdated, cart.UserID, AggregateTypeCart, SourceCartService, data)
	if err != nil {
		return fmt.Errorf("create cart.updated event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicCartUpdated, event); err != nil {
		return fmt.Errorf("publish cart.updated event: %w", err)
	}

	p.logger.DebugContext(ctx, "published cart.updated event",
		slog.String("user_id", cart.UserID),
		slog.Int("item_count", cart.ItemCount()),
	)

	return nil
}

// PublishCartCleared publishes a cart.cleared event.
func (p *Producer) PublishCartCleared(ctx context.Context, userID string) error {
	data := CartClearedData{UserID: userID}

	event, err := pkgkafka.NewEvent(TopicCartCleared, userID, AggregateTypeCart, SourceCartService, data)
	if err != nil {
		return fmt.Errorf("create cart.cleared event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicCartCleared, event); err != nil {
		return fmt.Errorf("publish cart.cleared event: %w", err)
	}

	p.logger.DebugContext(ctx, "published cart.cleared event",
		slog.String("user_id", userID),
	)

	return nil
}
