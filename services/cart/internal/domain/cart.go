package domain

import "time"

// Cart represents a shopping cart.
type Cart struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Items     []CartItem `json:"items"`
	Currency  string     `json:"currency"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ExpiresAt time.Time  `json:"expires_at"`
}

// CartItem represents a single item in the cart.
type CartItem struct {
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id"`
	Name      string `json:"name"`
	SKU       string `json:"sku"`
	Price     int64  `json:"price"`
	Quantity  int    `json:"quantity"`
	ImageURL  string `json:"image_url,omitempty"`
}

// TotalAmount calculates the total price of all items in the cart (in cents).
func (c *Cart) TotalAmount() int64 {
	var total int64
	for _, item := range c.Items {
		total += item.Price * int64(item.Quantity)
	}
	return total
}

// ItemCount returns the total number of items in the cart.
func (c *Cart) ItemCount() int {
	var count int
	for _, item := range c.Items {
		count += item.Quantity
	}
	return count
}
