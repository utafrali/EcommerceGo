package httputil

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/pkg/validator"
)

// Response is the standard JSON response envelope used across all services.
type Response struct {
	Data  any            `json:"data,omitempty"`
	Error *ErrorResponse `json:"error,omitempty"`
}

// ErrorResponse represents an error in the standard response format.
type ErrorResponse struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// WriteJSON writes a JSON response with the given status code.
// If encoding fails, the error is logged but headers are already sent so nothing can be done.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Headers are already sent; nothing meaningful can be done if encoding fails.
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes a standardized error response based on the error type.
// It handles AppError, standard errors (ErrNotFound, ErrAlreadyExists, ErrInvalidInput),
// and logs internal server errors.
func WriteError(w http.ResponseWriter, r *http.Request, err error, logger *slog.Logger) {
	// Check if it's an AppError (custom error with code, message, and status)
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		WriteJSON(w, appErr.Status, Response{
			Error: &ErrorResponse{Code: appErr.Code, Message: appErr.Message},
		})
		return
	}

	// Determine status code and error details
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

	// Log internal errors
	if status == http.StatusInternalServerError {
		logger.ErrorContext(r.Context(), "internal error",
			slog.String("error", err.Error()),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
	}

	WriteJSON(w, status, Response{
		Error: &ErrorResponse{Code: code, Message: message},
	})
}

// WriteValidationError writes a standardized validation error response.
// It handles ValidationError from the validator package and returns field-level errors.
func WriteValidationError(w http.ResponseWriter, err error) {
	var valErr *validator.ValidationError
	if errors.As(err, &valErr) {
		WriteJSON(w, http.StatusBadRequest, Response{
			Error: &ErrorResponse{
				Code:    "VALIDATION_ERROR",
				Message: "request validation failed",
				Fields:  valErr.Fields(),
			},
		})
		return
	}

	WriteJSON(w, http.StatusBadRequest, Response{
		Error: &ErrorResponse{Code: "INVALID_INPUT", Message: err.Error()},
	})
}
