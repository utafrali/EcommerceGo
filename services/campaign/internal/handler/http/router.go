package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/utafrali/EcommerceGo/pkg/health"
	"github.com/utafrali/EcommerceGo/pkg/middleware"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/service"
)

// NewRouter creates a chi router with all campaign service routes registered.
func NewRouter(
	campaignService *service.CampaignService,
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

	// Campaign API endpoints
	campaignHandler := NewCampaignHandler(campaignService, logger)

	r.Route("/api/v1/campaigns", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		r.Post("/", campaignHandler.CreateCampaign)
		r.Get("/", campaignHandler.ListCampaigns)
		r.Get("/{id}", campaignHandler.GetCampaign)
		r.Put("/{id}", campaignHandler.UpdateCampaign)
		r.Post("/{id}/deactivate", campaignHandler.DeactivateCampaign)
	})

	r.Route("/api/v1/coupons", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		r.Post("/validate", campaignHandler.ValidateCoupon)
		r.Post("/apply", campaignHandler.ApplyCoupon)
	})

	return r
}
