package domain

import (
	"time"
)

// Review represents a product review submitted by a user.
type Review struct {
	ID        string    `json:"id"`
	ProductID string    `json:"product_id"`
	UserID    string    `json:"user_id"`
	Rating    int       `json:"rating"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ReviewSummary contains aggregate review statistics for a product.
type ReviewSummary struct {
	AverageRating float64 `json:"average_rating"`
	TotalCount    int     `json:"total_count"`
}
