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
	r.Use(chimw.Compress(5))
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(middleware.RequestLogging(logger))
	r.Use(middleware.PrometheusMetrics("campaign"))

	// Health check endpoints
	r.Get("/health/live", healthHandler.LivenessHandler())
	r.Get("/health/ready", healthHandler.ReadinessHandler())
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	// Campaign API endpoints
	campaignHandler := NewCampaignHandler(campaignService, logger)

	r.Route("/api/v1/campaigns", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		r.Post("/", campaignHandler.CreateCampaign)
		r.Get("/", campaignHandler.ListCampaigns)

		// Stacking rules delete endpoint (must come before /{id} to avoid conflict).
		r.Delete("/stacking-rules/{ruleId}", campaignHandler.DeleteStackingRule)

		r.Get("/{id}", campaignHandler.GetCampaign)
		r.Put("/{id}", campaignHandler.UpdateCampaign)
		r.Post("/{id}/deactivate", campaignHandler.DeactivateCampaign)
		r.Post("/{id}/stacking-rules", campaignHandler.CreateStackingRule)
		r.Get("/{id}/stacking-rules", campaignHandler.GetStackingRules)
	})

	r.Route("/api/v1/coupons", func(r chi.Router) {
		r.Use(ContentTypeJSON)

		r.Post("/validate", campaignHandler.ValidateCoupon)
		r.Post("/validate-multiple", campaignHandler.ValidateMultipleCoupons)
		r.Post("/apply", campaignHandler.ApplyCoupon)
	})

	return r
}
