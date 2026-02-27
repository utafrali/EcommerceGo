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

// SaveIfVersion atomically persists the cart only if the stored version matches
// the expected version. Uses Redis WATCH/MULTI/EXEC for optimistic locking.
// Returns (true, nil) on success, (false, nil) on version mismatch.
func (r *CartRepository) SaveIfVersion(ctx context.Context, cart *domain.Cart, expectedVersion int) (bool, error) {
	key := keyPrefix + cart.UserID

	// Use a Redis transaction with WATCH to implement optimistic locking.
	// The WATCH ensures the key hasn't been modified between our read and write.
	txf := func(tx *redis.Tx) error {
		// Read the current value inside the transaction.
		data, err := tx.Get(ctx, key).Bytes()
		if err != nil && err != redis.Nil {
			return fmt.Errorf("redis get cart in tx: %w", err)
		}

		// If the key exists, verify the version matches.
		if err != redis.Nil {
			var stored domain.Cart
			if err := json.Unmarshal(data, &stored); err != nil {
				return fmt.Errorf("unmarshal cart in tx: %w", err)
			}
			if stored.Version != expectedVersion {
				// Version mismatch: someone else modified the cart.
				return apperrors.ErrConflict
			}
		} else if expectedVersion != 0 {
			// Key does not exist but caller expected a specific version.
			return apperrors.ErrConflict
		}

		// Increment version and save.
		cart.Version = expectedVersion + 1
		newData, err := json.Marshal(cart)
		if err != nil {
			return fmt.Errorf("marshal cart in tx: %w", err)
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, newData, r.ttl)
			return nil
		})
		return err
	}

	err := r.client.Watch(ctx, txf, key)
	if err != nil {
		if err == redis.TxFailedErr {
			// WATCH detected a concurrent modification of the key.
			return false, nil
		}
		if err == apperrors.ErrConflict {
			// Version mismatch detected inside the transaction.
			return false, nil
		}
		return false, fmt.Errorf("redis watch tx: %w", err)
	}

	return true, nil
}

// Delete removes a cart from Redis by user ID.
func (r *CartRepository) Delete(ctx context.Context, userID string) error {
	key := keyPrefix + userID

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis del cart: %w", err)
	}

	return nil
}
