package http

import (
	"log/slog"
	"net/http"

	"github.com/utafrali/EcommerceGo/pkg/httputil"
	"github.com/utafrali/EcommerceGo/services/product/internal/repository/postgres"
)

// BrandHandler handles HTTP requests for brand endpoints.
type BrandHandler struct {
	repo   *postgres.BrandRepository
	logger *slog.Logger
}

// NewBrandHandler creates a new brand HTTP handler.
func NewBrandHandler(repo *postgres.BrandRepository, logger *slog.Logger) *BrandHandler {
	return &BrandHandler{
		repo:   repo,
		logger: logger,
	}
}

// ListBrands handles GET /api/v1/brands
func (h *BrandHandler) ListBrands(w http.ResponseWriter, r *http.Request) {
	brands, err := h.repo.ListAll(r.Context())
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: brands})
}
