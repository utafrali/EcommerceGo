package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	pkgmiddleware "github.com/utafrali/EcommerceGo/pkg/middleware"
	"github.com/utafrali/EcommerceGo/services/gateway/internal/config"
	"github.com/utafrali/EcommerceGo/services/gateway/internal/middleware"
	"github.com/utafrali/EcommerceGo/services/gateway/internal/proxy"
)

// NewRouter creates a chi router with global middleware, health endpoints,
// and proxy routes to all backend microservices.
func NewRouter(cfg *config.Config, sp *proxy.ServiceProxy, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	// Global middleware stack (applied in order).
	r.Use(middleware.CORS())
	r.Use(middleware.RateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst, logger))
	r.Use(pkgmiddleware.Recovery(logger))
	r.Use(pkgmiddleware.RequestLogging(logger))

	// Health check endpoints (no auth required).
	r.Get("/health/live", LivenessHandler())
	r.Get("/health/ready", ReadinessHandler())

	// JWT auth middleware applied to all /api routes.
	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.JWTAuth(cfg.JWTSecret, logger))

		// Product Service
		r.Handle("/v1/products", sp.Handler("product"))
		r.Handle("/v1/products/*", sp.Handler("product"))

		// Categories & Brands (routed to product service)
		r.Handle("/v1/categories", sp.Handler("product"))
		r.Handle("/v1/categories/*", sp.Handler("product"))
		r.Handle("/v1/brands", sp.Handler("product"))
		r.Handle("/v1/brands/*", sp.Handler("product"))

		// Banners (routed to product service)
		r.Handle("/v1/banners", sp.Handler("product"))
		r.Handle("/v1/banners/*", sp.Handler("product"))

		// Cart Service
		r.Handle("/v1/cart", sp.Handler("cart"))
		r.Handle("/v1/cart/*", sp.Handler("cart"))

		// Order Service
		r.Handle("/v1/orders", sp.Handler("order"))
		r.Handle("/v1/orders/*", sp.Handler("order"))

		// Checkout Service
		r.Handle("/v1/checkout", sp.Handler("checkout"))
		r.Handle("/v1/checkout/*", sp.Handler("checkout"))

		// Payment Service
		r.Handle("/v1/payments", sp.Handler("payment"))
		r.Handle("/v1/payments/*", sp.Handler("payment"))

		// User Service
		r.Handle("/v1/users", sp.Handler("user"))
		r.Handle("/v1/users/*", sp.Handler("user"))
		r.Handle("/v1/auth", sp.Handler("user"))
		r.Handle("/v1/auth/*", sp.Handler("user"))

		// Inventory Service
		r.Handle("/v1/inventory", sp.Handler("inventory"))
		r.Handle("/v1/inventory/*", sp.Handler("inventory"))

		// Campaign Service
		r.Handle("/v1/campaigns", sp.Handler("campaign"))
		r.Handle("/v1/campaigns/*", sp.Handler("campaign"))
		r.Handle("/v1/coupons", sp.Handler("campaign"))
		r.Handle("/v1/coupons/*", sp.Handler("campaign"))

		// Notification Service
		r.Handle("/v1/notifications", sp.Handler("notification"))
		r.Handle("/v1/notifications/*", sp.Handler("notification"))

		// Search Service
		r.Handle("/v1/search", sp.Handler("search"))
		r.Handle("/v1/search/*", sp.Handler("search"))

		// Media Service
		r.Handle("/v1/media", sp.Handler("media"))
		r.Handle("/v1/media/*", sp.Handler("media"))
	})

	return r
}
