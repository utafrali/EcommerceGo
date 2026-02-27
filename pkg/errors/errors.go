package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Standard sentinel errors for common cases.
var (
	ErrNotFound       = errors.New("resource not found")
	ErrAlreadyExists  = errors.New("resource already exists")
	ErrInvalidInput   = errors.New("invalid input")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
	ErrInternal       = errors.New("internal error")
	ErrConflict       = errors.New("conflict")
	ErrServiceUnavail = errors.New("service unavailable")
	ErrPaymentFailed  = errors.New("payment failed")
)

// AppError represents a structured application error with HTTP status mapping.
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// NotFound creates a 404 error.
func NotFound(resource, id string) *AppError {
	return &AppError{
		Code:    "NOT_FOUND",
		Message: fmt.Sprintf("%s with id %s not found", resource, id),
		Status:  http.StatusNotFound,
		Err:     ErrNotFound,
	}
}

// AlreadyExists creates a 409 error.
func AlreadyExists(resource, field, value string) *AppError {
	return &AppError{
		Code:    "ALREADY_EXISTS",
		Message: fmt.Sprintf("%s with %s %q already exists", resource, field, value),
		Status:  http.StatusConflict,
		Err:     ErrAlreadyExists,
	}
}

// InvalidInput creates a 400 error.
func InvalidInput(message string) *AppError {
	return &AppError{
		Code:    "INVALID_INPUT",
		Message: message,
		Status:  http.StatusBadRequest,
		Err:     ErrInvalidInput,
	}
}

// Unauthorized creates a 401 error.
func Unauthorized(message string) *AppError {
	return &AppError{
		Code:    "UNAUTHORIZED",
		Message: message,
		Status:  http.StatusUnauthorized,
		Err:     ErrUnauthorized,
	}
}

// Forbidden creates a 403 error.
func Forbidden(message string) *AppError {
	return &AppError{
		Code:    "FORBIDDEN",
		Message: message,
		Status:  http.StatusForbidden,
		Err:     ErrForbidden,
	}
}

// Internal creates a 500 error.
func Internal(err error) *AppError {
	return &AppError{
		Code:    "INTERNAL_ERROR",
		Message: "an internal error occurred",
		Status:  http.StatusInternalServerError,
		Err:     err,
	}
}

// PaymentFailed creates a 422 error for a payment charge failure.
func PaymentFailed(message string) *AppError {
	return &AppError{
		Code:    "PAYMENT_FAILED",
		Message: message,
		Status:  http.StatusUnprocessableEntity,
		Err:     ErrPaymentFailed,
	}
}

// Wrap wraps an error with additional context.
func Wrap(err error, message string) error {
	return fmt.Errorf("%s: %w", message, err)
}

// HTTPStatus returns the HTTP status code for the given error.
func HTTPStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Status
	}

	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrAlreadyExists), errors.Is(err, ErrConflict):
		return http.StatusConflict
	case errors.Is(err, ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrPaymentFailed):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}
