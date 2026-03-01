package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/utafrali/EcommerceGo/pkg/httputil"
	"github.com/utafrali/EcommerceGo/pkg/validator"
	"github.com/utafrali/EcommerceGo/services/payment/internal/domain"
	"github.com/utafrali/EcommerceGo/services/payment/internal/service"
)

// PaymentHandler handles HTTP requests for payment endpoints.
type PaymentHandler struct {
	service *service.PaymentService
	logger  *slog.Logger
}

// NewPaymentHandler creates a new payment HTTP handler.
func NewPaymentHandler(svc *service.PaymentService, logger *slog.Logger) *PaymentHandler {
	return &PaymentHandler{
		service: svc,
		logger:  logger,
	}
}

// --- Request DTOs ---

// CreatePaymentRequest is the JSON request body for creating a payment.
type CreatePaymentRequest struct {
	CheckoutID string `json:"checkout_id" validate:"required,uuid"`
	OrderID    string `json:"order_id" validate:"required,uuid"`
	UserID     string `json:"user_id" validate:"required,uuid"`
	Amount     int64  `json:"amount" validate:"required,gt=0"`
	Currency   string `json:"currency" validate:"required,len=3"`
	Method     string `json:"method" validate:"required,oneof=credit_card debit_card bank_transfer wallet"`
}

// RefundPaymentRequest is the JSON request body for refunding a payment.
type RefundPaymentRequest struct {
	Amount int64  `json:"amount" validate:"required,gt=0"`
	Reason string `json:"reason" validate:"required,min=3"`
}

// --- Handlers ---

// CreatePayment handles POST /api/v1/payments
func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

	var req CreatePaymentRequest
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

	input := &service.CreatePaymentInput{
		CheckoutID: req.CheckoutID,
		OrderID:    req.OrderID,
		UserID:     req.UserID,
		Amount:     req.Amount,
		Currency:   req.Currency,
		Method:     req.Method,
	}

	payment, err := h.service.CreatePayment(r.Context(), input)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, httputil.Response{Data: payment})
}

// GetPayment handles GET /api/v1/payments/{id}
func (h *PaymentHandler) GetPayment(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	payment, err := h.service.GetPayment(r.Context(), id.String())
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: payment})
}

// ProcessPayment handles POST /api/v1/payments/{id}/process
func (h *PaymentHandler) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	payment, err := h.service.ProcessPayment(r.Context(), id.String())
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: payment})
}

// RefundPayment handles POST /api/v1/payments/{id}/refund
func (h *PaymentHandler) RefundPayment(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	id, ok := httputil.ParseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	var req RefundPaymentRequest
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

	input := &service.RefundPaymentInput{
		Amount: req.Amount,
		Reason: req.Reason,
	}

	refund, err := h.service.RefundPayment(r.Context(), id.String(), input)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: refund})
}

// GetPaymentByCheckoutID handles GET /api/v1/payments/checkout/{checkoutId}
func (h *PaymentHandler) GetPaymentByCheckoutID(w http.ResponseWriter, r *http.Request) {
	checkoutID, ok := httputil.ParseUUID(w, chi.URLParam(r, "checkoutId"))
	if !ok {
		return
	}

	payment, err := h.service.GetPaymentByCheckoutID(r.Context(), checkoutID.String())
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: payment})
}

// ListPaymentsByUser handles GET /api/v1/payments/user/{userId}
func (h *PaymentHandler) ListPaymentsByUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := httputil.ParseUUID(w, chi.URLParam(r, "userId"))
	if !ok {
		return
	}

	page := 1
	perPage := 20

	if v := r.URL.Query().Get("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
				Error: &httputil.ErrorResponse{Code: "INVALID_PARAMETER", Message: "page must be a valid positive integer"},
			})
			return
		}
		page = p
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		pp, err := strconv.Atoi(v)
		if err != nil || pp < 1 || pp > 100 {
			httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
				Error: &httputil.ErrorResponse{Code: "INVALID_PARAMETER", Message: "per_page must be a valid integer between 1 and 100"},
			})
			return
		}
		perPage = pp
	}

	payments, total, err := h.service.ListPaymentsByUser(r.Context(), userID.String(), page, perPage)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.NewPaginatedResponse[domain.Payment](payments, total, page, perPage))
}
