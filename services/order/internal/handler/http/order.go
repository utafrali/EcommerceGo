package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/pkg/validator"
	"github.com/utafrali/EcommerceGo/services/order/internal/domain"
	"github.com/utafrali/EcommerceGo/services/order/internal/repository"
	"github.com/utafrali/EcommerceGo/services/order/internal/service"
)

// OrderHandler handles HTTP requests for order endpoints.
type OrderHandler struct {
	service *service.OrderService
	logger  *slog.Logger
}

// NewOrderHandler creates a new order HTTP handler.
func NewOrderHandler(svc *service.OrderService, logger *slog.Logger) *OrderHandler {
	return &OrderHandler{
		service: svc,
		logger:  logger,
	}
}

// --- Request DTOs ---

// CreateOrderItemRequest is the JSON request body for an order line item.
type CreateOrderItemRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	VariantID string `json:"variant_id"`
	Name      string `json:"name" validate:"required"`
	SKU       string `json:"sku"`
	Price     int64  `json:"price" validate:"required,gte=0"`
	Quantity  int    `json:"quantity" validate:"required,gte=1"`
}

// CreateOrderRequest is the JSON request body for creating an order.
type CreateOrderRequest struct {
	UserID          string                   `json:"user_id" validate:"required"`
	Items           []CreateOrderItemRequest `json:"items" validate:"required,min=1,dive"`
	DiscountAmount  int64                    `json:"discount_amount" validate:"gte=0"`
	ShippingAmount  int64                    `json:"shipping_amount" validate:"gte=0"`
	Currency        string                   `json:"currency" validate:"required,len=3"`
	ShippingAddress *domain.Address          `json:"shipping_address"`
	BillingAddress  *domain.Address          `json:"billing_address"`
	Notes           string                   `json:"notes"`
}

// UpdateStatusRequest is the JSON request body for updating order status.
type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required"`
	Reason string `json:"reason"`
}

// CancelOrderRequest is the JSON request body for canceling an order.
type CancelOrderRequest struct {
	Reason string `json:"reason"`
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
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
}

// --- Handlers ---

// CreateOrder handles POST /api/v1/orders
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req CreateOrderRequest
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

	items := make([]service.CreateOrderItemInput, len(req.Items))
	for i, item := range req.Items {
		items[i] = service.CreateOrderItemInput{
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			Name:      item.Name,
			SKU:       item.SKU,
			Price:     item.Price,
			Quantity:  item.Quantity,
		}
	}

	input := service.CreateOrderInput{
		UserID:          req.UserID,
		Items:           items,
		DiscountAmount:  req.DiscountAmount,
		ShippingAmount:  req.ShippingAmount,
		Currency:        req.Currency,
		ShippingAddress: req.ShippingAddress,
		BillingAddress:  req.BillingAddress,
		Notes:           req.Notes,
	}

	order, err := h.service.CreateOrder(r.Context(), input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, response{Data: order})
}

// ListOrders handles GET /api/v1/orders
func (h *OrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	filter := repository.OrderFilter{
		Page:    1,
		PerPage: 20,
	}

	if v := r.URL.Query().Get("page"); v != "" {
		page, err := strconv.Atoi(v)
		if err != nil || page < 1 {
			writeJSON(w, http.StatusBadRequest, response{
				Error: &errorResponse{Code: "INVALID_PARAMETER", Message: "page must be a valid positive integer"},
			})
			return
		}
		filter.Page = page
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		perPage, err := strconv.Atoi(v)
		if err != nil || perPage < 1 || perPage > 100 {
			writeJSON(w, http.StatusBadRequest, response{
				Error: &errorResponse{Code: "INVALID_PARAMETER", Message: "per_page must be a valid integer between 1 and 100"},
			})
			return
		}
		filter.PerPage = perPage
	}
	if v := r.URL.Query().Get("user_id"); v != "" {
		filter.UserID = &v
	}
	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = &v
	}

	orders, total, err := h.service.ListOrders(r.Context(), filter)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	totalPages := total / filter.PerPage
	if total%filter.PerPage > 0 {
		totalPages++
	}

	writeJSON(w, http.StatusOK, listResponse{
		Data:       orders,
		TotalCount: total,
		Page:       filter.Page,
		PerPage:    filter.PerPage,
		TotalPages: totalPages,
		HasNext:    filter.Page < totalPages,
	})
}

// GetOrder handles GET /api/v1/orders/{id}
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "order id is required"},
		})
		return
	}

	order, err := h.service.GetOrder(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: order})
}

// UpdateOrderStatus handles PUT /api/v1/orders/{id}/status
func (h *OrderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "order id is required"},
		})
		return
	}

	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req UpdateStatusRequest
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

	order, err := h.service.UpdateOrderStatus(r.Context(), id, req.Status, req.Reason)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: order})
}

// CancelOrder handles POST /api/v1/orders/{id}/cancel
func (h *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "order id is required"},
		})
		return
	}

	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req CancelOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body for cancel; default reason is empty.
		req = CancelOrderRequest{}
	}

	order, err := h.service.CancelOrder(r.Context(), id, req.Reason)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: order})
}

// --- Helpers ---

func (h *OrderHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
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
	} else if errors.Is(err, apperrors.ErrAlreadyExists) {
		code = "ALREADY_EXISTS"
		message = "resource already exists"
		status = http.StatusConflict
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

func (h *OrderHandler) writeValidationError(w http.ResponseWriter, err error) {
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
