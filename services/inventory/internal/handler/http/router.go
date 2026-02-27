package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/utafrali/EcommerceGo/pkg/health"
	"github.com/utafrali/EcommerceGo/pkg/middleware"
	"github.com/utafrali/EcommerceGo/services/inventory/internal/service"
)

// NewRouter creates a chi router with all inventory service routes registered.
func NewRouter(
	inventoryService *service.InventoryService,
	healthHandler *health.Handler,
	logger *slog.Logger,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(CORS)
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.RequestLogging(logger))
	r.Use(middleware.PrometheusMetrics("inventory"))

	// Health check endpoints
	r.Get("/health/live", healthHandler.LivenessHandler())
	r.Get("/health/ready", healthHandler.ReadinessHandler())
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	// Inventory API endpoints
	inventoryHandler := NewInventoryHandler(inventoryService, logger)

	r.Route("/api/v1/inventory", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		// Stock initialization (create new stock record)
		r.Post("/", inventoryHandler.InitializeStock)

		// Stock operations
		r.Get("/{productId}/variants/{variantId}", inventoryHandler.GetStock)
		r.Put("/{productId}/variants/{variantId}", inventoryHandler.AdjustStock)

		// Availability and reservation operations
		r.Post("/check", inventoryHandler.CheckAvailability)
		r.Post("/reserve", inventoryHandler.ReserveStock)
		r.Post("/release", inventoryHandler.ReleaseReservation)
		r.Post("/confirm", inventoryHandler.ConfirmReservation)

		// Admin operations
		r.Get("/low-stock", inventoryHandler.ListLowStock)
	})

	return r
}
