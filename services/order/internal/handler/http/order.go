package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/utafrali/EcommerceGo/pkg/httputil"
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
	ProductID string `json:"product_id" validate:"required,uuid"`
	VariantID string `json:"variant_id" validate:"omitempty,uuid"`
	Name      string `json:"name" validate:"required"`
	SKU       string `json:"sku"`
	Price     int64  `json:"price" validate:"required,gte=0"`
	Quantity  int    `json:"quantity" validate:"required,gte=1"`
}

// CreateOrderRequest is the JSON request body for creating an order.
type CreateOrderRequest struct {
	UserID          string                   `json:"user_id" validate:"required,uuid"`
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
	Status string `json:"status" validate:"required,oneof=pending confirmed processing shipped delivered canceled refunded"`
	Reason string `json:"reason"`
}

// CancelOrderRequest is the JSON request body for canceling an order.
type CancelOrderRequest struct {
	Reason string `json:"reason"`
}

// --- Handlers ---

// CreateOrder handles POST /api/v1/orders
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req CreateOrderRequest
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
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, httputil.Response{Data: order})
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
	if v := r.URL.Query().Get("user_id"); v != "" {
		filter.UserID = &v
	}
	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = &v
	}

	orders, total, err := h.service.ListOrders(r.Context(), filter)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.NewPaginatedResponse(orders, total, filter.Page, filter.PerPage))
}

// GetOrder handles GET /api/v1/orders/{id}
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	order, err := h.service.GetOrder(r.Context(), id.String())
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: order})
}

// UpdateOrderStatus handles PUT /api/v1/orders/{id}/status
func (h *OrderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req UpdateStatusRequest
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

	order, err := h.service.UpdateOrderStatus(r.Context(), id.String(), req.Status, req.Reason)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: order})
}

// CancelOrder handles POST /api/v1/orders/{id}/cancel
func (h *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req CancelOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body for cancel; default reason is empty.
		req = CancelOrderRequest{}
	}

	order, err := h.service.CancelOrder(r.Context(), id.String(), req.Reason)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: order})
}

