package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/utafrali/EcommerceGo/pkg/health"
	"github.com/utafrali/EcommerceGo/pkg/middleware"
	"github.com/utafrali/EcommerceGo/services/user/internal/auth"
	"github.com/utafrali/EcommerceGo/services/user/internal/service"
)

// NewRouter creates a chi router with all user service routes registered.
func NewRouter(
	userService *service.UserService,
	jwtManager *auth.JWTManager,
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

	// Auth endpoints (public)
	authHandler := NewAuthHandler(userService)
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.RefreshToken)
		r.Post("/forgot-password", authHandler.ForgotPassword)
		r.Post("/reset-password", authHandler.ResetPassword)
	})

	// User profile and address endpoints (auth required)
	userHandler := NewUserHandler(userService)

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

	r.Route("/api/v1/users", func(r chi.Router) {
		r.Use(ContentTypeJSON)
		r.Use(middleware.Auth(tokenValidator))

		r.Get("/me", userHandler.GetProfile)
		r.Put("/me", userHandler.UpdateProfile)

		r.Get("/me/addresses", userHandler.ListAddresses)
		r.Post("/me/addresses", userHandler.CreateAddress)
		r.Put("/me/addresses/{id}", userHandler.UpdateAddress)
		r.Delete("/me/addresses/{id}", userHandler.DeleteAddress)
	})

	return r
}
