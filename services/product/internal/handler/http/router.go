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
	"github.com/utafrali/EcommerceGo/services/product/docs"
	"github.com/utafrali/EcommerceGo/services/product/internal/repository/postgres"
	"github.com/utafrali/EcommerceGo/services/product/internal/service"
)

// NewRouter creates a chi router with all product service routes registered.
func NewRouter(
	productService *service.ProductService,
	reviewService *service.ReviewService,
	categoryRepo *postgres.CategoryRepository,
	brandRepo *postgres.BrandRepository,
	bannerRepo *postgres.BannerRepository,
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
	r.Use(middleware.PrometheusMetrics("product"))
	r.Use(middleware.Tracing("product"))
	r.Use(middleware.RequestLogger(logger))

	// Health check endpoints
	r.Get("/health/live", healthHandler.LivenessHandler())
	r.Get("/health/ready", healthHandler.ReadinessHandler())
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	// Pprof debug endpoints with IP allowlist.
	middleware.RegisterPprof(r, pprofCIDRs, logger)

	// Product API endpoints
	productHandler := NewProductHandler(productService, logger)

	r.Route("/api/v1/products", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		// GET listing routes with cache headers (60s)
		r.Group(func(r chi.Router) {
			r.Use(middleware.CacheControl(60))
			r.Get("/", productHandler.ListProducts)
			r.Get("/{idOrSlug}", productHandler.GetProduct)
		})

		r.Post("/", productHandler.CreateProduct)
		r.Put("/{id}", productHandler.UpdateProduct)
		r.Delete("/{id}", productHandler.DeleteProduct)
	})

	// Review API endpoints (nested under products)
	reviewHandler := NewReviewHandler(reviewService, logger)

	r.Route("/api/v1/products/{productId}/reviews", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		r.Get("/", reviewHandler.ListReviews)
		r.Post("/", reviewHandler.CreateReview)
	})

	// Category API endpoints (full CRUD + tree)
	categoryHandler := NewCategoryHandler(categoryRepo, logger)

	r.Route("/api/v1/categories", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		// GET listing routes with cache headers (60s)
		r.Group(func(r chi.Router) {
			r.Use(middleware.CacheControl(60))
			r.Get("/", categoryHandler.ListCategories)
			r.Get("/{id}", categoryHandler.GetCategory)
		})

		r.Post("/", categoryHandler.CreateCategory)
		r.Put("/{id}", categoryHandler.UpdateCategory)
		r.Delete("/{id}", categoryHandler.DeleteCategory)
	})

	// Brand API endpoints
	brandHandler := NewBrandHandler(brandRepo, logger)

	r.Route("/api/v1/brands", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		// GET listing routes with cache headers (60s)
		r.Group(func(r chi.Router) {
			r.Use(middleware.CacheControl(60))
			r.Get("/", brandHandler.ListBrands)
		})
	})

	// Banner API endpoints
	bannerHandler := NewBannerHandler(bannerRepo, logger)

	r.Route("/api/v1/banners", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		// GET listing routes with cache headers (60s)
		r.Group(func(r chi.Router) {
			r.Use(middleware.CacheControl(60))
			r.Get("/", bannerHandler.ListBanners)
			r.Get("/{id}", bannerHandler.GetBanner)
		})

		r.Post("/", bannerHandler.CreateBanner)
		r.Put("/{id}", bannerHandler.UpdateBanner)
		r.Delete("/{id}", bannerHandler.DeleteBanner)
	})

	// Swagger documentation
	r.Get("/swagger/doc.json", docs.ServeSpec)
	r.Get("/swagger/", docs.ServeUI)

	return r
}
