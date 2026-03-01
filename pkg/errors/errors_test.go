package errors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Sentinel error identity ---

func TestSentinelErrors_AreDistinct(t *testing.T) {
	sentinels := []error{
		ErrNotFound, ErrAlreadyExists, ErrInvalidInput, ErrUnauthorized,
		ErrForbidden, ErrInternal, ErrConflict, ErrServiceUnavail,
		ErrPaymentFailed, ErrGone,
	}

	for i := 0; i < len(sentinels); i++ {
		for j := i + 1; j < len(sentinels); j++ {
			assert.NotEqual(t, sentinels[i], sentinels[j],
				"sentinels %d and %d should be distinct", i, j)
		}
	}
}

// --- AppError behavior ---

func TestAppError_ErrorString_WithWrappedError(t *testing.T) {
	inner := fmt.Errorf("db connection lost")
	appErr := &AppError{Code: "INTERNAL_ERROR", Message: "something broke", Err: inner}
	assert.Contains(t, appErr.Error(), "INTERNAL_ERROR")
	assert.Contains(t, appErr.Error(), "something broke")
	assert.Contains(t, appErr.Error(), "db connection lost")
}

func TestAppError_ErrorString_WithoutWrappedError(t *testing.T) {
	appErr := &AppError{Code: "NOT_FOUND", Message: "user not found"}
	assert.Equal(t, "NOT_FOUND: user not found", appErr.Error())
}

func TestAppError_Unwrap(t *testing.T) {
	inner := ErrNotFound
	appErr := &AppError{Code: "NOT_FOUND", Message: "nope", Err: inner}
	assert.True(t, errors.Is(appErr, ErrNotFound))
}

func TestAppError_Unwrap_Nil(t *testing.T) {
	appErr := &AppError{Code: "TEST", Message: "test"}
	assert.Nil(t, appErr.Unwrap())
}

// --- Constructor functions ---

func TestNotFound(t *testing.T) {
	err := NotFound("product", "abc-123")
	require.NotNil(t, err)
	assert.Equal(t, "NOT_FOUND", err.Code)
	assert.Contains(t, err.Message, "product")
	assert.Contains(t, err.Message, "abc-123")
	assert.Equal(t, http.StatusNotFound, err.Status)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestAlreadyExists(t *testing.T) {
	err := AlreadyExists("user", "email", "a@b.com")
	require.NotNil(t, err)
	assert.Equal(t, "ALREADY_EXISTS", err.Code)
	assert.Contains(t, err.Message, "user")
	assert.Contains(t, err.Message, "email")
	assert.Contains(t, err.Message, "a@b.com")
	assert.Equal(t, http.StatusConflict, err.Status)
	assert.True(t, errors.Is(err, ErrAlreadyExists))
}

func TestInvalidInput(t *testing.T) {
	err := InvalidInput("name is required")
	require.NotNil(t, err)
	assert.Equal(t, "INVALID_INPUT", err.Code)
	assert.Equal(t, "name is required", err.Message)
	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.True(t, errors.Is(err, ErrInvalidInput))
}

func TestUnauthorized(t *testing.T) {
	err := Unauthorized("invalid token")
	require.NotNil(t, err)
	assert.Equal(t, "UNAUTHORIZED", err.Code)
	assert.Equal(t, http.StatusUnauthorized, err.Status)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestForbidden(t *testing.T) {
	err := Forbidden("not allowed")
	require.NotNil(t, err)
	assert.Equal(t, "FORBIDDEN", err.Code)
	assert.Equal(t, http.StatusForbidden, err.Status)
	assert.True(t, errors.Is(err, ErrForbidden))
}

func TestInternal(t *testing.T) {
	inner := fmt.Errorf("segfault")
	err := Internal(inner)
	require.NotNil(t, err)
	assert.Equal(t, "INTERNAL_ERROR", err.Code)
	assert.Equal(t, http.StatusInternalServerError, err.Status)
	assert.Contains(t, err.Error(), "segfault")
}

func TestPaymentFailed(t *testing.T) {
	err := PaymentFailed("card declined")
	require.NotNil(t, err)
	assert.Equal(t, "PAYMENT_FAILED", err.Code)
	assert.Equal(t, http.StatusUnprocessableEntity, err.Status)
	assert.True(t, errors.Is(err, ErrPaymentFailed))
}

func TestGone(t *testing.T) {
	err := Gone("session expired")
	require.NotNil(t, err)
	assert.Equal(t, "GONE", err.Code)
	assert.Equal(t, http.StatusGone, err.Status)
	assert.True(t, errors.Is(err, ErrGone))
}

func TestConflict(t *testing.T) {
	err := Conflict("version mismatch")
	require.NotNil(t, err)
	assert.Equal(t, "CONFLICT", err.Code)
	assert.Equal(t, http.StatusConflict, err.Status)
	assert.True(t, errors.Is(err, ErrConflict))
}

// --- Wrap ---

func TestWrap(t *testing.T) {
	inner := ErrNotFound
	wrapped := Wrap(inner, "get user")
	assert.Contains(t, wrapped.Error(), "get user")
	assert.True(t, errors.Is(wrapped, ErrNotFound))
}

// --- HTTPStatus ---

func TestHTTPStatus_AppError(t *testing.T) {
	appErr := NotFound("item", "1")
	assert.Equal(t, http.StatusNotFound, HTTPStatus(appErr))
}

func TestHTTPStatus_SentinelErrors(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{ErrNotFound, http.StatusNotFound},
		{ErrAlreadyExists, http.StatusConflict},
		{ErrConflict, http.StatusConflict},
		{ErrInvalidInput, http.StatusBadRequest},
		{ErrUnauthorized, http.StatusUnauthorized},
		{ErrForbidden, http.StatusForbidden},
		{ErrPaymentFailed, http.StatusUnprocessableEntity},
		{ErrGone, http.StatusGone},
	}

	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			assert.Equal(t, tt.status, HTTPStatus(tt.err))
		})
	}
}

func TestHTTPStatus_WrappedSentinel(t *testing.T) {
	wrapped := fmt.Errorf("outer: %w", ErrNotFound)
	assert.Equal(t, http.StatusNotFound, HTTPStatus(wrapped))
}

func TestHTTPStatus_UnknownError(t *testing.T) {
	assert.Equal(t, http.StatusInternalServerError, HTTPStatus(fmt.Errorf("unknown")))
}
