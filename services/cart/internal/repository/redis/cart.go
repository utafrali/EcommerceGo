package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/cart/internal/domain"
)

const keyPrefix = "cart:"

// CartRepository implements repository.CartRepository using Redis.
type CartRepository struct {
	client *redis.Client
	ttl    time.Duration
}

// NewCartRepository creates a new Redis-backed cart repository.
func NewCartRepository(client *redis.Client, ttl time.Duration) *CartRepository {
	return &CartRepository{
		client: client,
		ttl:    ttl,
	}
}

// Get retrieves a cart by user ID from Redis.
func (r *CartRepository) Get(ctx context.Context, userID string) (*domain.Cart, error) {
	key := keyPrefix + userID

	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, apperrors.NotFound("cart", userID)
		}
		return nil, fmt.Errorf("redis get cart: %w", err)
	}

	var cart domain.Cart
	if err := json.Unmarshal(data, &cart); err != nil {
		return nil, fmt.Errorf("unmarshal cart: %w", err)
	}

	return &cart, nil
}

// Save persists a cart to Redis with the configured TTL.
func (r *CartRepository) Save(ctx context.Context, cart *domain.Cart) error {
	key := keyPrefix + cart.UserID

	data, err := json.Marshal(cart)
	if err != nil {
		return fmt.Errorf("marshal cart: %w", err)
	}

	if err := r.client.Set(ctx, key, data, r.ttl).Err(); err != nil {
		return fmt.Errorf("redis set cart: %w", err)
	}

	return nil
}

// Delete removes a cart from Redis by user ID.
func (r *CartRepository) Delete(ctx context.Context, userID string) error {
	key := keyPrefix + userID

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis del cart: %w", err)
	}

	return nil
}
