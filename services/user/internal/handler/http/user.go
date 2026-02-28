package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/utafrali/EcommerceGo/pkg/httputil"
	"github.com/utafrali/EcommerceGo/pkg/middleware"
	"github.com/utafrali/EcommerceGo/pkg/validator"
	"github.com/utafrali/EcommerceGo/services/user/internal/service"
)

// UserHandler handles HTTP requests for user profile and address endpoints.
type UserHandler struct {
	service *service.UserService
	logger  *slog.Logger
}

// NewUserHandler creates a new user HTTP handler.
func NewUserHandler(svc *service.UserService, logger *slog.Logger) *UserHandler {
	return &UserHandler{service: svc, logger: logger}
}

// --- Request DTOs ---

// UpdateProfileRequest is the JSON request body for updating user profile.
type UpdateProfileRequest struct {
	FirstName *string `json:"first_name" validate:"omitempty,min=1,max=100"`
	LastName  *string `json:"last_name" validate:"omitempty,min=1,max=100"`
	Phone     *string `json:"phone" validate:"omitempty,max=20"`
}

// CreateAddressRequest is the JSON request body for creating an address.
type CreateAddressRequest struct {
	Label        string `json:"label" validate:"omitempty,max=50"`
	FirstName    string `json:"first_name" validate:"required,min=1,max=100"`
	LastName     string `json:"last_name" validate:"required,min=1,max=100"`
	AddressLine1 string `json:"address_line1" validate:"required,min=1,max=500"`
	AddressLine2 string `json:"address_line2" validate:"omitempty,max=500"`
	City         string `json:"city" validate:"required,min=1,max=100"`
	State        string `json:"state" validate:"omitempty,max=100"`
	PostalCode   string `json:"postal_code" validate:"required,min=1,max=20"`
	CountryCode  string `json:"country_code" validate:"required,len=2"`
	Phone        string `json:"phone" validate:"omitempty,max=20"`
	IsDefault    bool   `json:"is_default"`
}

// UpdateAddressRequest is the JSON request body for updating an address.
type UpdateAddressRequest struct {
	Label        *string `json:"label" validate:"omitempty,max=50"`
	FirstName    *string `json:"first_name" validate:"omitempty,min=1,max=100"`
	LastName     *string `json:"last_name" validate:"omitempty,min=1,max=100"`
	AddressLine1 *string `json:"address_line1" validate:"omitempty,min=1,max=500"`
	AddressLine2 *string `json:"address_line2" validate:"omitempty,max=500"`
	City         *string `json:"city" validate:"omitempty,min=1,max=100"`
	State        *string `json:"state" validate:"omitempty,max=100"`
	PostalCode   *string `json:"postal_code" validate:"omitempty,min=1,max=20"`
	CountryCode  *string `json:"country_code" validate:"omitempty,len=2"`
	Phone        *string `json:"phone" validate:"omitempty,max=20"`
}

// --- Handlers ---

// GetProfile handles GET /api/v1/users/me
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteJSON(w, http.StatusUnauthorized, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	user, err := h.service.GetProfile(r.Context(), userID)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: user})
}

// UpdateProfile handles PUT /api/v1/users/me
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteJSON(w, http.StatusUnauthorized, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

	var req UpdateProfileRequest
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

	input := service.UpdateProfileInput{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
	}

	user, err := h.service.UpdateProfile(r.Context(), userID, input)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: user})
}

// ListAddresses handles GET /api/v1/users/me/addresses
func (h *UserHandler) ListAddresses(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteJSON(w, http.StatusUnauthorized, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	addresses, err := h.service.ListAddresses(r.Context(), userID)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: addresses})
}

// CreateAddress handles POST /api/v1/users/me/addresses
func (h *UserHandler) CreateAddress(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteJSON(w, http.StatusUnauthorized, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

	var req CreateAddressRequest
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

	input := &service.CreateAddressInput{
		Label:        req.Label,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		AddressLine1: req.AddressLine1,
		AddressLine2: req.AddressLine2,
		City:         req.City,
		State:        req.State,
		PostalCode:   req.PostalCode,
		CountryCode:  req.CountryCode,
		Phone:        req.Phone,
		IsDefault:    req.IsDefault,
	}

	address, err := h.service.CreateAddress(r.Context(), userID, input)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, httputil.Response{Data: address})
}

// UpdateAddress handles PUT /api/v1/users/me/addresses/{id}
func (h *UserHandler) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteJSON(w, http.StatusUnauthorized, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	addressID := chi.URLParam(r, "id")
	if addressID == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "address id is required"},
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

	var req UpdateAddressRequest
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

	input := &service.UpdateAddressInput{
		Label:        req.Label,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		AddressLine1: req.AddressLine1,
		AddressLine2: req.AddressLine2,
		City:         req.City,
		State:        req.State,
		PostalCode:   req.PostalCode,
		CountryCode:  req.CountryCode,
		Phone:        req.Phone,
	}

	address, err := h.service.UpdateAddress(r.Context(), userID, addressID, input)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: address})
}

// DeleteAddress handles DELETE /api/v1/users/me/addresses/{id}
func (h *UserHandler) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteJSON(w, http.StatusUnauthorized, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	addressID := chi.URLParam(r, "id")
	if addressID == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "address id is required"},
		})
		return
	}

	if err := h.service.DeleteAddress(r.Context(), userID, addressID); err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: map[string]string{"id": addressID, "status": "deleted"}})
}
