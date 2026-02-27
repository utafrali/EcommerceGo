package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/pkg/validator"
	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
	"github.com/utafrali/EcommerceGo/services/product/internal/repository/postgres"
)

// categorySlugRegexp matches characters not allowed in a slug.
var categorySlugRegexp = regexp.MustCompile(`[^a-z0-9\-]+`)

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

// --- Request DTOs ---

// CreateCategoryRequest is the JSON request body for creating a category.
type CreateCategoryRequest struct {
	Name        string  `json:"name" validate:"required,min=1,max=255"`
	ParentID    *string `json:"parent_id" validate:"omitempty,uuid"`
	SortOrder   int     `json:"sort_order" validate:"gte=0"`
	IsActive    *bool   `json:"is_active"`
	ImageURL    *string `json:"image_url" validate:"omitempty,url"`
	IconURL     *string `json:"icon_url" validate:"omitempty,url"`
	Description *string `json:"description"`
}

// UpdateCategoryRequest is the JSON request body for updating a category.
type UpdateCategoryRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=1,max=255"`
	ParentID    *string `json:"parent_id" validate:"omitempty,uuid"`
	SortOrder   *int    `json:"sort_order" validate:"omitempty,gte=0"`
	IsActive    *bool   `json:"is_active"`
	ImageURL    *string `json:"image_url" validate:"omitempty"`
	IconURL     *string `json:"icon_url" validate:"omitempty"`
	Description *string `json:"description"`
}

// --- Handlers ---

// ListCategories handles GET /api/v1/categories
// Returns a flat list by default. Pass ?tree=true to get a nested tree structure.
func (h *CategoryHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("tree") == "true" {
		h.listCategoriesTree(w, r)
		return
	}

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

// listCategoriesTree returns categories as a nested tree.
func (h *CategoryHandler) listCategoriesTree(w http.ResponseWriter, r *http.Request) {
	tree, err := h.repo.ListTree(r.Context())
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to list categories tree",
			slog.String("error", err.Error()),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
		writeJSON(w, http.StatusInternalServerError, response{
			Error: &errorResponse{Code: "INTERNAL_ERROR", Message: fmt.Sprintf("failed to list categories tree: %v", err)},
		})
		return
	}

	writeJSON(w, http.StatusOK, response{Data: tree})
}

// GetCategory handles GET /api/v1/categories/{id}
// Accepts both a UUID (category ID) and a slug for lookup.
func (h *CategoryHandler) GetCategory(w http.ResponseWriter, r *http.Request) {
	idOrSlug := chi.URLParam(r, "id")
	if idOrSlug == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "category id or slug is required"},
		})
		return
	}

	var (
		category *domain.Category
		err      error
	)

	if _, parseErr := uuid.Parse(idOrSlug); parseErr == nil {
		category, err = h.repo.GetByID(r.Context(), idOrSlug)
	} else {
		category, err = h.repo.GetBySlug(r.Context(), idOrSlug)
	}

	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: category})
}

// CreateCategory handles POST /api/v1/categories
func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	// Determine the level based on parent.
	level := 0
	if req.ParentID != nil {
		parent, err := h.repo.GetByID(r.Context(), *req.ParentID)
		if err != nil {
			if errors.Is(err, apperrors.ErrNotFound) {
				writeJSON(w, http.StatusBadRequest, response{
					Error: &errorResponse{Code: "INVALID_INPUT", Message: "parent category not found"},
				})
				return
			}
			h.writeError(w, r, err)
			return
		}
		level = parent.Level + 1
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	now := time.Now().UTC()
	category := &domain.Category{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Slug:        generateCategorySlug(req.Name),
		ParentID:    req.ParentID,
		SortOrder:   req.SortOrder,
		IsActive:    isActive,
		ImageURL:    req.ImageURL,
		IconURL:     req.IconURL,
		Description: req.Description,
		Level:       level,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.repo.Create(r.Context(), category); err != nil {
		h.writeError(w, r, err)
		return
	}

	h.logger.InfoContext(r.Context(), "category created",
		slog.String("category_id", category.ID),
		slog.String("slug", category.Slug),
	)

	writeJSON(w, http.StatusCreated, response{Data: category})
}

// UpdateCategory handles PUT /api/v1/categories/{id}
func (h *CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "category id is required"},
		})
		return
	}

	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	// Fetch the existing category.
	category, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// Apply partial updates.
	if req.Name != nil {
		if *req.Name == "" {
			writeJSON(w, http.StatusBadRequest, response{
				Error: &errorResponse{Code: "INVALID_INPUT", Message: "category name must not be empty"},
			})
			return
		}
		category.Name = *req.Name
		category.Slug = generateCategorySlug(*req.Name)
	}

	if req.ParentID != nil {
		// Validate the new parent exists and prevent self-referencing.
		if *req.ParentID == id {
			writeJSON(w, http.StatusBadRequest, response{
				Error: &errorResponse{Code: "INVALID_INPUT", Message: "a category cannot be its own parent"},
			})
			return
		}
		parent, err := h.repo.GetByID(r.Context(), *req.ParentID)
		if err != nil {
			if errors.Is(err, apperrors.ErrNotFound) {
				writeJSON(w, http.StatusBadRequest, response{
					Error: &errorResponse{Code: "INVALID_INPUT", Message: "parent category not found"},
				})
				return
			}
			h.writeError(w, r, err)
			return
		}
		category.ParentID = req.ParentID
		category.Level = parent.Level + 1
	}

	if req.SortOrder != nil {
		category.SortOrder = *req.SortOrder
	}

	if req.IsActive != nil {
		category.IsActive = *req.IsActive
	}

	if req.ImageURL != nil {
		category.ImageURL = req.ImageURL
	}

	if req.IconURL != nil {
		category.IconURL = req.IconURL
	}

	if req.Description != nil {
		category.Description = req.Description
	}

	if err := h.repo.Update(r.Context(), category); err != nil {
		h.writeError(w, r, err)
		return
	}

	h.logger.InfoContext(r.Context(), "category updated",
		slog.String("category_id", category.ID),
		slog.String("slug", category.Slug),
	)

	writeJSON(w, http.StatusOK, response{Data: category})
}

// DeleteCategory handles DELETE /api/v1/categories/{id}
func (h *CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "category id is required"},
		})
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.writeError(w, r, err)
		return
	}

	h.logger.InfoContext(r.Context(), "category deleted",
		slog.String("category_id", id),
	)

	writeJSON(w, http.StatusOK, response{Data: map[string]string{"id": id, "status": "deleted"}})
}

// --- Helpers ---

func (h *CategoryHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	handleWriteError(w, r, err, h.logger)
}

func (h *CategoryHandler) writeValidationError(w http.ResponseWriter, err error) {
	handleWriteValidationError(w, err)
}

// generateCategorySlug creates a URL-friendly slug from the given name.
// Supports Turkish characters by transliterating them before slugifying.
func generateCategorySlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))

	// Transliterate common Turkish characters.
	replacer := strings.NewReplacer(
		"\u00e7", "c", // c with cedilla
		"\u011f", "g", // g with breve
		"\u0131", "i", // dotless i
		"\u00f6", "o", // o with umlaut
		"\u015f", "s", // s with cedilla
		"\u00fc", "u", // u with umlaut
		"\u00c7", "c", // capital C with cedilla
		"\u011e", "g", // capital G with breve
		"\u0130", "i", // capital I with dot
		"\u00d6", "o", // capital O with umlaut
		"\u015e", "s", // capital S with cedilla
		"\u00dc", "u", // capital U with umlaut
	)
	slug = replacer.Replace(slug)

	slug = categorySlugRegexp.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")

	// Collapse consecutive hyphens.
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	return slug
}
