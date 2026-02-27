package http

import (
	"log/slog"
	"time"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/utafrali/EcommerceGo/pkg/health"
	"github.com/utafrali/EcommerceGo/pkg/middleware"
	"github.com/utafrali/EcommerceGo/services/notification/internal/service"
)

// NewRouter creates a chi router with all notification service routes registered.
func NewRouter(
	notificationService *service.NotificationService,
	healthHandler *health.Handler,
	logger *slog.Logger,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(CORS)
	r.Use(middleware.Recovery(logger))
	r.Use(chimw.Compress(5))
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(middleware.RequestLogging(logger))
	r.Use(middleware.PrometheusMetrics("notification"))

	// Health check endpoints
	r.Get("/health/live", healthHandler.LivenessHandler())
	r.Get("/health/ready", healthHandler.ReadinessHandler())
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	// Notification API endpoints
	notificationHandler := NewNotificationHandler(notificationService, logger)

	r.Route("/api/v1/notifications", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		r.Post("/", notificationHandler.SendNotification)
		r.Get("/{id}", notificationHandler.GetNotification)
		r.Get("/user/{userId}", notificationHandler.ListNotificationsByUser)
		r.Put("/{id}/read", notificationHandler.MarkAsRead)
		r.Post("/{id}/retry", notificationHandler.RetryNotification)
	})

	return r
}
