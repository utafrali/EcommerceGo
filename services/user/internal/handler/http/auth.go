package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/utafrali/EcommerceGo/pkg/validator"
	"github.com/utafrali/EcommerceGo/services/user/internal/service"
)

// AuthHandler handles HTTP requests for auth endpoints.
type AuthHandler struct {
	service *service.UserService
	logger  *slog.Logger
}

// NewAuthHandler creates a new auth HTTP handler.
func NewAuthHandler(svc *service.UserService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{service: svc, logger: logger}
}

// --- Request DTOs ---

// RegisterRequest is the JSON request body for user registration.
type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required,min=1,max=100"`
	LastName  string `json:"last_name" validate:"required,min=1,max=100"`
}

// LoginRequest is the JSON request body for user login.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RefreshTokenRequest is the JSON request body for token refresh.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ForgotPasswordRequest is the JSON request body for forgot password.
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordRequest is the JSON request body for password reset.
type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// --- Response types ---

// AuthResponse wraps user data with tokens.
type AuthResponse struct {
	User   any `json:"user"`
	Tokens any `json:"tokens"`
}

// --- Handlers ---

// Register handles POST /api/v1/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

	var req RegisterRequest
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

	input := service.RegisterInput{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}

	user, tokens, err := h.service.Register(r.Context(), input)
	if err != nil {
		writeAppError(w, r, err, h.logger)
		return
	}

	writeJSON(w, http.StatusCreated, response{
		Data: AuthResponse{
			User:   user,
			Tokens: tokens,
		},
	})
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

	var req LoginRequest
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

	input := service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	}

	user, tokens, err := h.service.Login(r.Context(), input)
	if err != nil {
		writeAppError(w, r, err, h.logger)
		return
	}

	writeJSON(w, http.StatusOK, response{
		Data: AuthResponse{
			User:   user,
			Tokens: tokens,
		},
	})
}

// RefreshToken handles POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

	var req RefreshTokenRequest
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

	tokens, err := h.service.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		writeAppError(w, r, err, h.logger)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: tokens})
}

// ForgotPassword handles POST /api/v1/auth/forgot-password
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

	var req ForgotPasswordRequest
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

	if err := h.service.ForgotPassword(r.Context(), req.Email); err != nil {
		writeAppError(w, r, err, h.logger)
		return
	}

	writeJSON(w, http.StatusOK, response{
		Data: map[string]string{"message": "if the email exists, a password reset link has been sent"},
	})
}

// ResetPassword handles POST /api/v1/auth/reset-password
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

	var req ResetPasswordRequest
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

	if err := h.service.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		writeAppError(w, r, err, h.logger)
		return
	}

	writeJSON(w, http.StatusOK, response{
		Data: map[string]string{"message": "password has been reset successfully"},
	})
}
