package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/utafrali/EcommerceGo/pkg/httputil"
	"github.com/utafrali/EcommerceGo/pkg/validator"
	"github.com/utafrali/EcommerceGo/services/search/internal/domain"
	"github.com/utafrali/EcommerceGo/services/search/internal/service"
)

// SearchHandler handles HTTP requests for search endpoints.
type SearchHandler struct {
	service *service.SearchService
	logger  *slog.Logger
}

// NewSearchHandler creates a new search HTTP handler.
func NewSearchHandler(svc *service.SearchService, logger *slog.Logger) *SearchHandler {
	return &SearchHandler{
		service: svc,
		logger:  logger,
	}
}

// --- Request DTOs ---

// IndexProductRequest is the JSON request body for indexing a product.
type IndexProductRequest struct {
	ID           string            `json:"id" validate:"required"`
	Name         string            `json:"name" validate:"required,min=1"`
	Slug         string            `json:"slug"`
	Description  string            `json:"description"`
	CategoryID   string            `json:"category_id"`
	CategoryName string            `json:"category_name"`
	BrandID      string            `json:"brand_id"`
	BrandName    string            `json:"brand_name"`
	BasePrice    int64             `json:"base_price"`
	Currency     string            `json:"currency"`
	Status       string            `json:"status"`
	ImageURL     string            `json:"image_url"`
	Tags         []string          `json:"tags"`
	Attributes   map[string]string `json:"attributes"`
}

// BulkIndexRequest is the JSON request body for bulk indexing products.
type BulkIndexRequest struct {
	Products []IndexProductRequest `json:"products" validate:"required,min=1,max=500,dive"`
}

// --- Handlers ---

// Search handles GET /api/v1/search
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	rawQuery := r.URL.Query().Get("q")
	trimmedQuery := strings.TrimSpace(rawQuery)

	sortBy := r.URL.Query().Get("sort")
	switch sortBy {
	case "", "relevance", "price_asc", "price_desc", "newest", "name_asc", "name_desc":
		// valid sort value
	default:
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{
				Code:    "INVALID_PARAMETER",
				Message: "sort must be one of: relevance, price_asc, price_desc, newest, name_asc, name_desc",
			},
		})
		return
	}

	query := &domain.SearchQuery{
		Query:   trimmedQuery,
		SortBy:  sortBy,
		Page:    1,
		PerPage: 20,
	}

	if v := r.URL.Query().Get("category_id"); v != "" {
		query.CategoryID = &v
	}
	if v := r.URL.Query().Get("brand_id"); v != "" {
		query.BrandID = &v
	}
	if v := r.URL.Query().Get("status"); v != "" {
		query.Status = &v
	}
	if v := r.URL.Query().Get("min_price"); v != "" {
		price, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
				Error: &httputil.ErrorResponse{Code: "INVALID_PARAMETER", Message: "min_price must be a valid number"},
			})
			return
		}
		if price < 0 {
			httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
				Error: &httputil.ErrorResponse{Code: "INVALID_PARAMETER", Message: "min_price must not be negative"},
			})
			return
		}
		query.MinPrice = &price
	}
	if v := r.URL.Query().Get("max_price"); v != "" {
		price, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
				Error: &httputil.ErrorResponse{Code: "INVALID_PARAMETER", Message: "max_price must be a valid number"},
			})
			return
		}
		if price < 0 {
			httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
				Error: &httputil.ErrorResponse{Code: "INVALID_PARAMETER", Message: "max_price must not be negative"},
			})
			return
		}
		query.MaxPrice = &price
	}
	if query.MinPrice != nil && query.MaxPrice != nil && *query.MinPrice > *query.MaxPrice {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_PARAMETER", Message: "min_price must not exceed max_price"},
		})
		return
	}
	if v := r.URL.Query().Get("page"); v != "" {
		if page, err := strconv.Atoi(v); err == nil && page > 0 {
			query.Page = page
		}
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		if perPage, err := strconv.Atoi(v); err == nil && perPage > 0 && perPage <= 100 {
			query.PerPage = perPage
		}
	}

	result, err := h.service.Search(r.Context(), query)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: result})
}

// IndexProduct handles POST /api/v1/search/index
func (h *SearchHandler) IndexProduct(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req IndexProductRequest
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

	input := &service.IndexProductInput{
		ID:           req.ID,
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  req.Description,
		CategoryID:   req.CategoryID,
		CategoryName: req.CategoryName,
		BrandID:      req.BrandID,
		BrandName:    req.BrandName,
		BasePrice:    req.BasePrice,
		Currency:     req.Currency,
		Status:       req.Status,
		ImageURL:     req.ImageURL,
		Tags:         req.Tags,
		Attributes:   req.Attributes,
	}

	if err := h.service.IndexProduct(r.Context(), input); err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: map[string]string{"id": req.ID, "status": "indexed"}})
}

// DeleteProduct handles DELETE /api/v1/search/{id}
func (h *SearchHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	if err := h.service.DeleteProduct(r.Context(), id.String()); err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: map[string]string{"id": id.String(), "status": "deleted"}})
}

// BulkIndex handles POST /api/v1/search/bulk
func (h *SearchHandler) BulkIndex(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB limit for bulk endpoint

	var req BulkIndexRequest
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

	inputs := make([]service.IndexProductInput, 0, len(req.Products))
	for _, p := range req.Products {
		inputs = append(inputs, service.IndexProductInput{
			ID:           p.ID,
			Name:         p.Name,
			Slug:         p.Slug,
			Description:  p.Description,
			CategoryID:   p.CategoryID,
			CategoryName: p.CategoryName,
			BrandID:      p.BrandID,
			BrandName:    p.BrandName,
			BasePrice:    p.BasePrice,
			Currency:     p.Currency,
			Status:       p.Status,
			ImageURL:     p.ImageURL,
			Tags:         p.Tags,
			Attributes:   p.Attributes,
		})
	}

	if err := h.service.BulkIndex(r.Context(), inputs); err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: map[string]any{"indexed": len(inputs), "status": "ok"}})
}

// Reindex handles POST /api/v1/search/reindex
func (h *SearchHandler) Reindex(w http.ResponseWriter, r *http.Request) {
	go func() {
		ctx := context.Background()
		if err := h.service.Reindex(ctx); err != nil {
			h.logger.ErrorContext(ctx, "background reindex failed", "error", err)
		}
	}()

	httputil.WriteJSON(w, http.StatusAccepted, httputil.Response{Data: map[string]string{"status": "reindex started"}})
}

// Suggest handles GET /api/v1/search/suggest
func (h *SearchHandler) Suggest(w http.ResponseWriter, r *http.Request) {
	prefix := strings.TrimSpace(r.URL.Query().Get("q"))
	if prefix == "" {
		httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: map[string]any{"suggestions": []string{}}})
		return
	}

	limit := 5
	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 20 {
			limit = l
		}
	}

	suggestions, err := h.service.Suggest(r.Context(), prefix, limit)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: map[string]any{"suggestions": suggestions}})
}
