package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/pkg/middleware"
	"github.com/utafrali/EcommerceGo/pkg/validator"
	"github.com/utafrali/EcommerceGo/services/user/internal/service"
)

// UserHandler handles HTTP requests for user profile and address endpoints.
type UserHandler struct {
	service *service.UserService
}

// NewUserHandler creates a new user HTTP handler.
func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{service: svc}
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
		writeJSON(w, http.StatusUnauthorized, response{
			Error: &errorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	user, err := h.service.GetProfile(r.Context(), userID)
	if err != nil {
		writeAppError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: user})
}

// UpdateProfile handles PUT /api/v1/users/me
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, response{
			Error: &errorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		writeValidationError(w, err)
		return
	}

	input := service.UpdateProfileInput{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
	}

	user, err := h.service.UpdateProfile(r.Context(), userID, input)
	if err != nil {
		writeAppError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: user})
}

// ListAddresses handles GET /api/v1/users/me/addresses
func (h *UserHandler) ListAddresses(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, response{
			Error: &errorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	addresses, err := h.service.ListAddresses(r.Context(), userID)
	if err != nil {
		writeAppError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: addresses})
}

// CreateAddress handles POST /api/v1/users/me/addresses
func (h *UserHandler) CreateAddress(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, response{
			Error: &errorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	var req CreateAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		writeValidationError(w, err)
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
		writeAppError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, response{Data: address})
}

// UpdateAddress handles PUT /api/v1/users/me/addresses/{id}
func (h *UserHandler) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, response{
			Error: &errorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	addressID := chi.URLParam(r, "id")
	if addressID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "address id is required"},
		})
		return
	}

	var req UpdateAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		writeValidationError(w, err)
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
		writeAppError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: address})
}

// DeleteAddress handles DELETE /api/v1/users/me/addresses/{id}
func (h *UserHandler) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, response{
			Error: &errorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	addressID := chi.URLParam(r, "id")
	if addressID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "address id is required"},
		})
		return
	}

	if err := h.service.DeleteAddress(r.Context(), userID, addressID); err != nil {
		writeAppError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: map[string]string{"id": addressID, "status": "deleted"}})
}

// --- Shared response helpers ---

type response struct {
	Data  any            `json:"data,omitempty"`
	Error *errorResponse `json:"error,omitempty"`
}

type errorResponse struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Headers are already sent; nothing meaningful can be done if encoding fails.
	_ = json.NewEncoder(w).Encode(v)
}

func writeAppError(w http.ResponseWriter, _ *http.Request, err error) {
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
	case errors.Is(err, apperrors.ErrUnauthorized):
		code = "UNAUTHORIZED"
		message = err.Error()
		status = http.StatusUnauthorized
	}

	writeJSON(w, status, response{
		Error: &errorResponse{Code: code, Message: message},
	})
}

func writeValidationError(w http.ResponseWriter, err error) {
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
