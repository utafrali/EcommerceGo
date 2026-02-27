package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/utafrali/EcommerceGo/pkg/validator"
	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
	"github.com/utafrali/EcommerceGo/services/product/internal/repository/postgres"
)

// BannerHandler handles HTTP requests for banner endpoints.
type BannerHandler struct {
	repo   *postgres.BannerRepository
	logger *slog.Logger
}

// NewBannerHandler creates a new banner HTTP handler.
func NewBannerHandler(repo *postgres.BannerRepository, logger *slog.Logger) *BannerHandler {
	return &BannerHandler{
		repo:   repo,
		logger: logger,
	}
}

// --- Request DTOs ---

// CreateBannerRequest is the JSON request body for creating a banner.
type CreateBannerRequest struct {
	Title     string     `json:"title" validate:"required,min=1,max=255"`
	Subtitle  *string    `json:"subtitle" validate:"omitempty,max=500"`
	ImageURL  string     `json:"image_url" validate:"required,url"`
	LinkURL   string     `json:"link_url" validate:"required"`
	LinkType  string     `json:"link_type" validate:"required,oneof=internal external"`
	Position  string     `json:"position" validate:"required,oneof=hero_slider mid_banner category_banner"`
	SortOrder int        `json:"sort_order" validate:"gte=0"`
	IsActive  *bool      `json:"is_active"`
	StartsAt  *time.Time `json:"starts_at"`
	EndsAt    *time.Time `json:"ends_at"`
}

// UpdateBannerRequest is the JSON request body for updating a banner.
type UpdateBannerRequest struct {
	Title     *string    `json:"title" validate:"omitempty,min=1,max=255"`
	Subtitle  *string    `json:"subtitle" validate:"omitempty,max=500"`
	ImageURL  *string    `json:"image_url" validate:"omitempty,url"`
	LinkURL   *string    `json:"link_url"`
	LinkType  *string    `json:"link_type" validate:"omitempty,oneof=internal external"`
	Position  *string    `json:"position" validate:"omitempty,oneof=hero_slider mid_banner category_banner"`
	SortOrder *int       `json:"sort_order" validate:"omitempty,gte=0"`
	IsActive  *bool      `json:"is_active"`
	StartsAt  *time.Time `json:"starts_at"`
	EndsAt    *time.Time `json:"ends_at"`
}

// --- Handlers ---

// ListBanners handles GET /api/v1/banners
func (h *BannerHandler) ListBanners(w http.ResponseWriter, r *http.Request) {
	filter := domain.BannerFilter{
		Page:    1,
		PerPage: 20,
	}

	if v := r.URL.Query().Get("page"); v != "" {
		if page, err := strconv.Atoi(v); err == nil && page > 0 {
			filter.Page = page
		}
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		if perPage, err := strconv.Atoi(v); err == nil && perPage > 0 && perPage <= 100 {
			filter.PerPage = perPage
		}
	}
	if v := r.URL.Query().Get("position"); v != "" {
		filter.Position = &v
	}
	if v := r.URL.Query().Get("is_active"); v != "" {
		active := strings.ToLower(v) == "true"
		filter.IsActive = &active
	}

	banners, total, err := h.repo.List(r.Context(), filter)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	totalPages := total / filter.PerPage
	if total%filter.PerPage > 0 {
		totalPages++
	}

	writeJSON(w, http.StatusOK, listResponse{
		Data:       banners,
		TotalCount: total,
		Page:       filter.Page,
		PerPage:    filter.PerPage,
		TotalPages: totalPages,
	})
}

// GetBanner handles GET /api/v1/banners/{id}
func (h *BannerHandler) GetBanner(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "banner id is required"},
		})
		return
	}

	banner, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: banner})
}

// CreateBanner handles POST /api/v1/banners
func (h *BannerHandler) CreateBanner(w http.ResponseWriter, r *http.Request) {
	var req CreateBannerRequest
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

	now := time.Now().UTC()
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	banner := &domain.Banner{
		ID:        uuid.New().String(),
		Title:     req.Title,
		Subtitle:  req.Subtitle,
		ImageURL:  req.ImageURL,
		LinkURL:   req.LinkURL,
		LinkType:  req.LinkType,
		Position:  req.Position,
		SortOrder: req.SortOrder,
		IsActive:  isActive,
		StartsAt:  req.StartsAt,
		EndsAt:    req.EndsAt,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.repo.Create(r.Context(), banner); err != nil {
		h.writeError(w, r, err)
		return
	}

	h.logger.InfoContext(r.Context(), "banner created",
		slog.String("banner_id", banner.ID),
		slog.String("position", banner.Position),
	)

	writeJSON(w, http.StatusCreated, response{Data: banner})
}

// UpdateBanner handles PUT /api/v1/banners/{id}
func (h *BannerHandler) UpdateBanner(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "banner id is required"},
		})
		return
	}

	var req UpdateBannerRequest
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

	// Fetch existing banner.
	banner, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// Apply partial updates.
	if req.Title != nil {
		banner.Title = *req.Title
	}
	if req.Subtitle != nil {
		banner.Subtitle = req.Subtitle
	}
	if req.ImageURL != nil {
		banner.ImageURL = *req.ImageURL
	}
	if req.LinkURL != nil {
		banner.LinkURL = *req.LinkURL
	}
	if req.LinkType != nil {
		banner.LinkType = *req.LinkType
	}
	if req.Position != nil {
		banner.Position = *req.Position
	}
	if req.SortOrder != nil {
		banner.SortOrder = *req.SortOrder
	}
	if req.IsActive != nil {
		banner.IsActive = *req.IsActive
	}
	if req.StartsAt != nil {
		banner.StartsAt = req.StartsAt
	}
	if req.EndsAt != nil {
		banner.EndsAt = req.EndsAt
	}

	if err := h.repo.Update(r.Context(), banner); err != nil {
		h.writeError(w, r, err)
		return
	}

	h.logger.InfoContext(r.Context(), "banner updated",
		slog.String("banner_id", banner.ID),
		slog.String("position", banner.Position),
	)

	writeJSON(w, http.StatusOK, response{Data: banner})
}

// DeleteBanner handles DELETE /api/v1/banners/{id}
func (h *BannerHandler) DeleteBanner(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "banner id is required"},
		})
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.writeError(w, r, err)
		return
	}

	h.logger.InfoContext(r.Context(), "banner deleted",
		slog.String("banner_id", id),
	)

	writeJSON(w, http.StatusOK, response{Data: map[string]string{"id": id, "status": "deleted"}})
}

// --- Helpers (reuse patterns from ProductHandler) ---

func (h *BannerHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	handleWriteError(w, r, err, h.logger)
}

func (h *BannerHandler) writeValidationError(w http.ResponseWriter, err error) {
	handleWriteValidationError(w, err)
}
