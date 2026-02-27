package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/utafrali/EcommerceGo/pkg/health"
	"github.com/utafrali/EcommerceGo/pkg/middleware"
	"github.com/utafrali/EcommerceGo/services/cart/internal/service"
)

// NewRouter creates a chi router with all cart service routes registered.
func NewRouter(
	cartService *service.CartService,
	healthHandler *health.Handler,
	logger *slog.Logger,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(CORS)
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.RequestLogging(logger))

	// Health check endpoints
	r.Get("/health/live", healthHandler.LivenessHandler())
	r.Get("/health/ready", healthHandler.ReadinessHandler())

	// Cart API endpoints
	cartHandler := NewCartHandler(cartService, logger)

	r.Route("/api/v1/cart", func(r chi.Router) {
		r.Use(ContentTypeJSON)
		r.Use(UserIDFromHeader)

		r.Get("/", cartHandler.GetCart)
		r.Delete("/", cartHandler.ClearCart)

		r.Post("/items", cartHandler.AddItem)
		r.Put("/items/{productId}/{variantId}", cartHandler.UpdateItemQuantity)
		r.Delete("/items/{productId}/{variantId}", cartHandler.RemoveItem)
	})

	return r
}
