package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/pkg/validator"
	"github.com/utafrali/EcommerceGo/services/cart/internal/service"
)

// CartHandler handles HTTP requests for cart endpoints.
type CartHandler struct {
	service *service.CartService
	logger  *slog.Logger
}

// NewCartHandler creates a new cart HTTP handler.
func NewCartHandler(svc *service.CartService, logger *slog.Logger) *CartHandler {
	return &CartHandler{
		service: svc,
		logger:  logger,
	}
}

// --- Request DTOs ---

// AddItemRequest is the JSON request body for adding an item to the cart.
type AddItemRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	VariantID string `json:"variant_id" validate:"required"`
	Name      string `json:"name" validate:"required,min=1,max=500"`
	SKU       string `json:"sku" validate:"required"`
	Price     int64  `json:"price" validate:"required,gte=0"`
	Quantity  int    `json:"quantity" validate:"required,gte=1"`
	ImageURL  string `json:"image_url"`
}

// UpdateQuantityRequest is the JSON request body for updating an item's quantity.
type UpdateQuantityRequest struct {
	Quantity int `json:"quantity" validate:"gte=0"`
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

// --- Handlers ---

// GetCart handles GET /api/v1/cart
func (h *CartHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "X-User-ID header is required"},
		})
		return
	}

	cart, err := h.service.GetCart(r.Context(), userID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: cart})
}

// AddItem handles POST /api/v1/cart/items
func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "X-User-ID header is required"},
		})
		return
	}

	var req AddItemRequest
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

	input := service.AddItemInput{
		ProductID: req.ProductID,
		VariantID: req.VariantID,
		Name:      req.Name,
		SKU:       req.SKU,
		Price:     req.Price,
		Quantity:  req.Quantity,
		ImageURL:  req.ImageURL,
	}

	cart, err := h.service.AddItem(r.Context(), userID, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: cart})
}

// UpdateItemQuantity handles PUT /api/v1/cart/items/{productId}/{variantId}
func (h *CartHandler) UpdateItemQuantity(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "X-User-ID header is required"},
		})
		return
	}

	productID := chi.URLParam(r, "productId")
	variantID := chi.URLParam(r, "variantId")
	if productID == "" || variantID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "productId and variantId are required"},
		})
		return
	}

	var req UpdateQuantityRequest
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

	cart, err := h.service.UpdateItemQuantity(r.Context(), userID, productID, variantID, req.Quantity)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: cart})
}

// RemoveItem handles DELETE /api/v1/cart/items/{productId}/{variantId}
func (h *CartHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "X-User-ID header is required"},
		})
		return
	}

	productID := chi.URLParam(r, "productId")
	variantID := chi.URLParam(r, "variantId")
	if productID == "" || variantID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "productId and variantId are required"},
		})
		return
	}

	cart, err := h.service.RemoveItem(r.Context(), userID, productID, variantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: cart})
}

// ClearCart handles DELETE /api/v1/cart
func (h *CartHandler) ClearCart(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "X-User-ID header is required"},
		})
		return
	}

	if err := h.service.ClearCart(r.Context(), userID); err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: map[string]string{"status": "cleared"}})
}

// --- Helpers ---

func (h *CartHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
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

	if errors.Is(err, apperrors.ErrNotFound) {
		code = "NOT_FOUND"
		message = "resource not found"
		status = http.StatusNotFound
	} else if errors.Is(err, apperrors.ErrInvalidInput) {
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

func (h *CartHandler) writeValidationError(w http.ResponseWriter, err error) {
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
