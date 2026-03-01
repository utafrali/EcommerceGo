package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/utafrali/EcommerceGo/pkg/httputil"
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

// --- Handlers ---

// InitiateCheckout handles POST /api/v1/checkout
// @Summary Initiate a checkout session
// @Description Creates a new checkout session for the authenticated user. Requires X-User-ID header.
// @Tags checkout
// @Accept json
// @Produce json
// @Param X-User-ID header string true "Authenticated user UUID"
// @Param request body InitiateCheckoutRequest true "Checkout initiation data"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 422 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/checkout/ [post]
func (h *CheckoutHandler) InitiateCheckout(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "X-User-ID header is required"},
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

	var req InitiateCheckoutRequest
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
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, httputil.Response{Data: session})
}

// GetCheckout handles GET /api/v1/checkout/{id}
// @Summary Get checkout session
// @Description Returns a checkout session by ID. Only the session owner (X-User-ID) may access it.
// @Tags checkout
// @Produce json
// @Param id path string true "Checkout session UUID"
// @Param X-User-ID header string true "Authenticated user UUID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/checkout/{id} [get]
func (h *CheckoutHandler) GetCheckout(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "checkout id is required"},
		})
		return
	}

	session, err := h.service.GetCheckout(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	if !authorizeSession(w, r, session) {
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: session})
}

// SetShippingAddress handles PUT /api/v1/checkout/{id}/shipping
// @Summary Set shipping address
// @Description Sets or updates the shipping address on the checkout session.
// @Tags checkout
// @Accept json
// @Produce json
// @Param id path string true "Checkout session UUID"
// @Param X-User-ID header string true "Authenticated user UUID"
// @Param request body SetShippingAddressRequest true "Shipping address data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 422 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/checkout/{id}/shipping [put]
func (h *CheckoutHandler) SetShippingAddress(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "checkout id is required"},
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

	var req SetShippingAddressRequest
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

	// Authorization: verify the caller owns this checkout session.
	existing, err := h.service.GetCheckout(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
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
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: session})
}

// SetPaymentMethod handles PUT /api/v1/checkout/{id}/payment
// @Summary Set payment method
// @Description Sets the payment method on the checkout session.
// @Tags checkout
// @Accept json
// @Produce json
// @Param id path string true "Checkout session UUID"
// @Param X-User-ID header string true "Authenticated user UUID"
// @Param request body SetPaymentMethodRequest true "Payment method data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 422 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/checkout/{id}/payment [put]
func (h *CheckoutHandler) SetPaymentMethod(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "checkout id is required"},
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

	var req SetPaymentMethodRequest
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

	// Authorization: verify the caller owns this checkout session.
	existing, err := h.service.GetCheckout(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}
	if !authorizeSession(w, r, existing) {
		return
	}

	session, err := h.service.SetPaymentMethod(r.Context(), id, req.PaymentMethod)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: session})
}

// ProcessCheckout handles POST /api/v1/checkout/{id}/process
// @Summary Process checkout
// @Description Runs the checkout saga: reserves inventory, creates order, and initiates payment.
// @Tags checkout
// @Produce json
// @Param id path string true "Checkout session UUID"
// @Param X-User-ID header string true "Authenticated user UUID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 503 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/checkout/{id}/process [post]
func (h *CheckoutHandler) ProcessCheckout(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "checkout id is required"},
		})
		return
	}

	// Authorization: verify the caller owns this checkout session.
	existing, err := h.service.GetCheckout(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}
	if !authorizeSession(w, r, existing) {
		return
	}

	session, err := h.service.ProcessCheckout(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: session})
}

// CancelCheckout handles POST /api/v1/checkout/{id}/cancel
// @Summary Cancel checkout session
// @Description Cancels an active checkout session. Terminal sessions cannot be canceled.
// @Tags checkout
// @Produce json
// @Param id path string true "Checkout session UUID"
// @Param X-User-ID header string true "Authenticated user UUID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 422 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/checkout/{id}/cancel [post]
func (h *CheckoutHandler) CancelCheckout(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "checkout id is required"},
		})
		return
	}

	// Authorization: verify the caller owns this checkout session.
	existing, err := h.service.GetCheckout(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}
	if !authorizeSession(w, r, existing) {
		return
	}

	session, err := h.service.CancelCheckout(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: session})
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
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "X-User-ID header is required"},
		})
		return false
	}
	if session.UserID != userID {
		httputil.WriteJSON(w, http.StatusForbidden, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "FORBIDDEN", Message: "you do not have access to this checkout session"},
		})
		return false
	}
	return true
}
