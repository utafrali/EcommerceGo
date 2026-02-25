package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/utafrali/EcommerceGo/pkg/health"
	"github.com/utafrali/EcommerceGo/pkg/middleware"
	"github.com/utafrali/EcommerceGo/services/order/internal/service"
)

// NewRouter creates a chi router with all order service routes registered.
func NewRouter(
	orderService *service.OrderService,
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

	// Order API endpoints
	orderHandler := NewOrderHandler(orderService, logger)

	r.Route("/api/v1/orders", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		r.Post("/", orderHandler.CreateOrder)
		r.Get("/", orderHandler.ListOrders)
		r.Get("/{id}", orderHandler.GetOrder)
		r.Put("/{id}/status", orderHandler.UpdateOrderStatus)
		r.Post("/{id}/cancel", orderHandler.CancelOrder)
	})

	return r
}
