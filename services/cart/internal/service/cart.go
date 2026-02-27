package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/cart/internal/domain"
	"github.com/utafrali/EcommerceGo/services/cart/internal/event"
	"github.com/utafrali/EcommerceGo/services/cart/internal/repository"
)

// Cart operation upper-bound limits to prevent abuse.
const (
	// MaxQuantityPerItem is the maximum quantity allowed for a single cart item.
	MaxQuantityPerItem = 100
	// MaxItemsPerCart is the maximum number of distinct items allowed in a cart.
	MaxItemsPerCart = 50
	// MaxPriceCents is the maximum price in cents (100,000.00) allowed per item.
	MaxPriceCents = 100_000_00
)

// AddItemInput holds the parameters for adding an item to the cart.
type AddItemInput struct {
	ProductID string `json:"product_id" validate:"required"`
	VariantID string `json:"variant_id" validate:"required"`
	Name      string `json:"name" validate:"required"`
	SKU       string `json:"sku" validate:"required"`
	Price     int64  `json:"price" validate:"required,gte=0"`
	Quantity  int    `json:"quantity" validate:"required,gte=1"`
	ImageURL  string `json:"image_url"`
}

// UpdateQuantityInput holds the parameters for updating an item quantity.
type UpdateQuantityInput struct {
	Quantity int `json:"quantity" validate:"gte=0"`
}

// CartService implements the business logic for cart operations.
type CartService struct {
	repo     repository.CartRepository
	producer *event.Producer
	logger   *slog.Logger
	cartTTL  time.Duration
}

// NewCartService creates a new cart service.
func NewCartService(repo repository.CartRepository, producer *event.Producer, logger *slog.Logger, cartTTL time.Duration) *CartService {
	return &CartService{
		repo:     repo,
		producer: producer,
		logger:   logger,
		cartTTL:  cartTTL,
	}
}

// GetCart retrieves the cart for a user. If no cart exists, returns an empty cart.
func (s *CartService) GetCart(ctx context.Context, userID string) (*domain.Cart, error) {
	if userID == "" {
		return nil, apperrors.InvalidInput("user id is required")
	}

	cart, err := s.repo.Get(ctx, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return s.newEmptyCart(userID), nil
		}
		return nil, fmt.Errorf("get cart: %w", err)
	}

	return cart, nil
}

// AddItem adds an item to the user's cart. If the same product+variant exists, it merges by increasing quantity.
// Uses optimistic locking to prevent race conditions on concurrent cart modifications.
func (s *CartService) AddItem(ctx context.Context, userID string, input AddItemInput) (*domain.Cart, error) {
	if userID == "" {
		return nil, apperrors.InvalidInput("user id is required")
	}
	if input.ProductID == "" {
		return nil, apperrors.InvalidInput("product id is required")
	}
	if input.VariantID == "" {
		return nil, apperrors.InvalidInput("variant id is required")
	}
	if input.Quantity <= 0 {
		return nil, apperrors.InvalidInput("quantity must be greater than 0")
	}
	if input.Price < 0 {
		return nil, apperrors.InvalidInput("price must not be negative")
	}
	if input.Quantity > MaxQuantityPerItem {
		return nil, apperrors.InvalidInput(fmt.Sprintf("quantity must not exceed %d", MaxQuantityPerItem))
	}
	if input.Price > MaxPriceCents {
		return nil, apperrors.InvalidInput(fmt.Sprintf("price must not exceed %d cents", MaxPriceCents))
	}

	cart, err := s.getOrCreateCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	expectedVersion := cart.Version

	// Check if the item already exists (same product+variant). If so, merge.
	found := false
	for i := range cart.Items {
		if cart.Items[i].ProductID == input.ProductID && cart.Items[i].VariantID == input.VariantID {
			newQty := cart.Items[i].Quantity + input.Quantity
			if newQty > MaxQuantityPerItem {
				return nil, apperrors.InvalidInput(fmt.Sprintf("combined quantity must not exceed %d", MaxQuantityPerItem))
			}
			cart.Items[i].Quantity = newQty
			// Update price and other fields in case they changed.
			cart.Items[i].Price = input.Price
			cart.Items[i].Name = input.Name
			cart.Items[i].SKU = input.SKU
			cart.Items[i].ImageURL = input.ImageURL
			found = true
			break
		}
	}

	if !found {
		if len(cart.Items) >= MaxItemsPerCart {
			return nil, apperrors.InvalidInput(fmt.Sprintf("cart must not contain more than %d items", MaxItemsPerCart))
		}
		cart.Items = append(cart.Items, domain.CartItem{
			ProductID: input.ProductID,
			VariantID: input.VariantID,
			Name:      input.Name,
			SKU:       input.SKU,
			Price:     input.Price,
			Quantity:  input.Quantity,
			ImageURL:  input.ImageURL,
		})
	}

	now := time.Now().UTC()
	cart.UpdatedAt = now
	cart.ExpiresAt = now.Add(s.cartTTL)

	ok, err := s.repo.SaveIfVersion(ctx, cart, expectedVersion)
	if err != nil {
		return nil, fmt.Errorf("save cart: %w", err)
	}
	if !ok {
		return nil, apperrors.Conflict("cart was modified concurrently, please retry")
	}

	if err := s.producer.PublishCartUpdated(ctx, cart); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish cart.updated event",
			slog.String("user_id", userID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "item added to cart",
		slog.String("user_id", userID),
		slog.String("product_id", input.ProductID),
		slog.String("variant_id", input.VariantID),
		slog.Int("quantity", input.Quantity),
	)

	return cart, nil
}

// UpdateItemQuantity updates the quantity of an item in the cart. If quantity is 0, the item is removed.
// Uses optimistic locking to prevent race conditions on concurrent cart modifications.
func (s *CartService) UpdateItemQuantity(ctx context.Context, userID, productID, variantID string, quantity int) (*domain.Cart, error) {
	if userID == "" {
		return nil, apperrors.InvalidInput("user id is required")
	}
	if productID == "" {
		return nil, apperrors.InvalidInput("product id is required")
	}
	if variantID == "" {
		return nil, apperrors.InvalidInput("variant id is required")
	}
	if quantity < 0 {
		return nil, apperrors.InvalidInput("quantity must not be negative")
	}
	if quantity > MaxQuantityPerItem {
		return nil, apperrors.InvalidInput(fmt.Sprintf("quantity must not exceed %d", MaxQuantityPerItem))
	}

	cart, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get cart for update: %w", err)
	}

	expectedVersion := cart.Version

	found := false
	for i := range cart.Items {
		if cart.Items[i].ProductID == productID && cart.Items[i].VariantID == variantID {
			found = true
			if quantity == 0 {
				// Remove the item.
				cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
			} else {
				cart.Items[i].Quantity = quantity
			}
			break
		}
	}

	if !found {
		return nil, apperrors.NotFound("cart item", productID+"/"+variantID)
	}

	now := time.Now().UTC()
	cart.UpdatedAt = now
	cart.ExpiresAt = now.Add(s.cartTTL)

	ok, err := s.repo.SaveIfVersion(ctx, cart, expectedVersion)
	if err != nil {
		return nil, fmt.Errorf("save cart: %w", err)
	}
	if !ok {
		return nil, apperrors.Conflict("cart was modified concurrently, please retry")
	}

	if err := s.producer.PublishCartUpdated(ctx, cart); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish cart.updated event",
			slog.String("user_id", userID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "cart item quantity updated",
		slog.String("user_id", userID),
		slog.String("product_id", productID),
		slog.String("variant_id", variantID),
		slog.Int("quantity", quantity),
	)

	return cart, nil
}

// RemoveItem removes a specific item from the cart.
// Uses optimistic locking to prevent race conditions on concurrent cart modifications.
func (s *CartService) RemoveItem(ctx context.Context, userID, productID, variantID string) (*domain.Cart, error) {
	if userID == "" {
		return nil, apperrors.InvalidInput("user id is required")
	}
	if productID == "" {
		return nil, apperrors.InvalidInput("product id is required")
	}
	if variantID == "" {
		return nil, apperrors.InvalidInput("variant id is required")
	}

	cart, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get cart for remove: %w", err)
	}

	expectedVersion := cart.Version

	found := false
	for i := range cart.Items {
		if cart.Items[i].ProductID == productID && cart.Items[i].VariantID == variantID {
			cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return nil, apperrors.NotFound("cart item", productID+"/"+variantID)
	}

	now := time.Now().UTC()
	cart.UpdatedAt = now
	cart.ExpiresAt = now.Add(s.cartTTL)

	ok, err := s.repo.SaveIfVersion(ctx, cart, expectedVersion)
	if err != nil {
		return nil, fmt.Errorf("save cart: %w", err)
	}
	if !ok {
		return nil, apperrors.Conflict("cart was modified concurrently, please retry")
	}

	if err := s.producer.PublishCartUpdated(ctx, cart); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish cart.updated event",
			slog.String("user_id", userID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "item removed from cart",
		slog.String("user_id", userID),
		slog.String("product_id", productID),
		slog.String("variant_id", variantID),
	)

	return cart, nil
}

// ClearCart removes all items from the user's cart.
func (s *CartService) ClearCart(ctx context.Context, userID string) error {
	if userID == "" {
		return apperrors.InvalidInput("user id is required")
	}

	if err := s.repo.Delete(ctx, userID); err != nil {
		return fmt.Errorf("delete cart: %w", err)
	}

	if err := s.producer.PublishCartCleared(ctx, userID); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish cart.cleared event",
			slog.String("user_id", userID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "cart cleared",
		slog.String("user_id", userID),
	)

	return nil
}

// getOrCreateCart retrieves the cart for a user, creating an empty one if it does not exist.
func (s *CartService) getOrCreateCart(ctx context.Context, userID string) (*domain.Cart, error) {
	cart, err := s.repo.Get(ctx, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return s.newEmptyCart(userID), nil
		}
		return nil, fmt.Errorf("get cart: %w", err)
	}
	return cart, nil
}

// newEmptyCart creates a new empty cart for the given user.
func (s *CartService) newEmptyCart(userID string) *domain.Cart {
	now := time.Now().UTC()
	return &domain.Cart{
		ID:        uuid.New().String(),
		UserID:    userID,
		Items:     []domain.CartItem{},
		Currency:  "USD",
		Version:   0,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(s.cartTTL),
	}
}
