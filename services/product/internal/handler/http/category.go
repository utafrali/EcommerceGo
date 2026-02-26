package http

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/utafrali/EcommerceGo/services/product/internal/repository/postgres"
)

// CategoryHandler handles HTTP requests for category endpoints.
type CategoryHandler struct {
	repo   *postgres.CategoryRepository
	logger *slog.Logger
}

// NewCategoryHandler creates a new category HTTP handler.
func NewCategoryHandler(repo *postgres.CategoryRepository, logger *slog.Logger) *CategoryHandler {
	return &CategoryHandler{
		repo:   repo,
		logger: logger,
	}
}

// ListCategories handles GET /api/v1/categories
func (h *CategoryHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.repo.ListAll(r.Context())
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to list categories",
			slog.String("error", err.Error()),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
		writeJSON(w, http.StatusInternalServerError, response{
			Error: &errorResponse{Code: "INTERNAL_ERROR", Message: fmt.Sprintf("failed to list categories: %v", err)},
		})
		return
	}

	writeJSON(w, http.StatusOK, response{Data: categories})
}
