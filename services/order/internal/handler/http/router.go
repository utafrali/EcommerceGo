package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/utafrali/EcommerceGo/pkg/health"
	"github.com/utafrali/EcommerceGo/pkg/middleware"
	"github.com/utafrali/EcommerceGo/services/order/internal/service"
)

// NewRouter creates a chi router with all order service routes registered.
func NewRouter(
	orderService *service.OrderService,
	healthHandler *health.Handler,
	logger *slog.Logger,
	pprofCIDRs []string,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Recovery(logger))
	r.Use(chimw.Compress(5))
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(middleware.RequestLogging(logger))
	r.Use(middleware.PrometheusMetrics("order"))
	r.Use(middleware.Tracing("order"))
	r.Use(middleware.RequestLogger(logger))

	// Health check endpoints
	r.Get("/health/live", healthHandler.LivenessHandler())
	r.Get("/health/ready", healthHandler.ReadinessHandler())
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	// Pprof debug endpoints with IP allowlist.
	middleware.RegisterPprof(r, pprofCIDRs, logger)

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
