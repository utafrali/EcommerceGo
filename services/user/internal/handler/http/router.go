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
	"github.com/utafrali/EcommerceGo/services/user/docs"
	"github.com/utafrali/EcommerceGo/services/user/internal/auth"
	"github.com/utafrali/EcommerceGo/services/user/internal/domain"
	"github.com/utafrali/EcommerceGo/services/user/internal/service"
)

// NewRouter creates a chi router with all user service routes registered.
func NewRouter(
	userService *service.UserService,
	wishlistRepo domain.WishlistRepository,
	jwtManager *auth.JWTManager,
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
	r.Use(middleware.PrometheusMetrics("user"))
	r.Use(middleware.Tracing("user"))
	r.Use(middleware.RequestLogger(logger))

	// Health check endpoints
	r.Get("/health/live", healthHandler.LivenessHandler())
	r.Get("/health/ready", healthHandler.ReadinessHandler())
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	// Pprof debug endpoints with IP allowlist.
	middleware.RegisterPprof(r, pprofCIDRs, logger)

	// Auth endpoints (public)
	authHandler := NewAuthHandler(userService, logger)
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.RefreshToken)
		r.Post("/forgot-password", authHandler.ForgotPassword)
		r.Post("/reset-password", authHandler.ResetPassword)
	})

	// Token validator that bridges to our internal JWTManager.
	tokenValidator := func(token string) (*middleware.Claims, error) {
		claims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			return nil, err
		}
		return &middleware.Claims{
			UserID: claims.UserID,
			Email:  claims.Email,
			Role:   claims.Role,
		}, nil
	}

	// Authenticated auth endpoints (change password)
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Use(ContentTypeJSON)
		r.Use(middleware.Auth(tokenValidator))

		r.Post("/change-password", authHandler.ChangePassword)
	})

	// User profile, address, and wishlist endpoints (auth required)
	userHandler := NewUserHandler(userService, logger)
	wishlistHandler := NewWishlistHandler(wishlistRepo, logger)

	r.Route("/api/v1/users", func(r chi.Router) {
		r.Use(ContentTypeJSON)
		r.Use(middleware.Auth(tokenValidator))

		r.Get("/me", userHandler.GetProfile)
		r.Put("/me", userHandler.UpdateProfile)

		r.Get("/me/addresses", userHandler.ListAddresses)
		r.Post("/me/addresses", userHandler.CreateAddress)
		r.Put("/me/addresses/{id}", userHandler.UpdateAddress)
		r.Delete("/me/addresses/{id}", userHandler.DeleteAddress)

		r.Get("/wishlist", wishlistHandler.List)
		r.Post("/wishlist/{productId}", wishlistHandler.Add)
		r.Delete("/wishlist/{productId}", wishlistHandler.Remove)
		r.Get("/wishlist/{productId}", wishlistHandler.Exists)
	})

	// Swagger documentation
	r.Get("/swagger/doc.json", docs.ServeSpec)
	r.Get("/swagger/", docs.ServeUI)

	return r
}
