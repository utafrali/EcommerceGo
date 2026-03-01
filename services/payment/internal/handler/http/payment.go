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
// @Summary Initialize a payment
// @Description Creates a new payment record in pending status. Use the /process endpoint to charge.
// @Tags payments
// @Accept json
// @Produce json
// @Param request body CreatePaymentRequest true "Payment creation data"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 422 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/payments/ [post]
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
// @Summary Get payment by ID
// @Description Returns a single payment record by its UUID.
// @Tags payments
// @Produce json
// @Param id path string true "Payment UUID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/payments/{id} [get]
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
// @Summary Process a payment
// @Description Submits the payment to the configured payment provider.
// @Tags payments
// @Produce json
// @Param id path string true "Payment UUID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 422 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/payments/{id}/process [post]
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
// @Summary Refund a payment
// @Description Issues a full or partial refund for a succeeded payment.
// @Tags payments
// @Accept json
// @Produce json
// @Param id path string true "Payment UUID"
// @Param request body RefundPaymentRequest true "Refund data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 422 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/payments/{id}/refund [post]
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
// @Summary Get payment by checkout ID
// @Description Returns the payment associated with a given checkout session UUID.
// @Tags payments
// @Produce json
// @Param checkoutId path string true "Checkout session UUID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/payments/checkout/{checkoutId} [get]
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
// @Summary List payments by user
// @Description Returns a paginated list of all payments for a given user.
// @Tags payments
// @Produce json
// @Param userId path string true "User UUID"
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page (max 100)" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/payments/user/{userId} [get]
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
