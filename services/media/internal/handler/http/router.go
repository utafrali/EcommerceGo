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
	"github.com/utafrali/EcommerceGo/services/media/docs"
	"github.com/utafrali/EcommerceGo/services/media/internal/service"
)

// NewRouter creates a chi router with all media service routes registered.
func NewRouter(
	mediaService *service.MediaService,
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
	r.Use(middleware.PrometheusMetrics("media"))
	r.Use(middleware.Tracing("media"))
	r.Use(middleware.RequestLogger(logger))

	// Health check endpoints
	r.Get("/health/live", healthHandler.LivenessHandler())
	r.Get("/health/ready", healthHandler.ReadinessHandler())
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	// Pprof debug endpoints with IP allowlist.
	middleware.RegisterPprof(r, pprofCIDRs, logger)

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

	// Swagger documentation
	r.Get("/swagger/doc.json", docs.ServeSpec)
	r.Get("/swagger/", docs.ServeUI)

	return r
}
