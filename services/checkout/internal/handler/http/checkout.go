package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/pkg/validator"
	"github.com/utafrali/EcommerceGo/services/checkout/internal/domain"
	"github.com/utafrali/EcommerceGo/services/checkout/internal/service"
)

// CheckoutHandler handles HTTP requests for checkout endpoints.
type CheckoutHandler struct {
	service *service.CheckoutService
	logger  *slog.Logger
}

// NewCheckoutHandler creates a new checkout HTTP handler.
func NewCheckoutHandler(svc *service.CheckoutService, logger *slog.Logger) *CheckoutHandler {
	return &CheckoutHandler{
		service: svc,
		logger:  logger,
	}
}

// --- Request DTOs ---

// InitiateCheckoutRequest is the JSON request body for initiating a checkout.
type InitiateCheckoutRequest struct {
	Items    []CheckoutItemRequest `json:"items" validate:"required,min=1,dive"`
	Currency string                `json:"currency" validate:"required,len=3"`
}

// CheckoutItemRequest represents a single item in the initiate checkout request.
type CheckoutItemRequest struct {
	ProductID string `json:"product_id" validate:"required,uuid"`
	VariantID string `json:"variant_id" validate:"required,uuid"`
	Name      string `json:"name" validate:"required"`
	SKU       string `json:"sku" validate:"required"`
	Price     int64  `json:"price" validate:"required,gt=0"`
	Quantity  int    `json:"quantity" validate:"required,gt=0"`
}

// SetShippingAddressRequest is the JSON request body for setting shipping address.
type SetShippingAddressRequest struct {
	FullName    string `json:"full_name" validate:"required"`
	AddressLine string `json:"address_line" validate:"required"`
	City        string `json:"city" validate:"required"`
	State       string `json:"state"`
	PostalCode  string `json:"postal_code" validate:"required"`
	Country     string `json:"country" validate:"required"`
	Phone       string `json:"phone"`
}

// SetPaymentMethodRequest is the JSON request body for setting the payment method.
type SetPaymentMethodRequest struct {
	PaymentMethod string `json:"payment_method" validate:"required"`
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

// InitiateCheckout handles POST /api/v1/checkout
func (h *CheckoutHandler) InitiateCheckout(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "X-User-ID header is required"},
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

	var req InitiateCheckoutRequest
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

	items := make([]service.CheckoutItemInput, len(req.Items))
	for i, item := range req.Items {
		items[i] = service.CheckoutItemInput{
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			Name:      item.Name,
			SKU:       item.SKU,
			Price:     item.Price,
			Quantity:  item.Quantity,
		}
	}

	input := &service.InitiateCheckoutInput{
		Items:    items,
		Currency: req.Currency,
	}

	session, err := h.service.InitiateCheckout(r.Context(), userID, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, response{Data: session})
}

// GetCheckout handles GET /api/v1/checkout/{id}
func (h *CheckoutHandler) GetCheckout(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "checkout id is required"},
		})
		return
	}

	session, err := h.service.GetCheckout(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	if !authorizeSession(w, r, session) {
		return
	}

	writeJSON(w, http.StatusOK, response{Data: session})
}

// SetShippingAddress handles PUT /api/v1/checkout/{id}/shipping
func (h *CheckoutHandler) SetShippingAddress(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "checkout id is required"},
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

	var req SetShippingAddressRequest
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

	// Authorization: verify the caller owns this checkout session.
	existing, err := h.service.GetCheckout(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if !authorizeSession(w, r, existing) {
		return
	}

	address := &domain.Address{
		FullName:    req.FullName,
		AddressLine: req.AddressLine,
		City:        req.City,
		State:       req.State,
		PostalCode:  req.PostalCode,
		Country:     req.Country,
		Phone:       req.Phone,
	}

	session, err := h.service.SetShippingAddress(r.Context(), id, address)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: session})
}

// SetPaymentMethod handles PUT /api/v1/checkout/{id}/payment
func (h *CheckoutHandler) SetPaymentMethod(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "checkout id is required"},
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

	var req SetPaymentMethodRequest
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

	// Authorization: verify the caller owns this checkout session.
	existing, err := h.service.GetCheckout(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if !authorizeSession(w, r, existing) {
		return
	}

	session, err := h.service.SetPaymentMethod(r.Context(), id, req.PaymentMethod)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: session})
}

// ProcessCheckout handles POST /api/v1/checkout/{id}/process
func (h *CheckoutHandler) ProcessCheckout(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "checkout id is required"},
		})
		return
	}

	// Authorization: verify the caller owns this checkout session.
	existing, err := h.service.GetCheckout(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if !authorizeSession(w, r, existing) {
		return
	}

	session, err := h.service.ProcessCheckout(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: session})
}

// CancelCheckout handles POST /api/v1/checkout/{id}/cancel
func (h *CheckoutHandler) CancelCheckout(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "checkout id is required"},
		})
		return
	}

	// Authorization: verify the caller owns this checkout session.
	existing, err := h.service.GetCheckout(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if !authorizeSession(w, r, existing) {
		return
	}

	session, err := h.service.CancelCheckout(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: session})
}

// --- Helpers ---

// getUserID extracts the X-User-ID header from the request.
func getUserID(r *http.Request) string {
	return r.Header.Get("X-User-ID")
}

// authorizeSession checks that the requesting user owns the checkout session.
// Returns true if authorized, false if it wrote an error response.
func authorizeSession(w http.ResponseWriter, r *http.Request, session *domain.CheckoutSession) bool {
	userID := getUserID(r)
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "X-User-ID header is required"},
		})
		return false
	}
	if session.UserID != userID {
		writeJSON(w, http.StatusForbidden, response{
			Error: &errorResponse{Code: "FORBIDDEN", Message: "you do not have access to this checkout session"},
		})
		return false
	}
	return true
}

func (h *CheckoutHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
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

func (h *CheckoutHandler) writeValidationError(w http.ResponseWriter, err error) {
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
