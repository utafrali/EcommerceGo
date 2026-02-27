package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
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
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(CORS)
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.RequestLogging(logger))
	r.Use(middleware.PrometheusMetrics("payment"))

	// Health check endpoints
	r.Get("/health/live", healthHandler.LivenessHandler())
	r.Get("/health/ready", healthHandler.ReadinessHandler())
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

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
