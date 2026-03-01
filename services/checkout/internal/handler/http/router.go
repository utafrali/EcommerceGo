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
	"github.com/utafrali/EcommerceGo/services/checkout/internal/service"
)

// NewRouter creates a chi router with all checkout service routes registered.
func NewRouter(
	checkoutService *service.CheckoutService,
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
	r.Use(middleware.PrometheusMetrics("checkout"))
	r.Use(middleware.Tracing("checkout"))
	r.Use(middleware.RequestLogger(logger))

	// Health check endpoints
	r.Get("/health/live", healthHandler.LivenessHandler())
	r.Get("/health/ready", healthHandler.ReadinessHandler())
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	// Pprof debug endpoints with IP allowlist.
	middleware.RegisterPprof(r, pprofCIDRs, logger)

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
