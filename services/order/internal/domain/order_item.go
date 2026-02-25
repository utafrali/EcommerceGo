package domain

// OrderItem represents a line item in an order.
type OrderItem struct {
	ID        string `json:"id"`
	OrderID   string `json:"order_id"`
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id"`
	Name      string `json:"name"`
	SKU       string `json:"sku"`
	Price     int64  `json:"price"`
	Quantity  int    `json:"quantity"`
}

// LineTotal returns the total price for this line item.
func (i *OrderItem) LineTotal() int64 {
	return i.Price * int64(i.Quantity)
}
