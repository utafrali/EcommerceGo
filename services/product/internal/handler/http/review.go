package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/utafrali/EcommerceGo/pkg/httputil"
	"github.com/utafrali/EcommerceGo/pkg/validator"
	"github.com/utafrali/EcommerceGo/services/product/internal/service"
)

// ReviewHandler handles HTTP requests for review endpoints.
type ReviewHandler struct {
	service *service.ReviewService
	logger  *slog.Logger
}

// NewReviewHandler creates a new review HTTP handler.
func NewReviewHandler(svc *service.ReviewService, logger *slog.Logger) *ReviewHandler {
	return &ReviewHandler{
		service: svc,
		logger:  logger,
	}
}

// --- Request DTOs ---

// CreateReviewRequest is the JSON request body for creating a review.
type CreateReviewRequest struct {
	Rating int    `json:"rating" validate:"required,min=1,max=5"`
	Title  string `json:"title" validate:"max=255"`
	Body   string `json:"body"`
}

// --- Handlers ---

// ListReviews handles GET /api/v1/products/{productId}/reviews
// @Summary List product reviews
// @Description Returns paginated reviews for a product with rating summary
// @Tags reviews
// @Produce json
// @Param productId path string true "Product UUID"
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page (max 100)" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/products/{productId}/reviews [get]
func (h *ReviewHandler) ListReviews(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productId")
	if productID == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "product id is required"},
		})
		return
	}

	page := 1
	perPage := 20

	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		if pp, err := strconv.Atoi(v); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}

	result, err := h.service.ListReviews(r.Context(), productID, page, perPage)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"data":        result.Reviews,
		"summary":     result.Summary,
		"total_count": result.TotalCount,
		"page":        result.Page,
		"per_page":    result.PerPage,
		"total_pages": result.TotalPages,
	})
}

// CreateReview handles POST /api/v1/products/{productId}/reviews
// @Summary Create a product review
// @Description Submits a review for a product. Requires X-User-ID header.
// @Tags reviews
// @Accept json
// @Produce json
// @Param productId path string true "Product UUID"
// @Param X-User-ID header string true "Authenticated user UUID"
// @Param request body CreateReviewRequest true "Review to submit"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 422 {object} map[string]interface{}
// @Router /api/v1/products/{productId}/reviews [post]
func (h *ReviewHandler) CreateReview(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productId")
	if productID == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "product id is required"},
		})
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "X-User-ID header is required"},
		})
		return
	}

	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req CreateReviewRequest
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

	input := &service.CreateReviewInput{
		ProductID: productID,
		UserID:    userID,
		Rating:    req.Rating,
		Title:     req.Title,
		Body:      req.Body,
	}

	review, err := h.service.CreateReview(r.Context(), input)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, httputil.Response{Data: review})
}
