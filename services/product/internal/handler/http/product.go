package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
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

// --- Response envelope ---

type response struct {
	Data  any            `json:"data,omitempty"`
	Error *errorResponse `json:"error,omitempty"`
}

type errorResponse struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type listResponse struct {
	Data       any `json:"data"`
	TotalCount int `json:"total_count"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

// --- Handlers ---

// ListProducts handles GET /api/v1/products
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	filter := repository.ProductFilter{
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
	if v := r.URL.Query().Get("category_id"); v != "" {
		filter.CategoryID = &v
	}
	if v := r.URL.Query().Get("brand_id"); v != "" {
		filter.BrandID = &v
	}
	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = &v
	}
	if v := r.URL.Query().Get("search"); v != "" {
		filter.Search = &v
	}
	if v := r.URL.Query().Get("min_price"); v != "" {
		if price, err := strconv.ParseInt(v, 10, 64); err == nil {
			filter.MinPrice = &price
		}
	}
	if v := r.URL.Query().Get("max_price"); v != "" {
		if price, err := strconv.ParseInt(v, 10, 64); err == nil {
			filter.MaxPrice = &price
		}
	}

	products, total, err := h.service.ListProducts(r.Context(), filter)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	totalPages := total / filter.PerPage
	if total%filter.PerPage > 0 {
		totalPages++
	}

	writeJSON(w, http.StatusOK, listResponse{
		Data:       products,
		TotalCount: total,
		Page:       filter.Page,
		PerPage:    filter.PerPage,
		TotalPages: totalPages,
	})
}

// GetProduct handles GET /api/v1/products/{idOrSlug}
// It accepts both a UUID (product ID) and a slug for lookup.
// Returns an enriched product detail including images, variants, category, and brand.
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	idOrSlug := chi.URLParam(r, "idOrSlug")
	if idOrSlug == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "product id or slug is required"},
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
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: detail})
}

// GetProductBySlug handles GET /api/v1/products/{slug}
func (h *ProductHandler) GetProductBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "slug is required"},
		})
		return
	}

	detail, err := h.service.GetProductDetailBySlug(r.Context(), slug)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: detail})
}

// CreateProduct handles POST /api/v1/products
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req CreateProductRequest
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
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, response{Data: product})
}

// UpdateProduct handles PUT /api/v1/products/{id}
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "product id is required"},
		})
		return
	}

	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req UpdateProductRequest
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

	product, err := h.service.UpdateProduct(r.Context(), id, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: product})
}

// DeleteProduct handles DELETE /api/v1/products/{id}
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "product id is required"},
		})
		return
	}

	if err := h.service.DeleteProduct(r.Context(), id); err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: map[string]string{"id": id, "status": "deleted"}})
}

// --- Helpers ---

func (h *ProductHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		writeJSON(w, appErr.Status, response{
			Error: &errorResponse{Code: appErr.Code, Message: appErr.Message},
		})
		return
	}

	status := apperrors.HTTPStatus(err)
	code := "INTERNAL_ERROR"
	message := "an internal error occurred"

	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		code = "NOT_FOUND"
		message = "resource not found"
		status = http.StatusNotFound
	case errors.Is(err, apperrors.ErrAlreadyExists):
		code = "ALREADY_EXISTS"
		message = "resource already exists"
		status = http.StatusConflict
	case errors.Is(err, apperrors.ErrInvalidInput):
		code = "INVALID_INPUT"
		message = err.Error()
		status = http.StatusBadRequest
	}

	if status == http.StatusInternalServerError {
		h.logger.ErrorContext(r.Context(), "internal error",
			slog.String("error", err.Error()),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
	}

	writeJSON(w, status, response{
		Error: &errorResponse{Code: code, Message: message},
	})
}

func (h *ProductHandler) writeValidationError(w http.ResponseWriter, err error) {
	var valErr *validator.ValidationError
	if errors.As(err, &valErr) {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{
				Code:    "VALIDATION_ERROR",
				Message: "request validation failed",
				Fields:  valErr.Fields(),
			},
		})
		return
	}

	writeJSON(w, http.StatusBadRequest, response{
		Error: &errorResponse{Code: "INVALID_INPUT", Message: err.Error()},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Headers are already sent; nothing meaningful can be done if encoding fails.
	_ = json.NewEncoder(w).Encode(v)
}

// handleWriteError is a standalone error writer usable by any handler in this package.
func handleWriteError(w http.ResponseWriter, r *http.Request, err error, logger *slog.Logger) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		writeJSON(w, appErr.Status, response{
			Error: &errorResponse{Code: appErr.Code, Message: appErr.Message},
		})
		return
	}

	status := apperrors.HTTPStatus(err)
	code := "INTERNAL_ERROR"
	message := "an internal error occurred"

	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		code = "NOT_FOUND"
		message = "resource not found"
		status = http.StatusNotFound
	case errors.Is(err, apperrors.ErrAlreadyExists):
		code = "ALREADY_EXISTS"
		message = "resource already exists"
		status = http.StatusConflict
	case errors.Is(err, apperrors.ErrInvalidInput):
		code = "INVALID_INPUT"
		message = err.Error()
		status = http.StatusBadRequest
	}

	if status == http.StatusInternalServerError {
		logger.ErrorContext(r.Context(), "internal error",
			slog.String("error", err.Error()),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
	}

	writeJSON(w, status, response{
		Error: &errorResponse{Code: code, Message: message},
	})
}

// handleWriteValidationError is a standalone validation error writer usable by any handler in this package.
func handleWriteValidationError(w http.ResponseWriter, err error) {
	var valErr *validator.ValidationError
	if errors.As(err, &valErr) {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{
				Code:    "VALIDATION_ERROR",
				Message: "request validation failed",
				Fields:  valErr.Fields(),
			},
		})
		return
	}

	writeJSON(w, http.StatusBadRequest, response{
		Error: &errorResponse{Code: "INVALID_INPUT", Message: err.Error()},
	})
}
