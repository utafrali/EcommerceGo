package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/utafrali/EcommerceGo/pkg/health"
	"github.com/utafrali/EcommerceGo/pkg/middleware"
	"github.com/utafrali/EcommerceGo/services/media/internal/service"
)

// NewRouter creates a chi router with all media service routes registered.
func NewRouter(
	mediaService *service.MediaService,
	healthHandler *health.Handler,
	logger *slog.Logger,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(CORS)
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.RequestLogging(logger))
	r.Use(middleware.PrometheusMetrics("media"))

	// Health check endpoints
	r.Get("/health/live", healthHandler.LivenessHandler())
	r.Get("/health/ready", healthHandler.ReadinessHandler())
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	// Media API endpoints
	mediaHandler := NewMediaHandler(mediaService, logger)

	r.Route("/api/v1/media", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		r.Post("/", mediaHandler.UploadMedia)
		r.Get("/{id}", mediaHandler.GetMedia)
		r.Get("/owner/{ownerType}/{ownerId}", mediaHandler.ListMediaByOwner)
		r.Put("/{id}", mediaHandler.UpdateMediaMetadata)
		r.Delete("/{id}", mediaHandler.DeleteMedia)
	})

	return r
}
