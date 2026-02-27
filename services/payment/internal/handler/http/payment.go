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

// CreatePayment handles POST /api/v1/payments
func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

	var req CreatePaymentRequest
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
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, response{Data: payment})
}

// GetPayment handles GET /api/v1/payments/{id}
func (h *PaymentHandler) GetPayment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "payment id is required"},
		})
		return
	}

	payment, err := h.service.GetPayment(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: payment})
}

// ProcessPayment handles POST /api/v1/payments/{id}/process
func (h *PaymentHandler) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "payment id is required"},
		})
		return
	}

	payment, err := h.service.ProcessPayment(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: payment})
}

// RefundPayment handles POST /api/v1/payments/{id}/refund
func (h *PaymentHandler) RefundPayment(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "payment id is required"},
		})
		return
	}

	var req RefundPaymentRequest
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

	input := &service.RefundPaymentInput{
		Amount: req.Amount,
		Reason: req.Reason,
	}

	refund, err := h.service.RefundPayment(r.Context(), id, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: refund})
}

// GetPaymentByCheckoutID handles GET /api/v1/payments/checkout/{checkoutId}
func (h *PaymentHandler) GetPaymentByCheckoutID(w http.ResponseWriter, r *http.Request) {
	checkoutID := chi.URLParam(r, "checkoutId")
	if checkoutID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "checkout id is required"},
		})
		return
	}

	payment, err := h.service.GetPaymentByCheckoutID(r.Context(), checkoutID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: payment})
}

// ListPaymentsByUser handles GET /api/v1/payments/user/{userId}
func (h *PaymentHandler) ListPaymentsByUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "user id is required"},
		})
		return
	}

	page := 1
	perPage := 20

	if v := r.URL.Query().Get("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			writeJSON(w, http.StatusBadRequest, response{
				Error: &errorResponse{Code: "INVALID_PARAMETER", Message: "page must be a valid positive integer"},
			})
			return
		}
		page = p
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		pp, err := strconv.Atoi(v)
		if err != nil || pp < 1 || pp > 100 {
			writeJSON(w, http.StatusBadRequest, response{
				Error: &errorResponse{Code: "INVALID_PARAMETER", Message: "per_page must be a valid integer between 1 and 100"},
			})
			return
		}
		perPage = pp
	}

	payments, total, err := h.service.ListPaymentsByUser(r.Context(), userID, page, perPage)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	writeJSON(w, http.StatusOK, listResponse{
		Data:       payments,
		TotalCount: total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	})
}

// --- Helpers ---

func (h *PaymentHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
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
	case errors.Is(err, apperrors.ErrPaymentFailed):
		code = "PAYMENT_FAILED"
		message = err.Error()
		status = http.StatusUnprocessableEntity
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

func (h *PaymentHandler) writeValidationError(w http.ResponseWriter, err error) {
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
	_ = json.NewEncoder(w).Encode(v)
}
