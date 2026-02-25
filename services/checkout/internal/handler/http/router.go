package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/utafrali/EcommerceGo/pkg/health"
	"github.com/utafrali/EcommerceGo/pkg/middleware"
	"github.com/utafrali/EcommerceGo/services/checkout/internal/service"
)

// NewRouter creates a chi router with all checkout service routes registered.
func NewRouter(
	checkoutService *service.CheckoutService,
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

	// Checkout API endpoints
	checkoutHandler := NewCheckoutHandler(checkoutService, logger)

	r.Route("/api/v1/checkout", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		r.Post("/", checkoutHandler.InitiateCheckout)
		r.Get("/{id}", checkoutHandler.GetCheckout)
		r.Put("/{id}/shipping", checkoutHandler.SetShippingAddress)
		r.Put("/{id}/payment", checkoutHandler.SetPaymentMethod)
		r.Post("/{id}/process", checkoutHandler.ProcessCheckout)
		r.Post("/{id}/cancel", checkoutHandler.CancelCheckout)
	})

	return r
}
