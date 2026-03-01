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
	"github.com/utafrali/EcommerceGo/services/cart/internal/service"
)

// NewRouter creates a chi router with all cart service routes registered.
func NewRouter(
	cartService *service.CartService,
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
	r.Use(middleware.PrometheusMetrics("cart"))
	r.Use(middleware.Tracing("cart"))
	r.Use(middleware.RequestLogger(logger))

	// Health check endpoints
	r.Get("/health/live", healthHandler.LivenessHandler())
	r.Get("/health/ready", healthHandler.ReadinessHandler())
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	// Pprof debug endpoints with IP allowlist.
	middleware.RegisterPprof(r, pprofCIDRs, logger)

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
