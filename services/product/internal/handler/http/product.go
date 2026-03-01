package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/utafrali/EcommerceGo/pkg/httputil"
	"github.com/utafrali/EcommerceGo/pkg/validator"
	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
	"github.com/utafrali/EcommerceGo/services/product/internal/repository"
	"github.com/utafrali/EcommerceGo/services/product/internal/service"
)

// ProductHandler handles HTTP requests for product endpoints.
type ProductHandler struct {
	service *service.ProductService
	logger  *slog.Logger
}

// NewProductHandler creates a new product HTTP handler.
func NewProductHandler(svc *service.ProductService, logger *slog.Logger) *ProductHandler {
	return &ProductHandler{
		service: svc,
		logger:  logger,
	}
}

// --- Request DTOs ---

// CreateProductRequest is the JSON request body for creating a product.
type CreateProductRequest struct {
	Name        string         `json:"name" validate:"required,min=1,max=500"`
	Description string         `json:"description"`
	BrandID     *string        `json:"brand_id" validate:"omitempty,uuid"`
	CategoryID  *string        `json:"category_id" validate:"omitempty,uuid"`
	BasePrice   int64          `json:"base_price" validate:"required,gte=0"`
	Currency    string         `json:"currency" validate:"required,len=3"`
	Metadata    map[string]any `json:"metadata"`
}

// UpdateProductRequest is the JSON request body for updating a product.
type UpdateProductRequest struct {
	Name        *string        `json:"name" validate:"omitempty,min=1,max=500"`
	Description *string        `json:"description"`
	BrandID     *string        `json:"brand_id" validate:"omitempty,uuid"`
	CategoryID  *string        `json:"category_id" validate:"omitempty,uuid"`
	Status      *string        `json:"status" validate:"omitempty,oneof=draft published archived"`
	BasePrice   *int64         `json:"base_price" validate:"omitempty,gte=0"`
	Currency    *string        `json:"currency" validate:"omitempty,len=3"`
	Metadata    map[string]any `json:"metadata"`
}

// --- Handlers ---

// ListProducts handles GET /api/v1/products
// @Summary List all products
// @Description Returns paginated list of products with optional filtering and sorting
// @Tags products
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page (max 100)" default(20)
// @Param category_id query string false "Filter by category UUID"
// @Param brand_id query string false "Filter by brand UUID"
// @Param status query string false "Filter by status" Enums(draft,published,archived)
// @Param sort_by query string false "Sort order" Enums(newest,price_asc,price_desc,name_asc,name_desc)
// @Param search query string false "Full-text search query"
// @Param min_price query int false "Minimum price in cents"
// @Param max_price query int false "Maximum price in cents"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/products [get]
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	filter := repository.ProductFilter{
		Page:    1,
		PerPage: 20,
	}

	if v := r.URL.Query().Get("page"); v != "" {
		page, err := strconv.Atoi(v)
		if err != nil || page < 1 {
			httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
				Error: &httputil.ErrorResponse{Code: "INVALID_PARAMETER", Message: "page must be a valid positive integer"},
			})
			return
		}
		filter.Page = page
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		perPage, err := strconv.Atoi(v)
		if err != nil || perPage < 1 || perPage > 100 {
			httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
				Error: &httputil.ErrorResponse{Code: "INVALID_PARAMETER", Message: "per_page must be a valid integer between 1 and 100"},
			})
			return
		}
		filter.PerPage = perPage
	}
	if v := r.URL.Query().Get("category_id"); v != "" {
		filter.CategoryID = &v
	}
	if v := r.URL.Query().Get("brand_id"); v != "" {
		filter.BrandID = &v
	}
	if v := r.URL.Query().Get("status"); v != "" {
		if !domain.IsValidStatus(v) {
			httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
				Error: &httputil.ErrorResponse{Code: "INVALID_PARAMETER", Message: "status must be one of: draft, published, archived"},
			})
			return
		}
		filter.Status = &v
	}
	if v := r.URL.Query().Get("search"); v != "" {
		filter.Search = &v
	}
	if v := r.URL.Query().Get("min_price"); v != "" {
		price, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
				Error: &httputil.ErrorResponse{Code: "INVALID_PARAMETER", Message: "min_price must be a valid number"},
			})
			return
		}
		filter.MinPrice = &price
	}
	if v := r.URL.Query().Get("max_price"); v != "" {
		price, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
				Error: &httputil.ErrorResponse{Code: "INVALID_PARAMETER", Message: "max_price must be a valid number"},
			})
			return
		}
		filter.MaxPrice = &price
	}

	if filter.MinPrice != nil && filter.MaxPrice != nil && *filter.MinPrice > *filter.MaxPrice {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_PARAMETER", Message: "min_price must not exceed max_price"},
		})
		return
	}

	if v := r.URL.Query().Get("sort_by"); v != "" {
		if !domain.IsValidSortBy(v) {
			httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
				Error: &httputil.ErrorResponse{Code: "INVALID_PARAMETER", Message: "sort_by must be one of: newest, price_asc, price_desc, name_asc, name_desc"},
			})
			return
		}
		filter.SortBy = v
	}

	products, total, err := h.service.ListProducts(r.Context(), filter)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.NewPaginatedResponse(products, total, filter.Page, filter.PerPage))
}

// GetProduct handles GET /api/v1/products/{idOrSlug}
// It accepts both a UUID (product ID) and a slug for lookup.
// Returns an enriched product detail including images, variants, category, and brand.
// @Summary Get product by ID or slug
// @Description Returns a product detail. Accepts both UUID and URL slug.
// @Tags products
// @Produce json
// @Param idOrSlug path string true "Product UUID or URL slug"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/products/{idOrSlug} [get]
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	idOrSlug := chi.URLParam(r, "idOrSlug")
	if idOrSlug == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "product id or slug is required"},
		})
		return
	}

	var (
		detail *domain.ProductDetail
		err    error
	)

	if _, parseErr := uuid.Parse(idOrSlug); parseErr == nil {
		detail, err = h.service.GetProductDetail(r.Context(), idOrSlug)
	} else {
		detail, err = h.service.GetProductDetailBySlug(r.Context(), idOrSlug)
	}

	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: detail})
}

// CreateProduct handles POST /api/v1/products
// @Summary Create a product
// @Description Creates a new product in the catalog
// @Tags products
// @Accept json
// @Produce json
// @Param request body CreateProductRequest true "Product to create"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 422 {object} map[string]interface{}
// @Router /api/v1/products [post]
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req CreateProductRequest
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

	input := &service.CreateProductInput{
		Name:        req.Name,
		Description: req.Description,
		BrandID:     req.BrandID,
		CategoryID:  req.CategoryID,
		BasePrice:   req.BasePrice,
		Currency:    req.Currency,
		Metadata:    req.Metadata,
	}

	product, err := h.service.CreateProduct(r.Context(), input)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, httputil.Response{Data: product})
}

// UpdateProduct handles PUT /api/v1/products/{id}
// @Summary Update a product
// @Description Partially updates a product — all fields are optional
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product UUID"
// @Param request body UpdateProductRequest true "Fields to update"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/products/{id} [put]
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req UpdateProductRequest
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

	input := &service.UpdateProductInput{
		Name:        req.Name,
		Description: req.Description,
		BrandID:     req.BrandID,
		CategoryID:  req.CategoryID,
		Status:      req.Status,
		BasePrice:   req.BasePrice,
		Currency:    req.Currency,
		Metadata:    req.Metadata,
	}

	product, err := h.service.UpdateProduct(r.Context(), id.String(), input)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: product})
}

// DeleteProduct handles DELETE /api/v1/products/{id}
// @Summary Delete a product
// @Description Soft-deletes a product by UUID
// @Tags products
// @Produce json
// @Param id path string true "Product UUID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/products/{id} [delete]
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
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
