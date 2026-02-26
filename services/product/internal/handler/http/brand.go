package http

import (
	"fmt"
	"log/slog"
	"net/http"

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
		h.logger.ErrorContext(r.Context(), "failed to list brands",
			slog.String("error", err.Error()),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
		writeJSON(w, http.StatusInternalServerError, response{
			Error: &errorResponse{Code: "INTERNAL_ERROR", Message: fmt.Sprintf("failed to list brands: %v", err)},
		})
		return
	}

	writeJSON(w, http.StatusOK, response{Data: brands})
}
