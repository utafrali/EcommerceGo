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
	"github.com/utafrali/EcommerceGo/services/search/internal/service"
)

// NewRouter creates a chi router with all search service routes registered.
func NewRouter(
	searchService *service.SearchService,
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
	r.Use(middleware.PrometheusMetrics("search"))

	// Health check endpoints
	r.Get("/health/live", healthHandler.LivenessHandler())
	r.Get("/health/ready", healthHandler.ReadinessHandler())
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	// Search API endpoints
	searchHandler := NewSearchHandler(searchService, logger)

	r.Route("/api/v1/search", func(r chi.Router) {
		r.Get("/suggest", searchHandler.Suggest)
		r.Get("/", searchHandler.Search)

		r.Group(func(r chi.Router) {
			r.Use(ContentTypeJSON)
			r.Post("/index", searchHandler.IndexProduct)
			r.Delete("/{id}", searchHandler.DeleteProduct)
			r.Post("/bulk", searchHandler.BulkIndex)
			r.Post("/reindex", searchHandler.Reindex)
		})
	})

	return r
}
