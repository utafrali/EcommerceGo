package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/pkg/httputil"
	"github.com/utafrali/EcommerceGo/pkg/slug"
	"github.com/utafrali/EcommerceGo/pkg/validator"
	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
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
// @Summary List categories
// @Description Returns a flat list of categories. Pass ?tree=true for a nested tree structure.
// @Tags categories
// @Produce json
// @Param tree query bool false "Return categories as a nested tree"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/categories [get]
func (h *CategoryHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("tree") == "true" {
		h.listCategoriesTree(w, r)
		return
	}

	categories, err := h.repo.ListAll(r.Context())
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: categories})
}

// listCategoriesTree returns categories as a nested tree.
func (h *CategoryHandler) listCategoriesTree(w http.ResponseWriter, r *http.Request) {
	tree, err := h.repo.ListTree(r.Context())
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: tree})
}

// GetCategory handles GET /api/v1/categories/{id}
// Accepts both a UUID (category ID) and a slug for lookup.
// @Summary Get category by ID or slug
// @Description Returns a category. Accepts UUID or URL slug.
// @Tags categories
// @Produce json
// @Param id path string true "Category UUID or slug"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/categories/{id} [get]
func (h *CategoryHandler) GetCategory(w http.ResponseWriter, r *http.Request) {
	idOrSlug := chi.URLParam(r, "id")
	if idOrSlug == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "category id or slug is required"},
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
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: category})
}

// CreateCategory handles POST /api/v1/categories
// @Summary Create a category
// @Description Creates a new product category
// @Tags categories
// @Accept json
// @Produce json
// @Param request body CreateCategoryRequest true "Category to create"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 422 {object} map[string]interface{}
// @Router /api/v1/categories [post]
func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		httputil.WriteValidationError(w, err)
		return
	}

	// Determine the level based on parent.
	level := 0
	if req.ParentID != nil {
		parent, err := h.repo.GetByID(r.Context(), *req.ParentID)
		if err != nil {
			if errors.Is(err, apperrors.ErrNotFound) {
				httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
					Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "parent category not found"},
				})
				return
			}
			httputil.WriteError(w, r, err, h.logger)
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
		Slug:        slug.Generate(req.Name),
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
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	h.logger.InfoContext(r.Context(), "category created",
		slog.String("category_id", category.ID),
		slog.String("slug", category.Slug),
	)

	httputil.WriteJSON(w, http.StatusCreated, httputil.Response{Data: category})
}

// UpdateCategory handles PUT /api/v1/categories/{id}
// @Summary Update a category
// @Description Partially updates a category
// @Tags categories
// @Accept json
// @Produce json
// @Param id path string true "Category UUID"
// @Param request body UpdateCategoryRequest true "Fields to update"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "category id is required"},
		})
		return
	}

	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		httputil.WriteValidationError(w, err)
		return
	}

	// Fetch the existing category.
	category, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	// Apply partial updates.
	if req.Name != nil {
		if *req.Name == "" {
			httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
				Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "category name must not be empty"},
			})
			return
		}
		category.Name = *req.Name
		category.Slug = slug.Generate(*req.Name)
	}

	if req.ParentID != nil {
		// Validate the new parent exists and prevent self-referencing.
		if *req.ParentID == id {
			httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
				Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "a category cannot be its own parent"},
			})
			return
		}
		parent, err := h.repo.GetByID(r.Context(), *req.ParentID)
		if err != nil {
			if errors.Is(err, apperrors.ErrNotFound) {
				httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
					Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "parent category not found"},
				})
				return
			}
			httputil.WriteError(w, r, err, h.logger)
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
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	h.logger.InfoContext(r.Context(), "category updated",
		slog.String("category_id", category.ID),
		slog.String("slug", category.Slug),
	)

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: category})
}

// DeleteCategory handles DELETE /api/v1/categories/{id}
// @Summary Delete a category
// @Description Deletes a category by UUID
// @Tags categories
// @Produce json
// @Param id path string true "Category UUID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "category id is required"},
		})
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	h.logger.InfoContext(r.Context(), "category deleted",
		slog.String("category_id", id),
	)

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: map[string]string{"id": id, "status": "deleted"}})
}
