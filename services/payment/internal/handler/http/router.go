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
	"github.com/utafrali/EcommerceGo/services/payment/internal/service"
)

// NewRouter creates a chi router with all payment service routes registered.
func NewRouter(
	paymentService *service.PaymentService,
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
	r.Use(middleware.PrometheusMetrics("payment"))
	r.Use(middleware.Tracing("payment"))
	r.Use(middleware.RequestLogger(logger))

	// Health check endpoints
	r.Get("/health/live", healthHandler.LivenessHandler())
	r.Get("/health/ready", healthHandler.ReadinessHandler())
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	// Pprof debug endpoints with IP allowlist.
	middleware.RegisterPprof(r, pprofCIDRs, logger)

	// Payment API endpoints
	paymentHandler := NewPaymentHandler(paymentService, logger)

	r.Route("/api/v1/payments", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		r.Post("/", paymentHandler.CreatePayment)
		r.Get("/{id}", paymentHandler.GetPayment)
		r.Post("/{id}/process", paymentHandler.ProcessPayment)
		r.Post("/{id}/refund", paymentHandler.RefundPayment)
		r.Get("/checkout/{checkoutId}", paymentHandler.GetPaymentByCheckoutID)
		r.Get("/user/{userId}", paymentHandler.ListPaymentsByUser)
	})

	return r
}
