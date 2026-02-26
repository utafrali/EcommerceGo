package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/utafrali/EcommerceGo/pkg/health"
	"github.com/utafrali/EcommerceGo/pkg/middleware"
	"github.com/utafrali/EcommerceGo/services/product/internal/repository/postgres"
	"github.com/utafrali/EcommerceGo/services/product/internal/service"
)

// NewRouter creates a chi router with all product service routes registered.
func NewRouter(
	productService *service.ProductService,
	reviewService *service.ReviewService,
	categoryRepo *postgres.CategoryRepository,
	brandRepo *postgres.BrandRepository,
	healthHandler *health.Handler,
	logger *slog.Logger,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(CORS)
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.RequestLogging(logger))

	// Health check endpoints
	r.Get("/health/live", healthHandler.LivenessHandler())
	r.Get("/health/ready", healthHandler.ReadinessHandler())

	// Product API endpoints
	productHandler := NewProductHandler(productService, logger)

	r.Route("/api/v1/products", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		r.Get("/", productHandler.ListProducts)
		r.Get("/{idOrSlug}", productHandler.GetProduct)
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

	// Category API endpoints
	categoryHandler := NewCategoryHandler(categoryRepo, logger)

	r.Route("/api/v1/categories", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		r.Get("/", categoryHandler.ListCategories)
	})

	// Brand API endpoints
	brandHandler := NewBrandHandler(brandRepo, logger)

	r.Route("/api/v1/brands", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		r.Get("/", brandHandler.ListBrands)
	})

	return r
}
