package handler

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/utafrali/EcommerceGo/pkg/health"
	pkgmiddleware "github.com/utafrali/EcommerceGo/pkg/middleware"
	"github.com/utafrali/EcommerceGo/services/gateway/internal/config"
	gwmiddleware "github.com/utafrali/EcommerceGo/services/gateway/internal/middleware"
	"github.com/utafrali/EcommerceGo/services/gateway/internal/proxy"
)

// NewRouter creates a chi router with global middleware, health endpoints,
// and proxy routes to all backend microservices.
func NewRouter(cfg *config.Config, sp *proxy.ServiceProxy, healthHandler *health.Handler, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	// Global middleware stack (applied in order).
	r.Use(pkgmiddleware.CORS(pkgmiddleware.CORSConfig{
		AllowedOrigins: cfg.CORSAllowedOrigins,
		AllowedMethods: cfg.CORSAllowedMethods,
		AllowedHeaders: cfg.CORSAllowedHeaders,
		ExposedHeaders: []string{"X-Correlation-ID", "X-User-ID"},
		MaxAge:         cfg.CORSMaxAge,
		Environment:    cfg.Environment,
	}))
	r.Use(gwmiddleware.RateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst, logger))
	r.Use(pkgmiddleware.Recovery(logger))
	r.Use(chimw.Compress(5))
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(pkgmiddleware.RequestLogging(logger))
	r.Use(pkgmiddleware.PrometheusMetrics("gateway"))
	r.Use(pkgmiddleware.Tracing("gateway"))
	r.Use(pkgmiddleware.RequestLogger(logger))

	// Health check endpoints (no auth required).
	r.Get("/health/live", healthHandler.LivenessHandler())
	r.Get("/health/ready", healthHandler.ReadinessHandler())

	// Metrics endpoint with IP allowlist protection.
	metricsHandler := metricsIPAllowlist(cfg.MetricsAllowedCIDRs, logger)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			promhttp.Handler().ServeHTTP(w, r)
		}),
	)
	r.Get("/metrics", metricsHandler.ServeHTTP)

	// Pprof debug endpoints with IP allowlist.
	pkgmiddleware.RegisterPprof(r, cfg.PprofAllowedCIDRs, logger)

	// JWT auth middleware applied to all /api routes.
	r.Route("/api", func(r chi.Router) {
		r.Use(gwmiddleware.JWTAuth(cfg.JWTSecret, logger))

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

// metricsIPAllowlist returns middleware that restricts access to requests
// from IPs within the configured CIDR ranges.
func metricsIPAllowlist(cidrs []string, logger *slog.Logger) func(http.Handler) http.Handler {
	var nets []*net.IPNet
	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			logger.Warn("invalid metrics CIDR, skipping", slog.String("cidr", cidr), slog.String("error", err.Error()))
			continue
		}
		nets = append(nets, ipNet)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			ip := net.ParseIP(host)

			allowed := false
			if ip != nil {
				for _, n := range nets {
					if n.Contains(ip) {
						allowed = true
						break
					}
				}
			}

			if !allowed {
				logger.Warn("metrics access denied",
					slog.String("ip", host),
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]string{
						"code":    "FORBIDDEN",
						"message": "metrics endpoint is restricted",
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
