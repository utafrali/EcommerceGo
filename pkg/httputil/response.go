package httputil

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/pkg/logger"
	"github.com/utafrali/EcommerceGo/pkg/validator"
)

// Response is the standard JSON response envelope used across all services.
type Response struct {
	Data  any            `json:"data,omitempty"`
	Error *ErrorResponse `json:"error,omitempty"`
}

// ErrorResponse represents an error in the standard response format.
type ErrorResponse struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
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
// and logs internal server errors. It prefers the request-scoped logger from
// context (set by the RequestLogger middleware) over the fallback logger.
func WriteError(w http.ResponseWriter, r *http.Request, err error, fallback *slog.Logger) {
	// Prefer the request-scoped logger (enriched with correlation_id, user_id,
	// trace_id, span_id) if the RequestLogger middleware has been mounted.
	l := logger.FromContext(r.Context())
	if l == slog.Default() {
		l = fallback
	}

	// Extract correlation ID from context to include in error responses.
	requestID := logger.CorrelationIDFromContext(r.Context())

	// Check if it's an AppError (custom error with code, message, and status)
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		WriteJSON(w, appErr.Status, Response{
			Error: &ErrorResponse{Code: appErr.Code, Message: appErr.Message, RequestID: requestID},
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
		l.ErrorContext(r.Context(), "internal error",
			slog.String("error", err.Error()),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
	}

	WriteJSON(w, status, Response{
		Error: &ErrorResponse{Code: code, Message: message, RequestID: requestID},
	})
}

// PaginatedResponse is a generic paginated list response envelope.
type PaginatedResponse[T any] struct {
	Data       []T  `json:"data"`
	TotalCount int  `json:"total_count"`
	Page       int  `json:"page"`
	PerPage    int  `json:"per_page"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
}

// NewPaginatedResponse constructs a PaginatedResponse from the given data, total
// count, page, and per-page values. It computes TotalPages and HasNext.
func NewPaginatedResponse[T any](data []T, totalCount, page, perPage int) PaginatedResponse[T] {
	totalPages := totalCount / perPage
	if totalCount%perPage > 0 {
		totalPages++
	}
	if data == nil {
		data = []T{}
	}
	return PaginatedResponse[T]{
		Data:       data,
		TotalCount: totalCount,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
	}
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

// ParseUUID validates that the given string is a valid UUID and returns it.
// If invalid, it writes a 400 Bad Request response with code INVALID_PARAMETER
// and returns uuid.Nil plus false, signaling the caller to return early.
func ParseUUID(w http.ResponseWriter, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(param)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, Response{
			Error: &ErrorResponse{
				Code:    "INVALID_PARAMETER",
				Message: "invalid UUID: " + param,
			},
		})
		return uuid.Nil, false
	}
	return id, true
}
