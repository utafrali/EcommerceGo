package httpclient

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeResponse creates an *http.Response with the given status code and body string.
func makeResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// structuredError builds a standard JSON error body.
func structuredError(code, message string) string {
	return `{"error":{"code":"` + code + `","message":"` + message + `"}}`
}

func TestParseResponseError_StructuredError_NotFound(t *testing.T) {
	resp := makeResponse(http.StatusNotFound, structuredError("NOT_FOUND", "product not found"))
	err := ParseResponseError(resp, "inventory")
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.True(t, errors.As(err, &appErr), "expected AppError, got %T: %v", err, err)
	assert.Equal(t, http.StatusNotFound, appErr.Status)
	assert.Equal(t, "NOT_FOUND", appErr.Code)
	assert.True(t, errors.Is(err, apperrors.ErrNotFound))
}

func TestParseResponseError_StructuredError_BadRequest(t *testing.T) {
	resp := makeResponse(http.StatusBadRequest, structuredError("INVALID_INPUT", "missing field name"))
	err := ParseResponseError(resp, "user-service")
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, http.StatusBadRequest, appErr.Status)
	assert.True(t, errors.Is(err, apperrors.ErrInvalidInput))
	assert.Contains(t, appErr.Message, "user-service")
}

func TestParseResponseError_StructuredError_Conflict(t *testing.T) {
	resp := makeResponse(http.StatusConflict, structuredError("CONFLICT", "version mismatch"))
	err := ParseResponseError(resp, "cart-service")
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, http.StatusConflict, appErr.Status)
	assert.True(t, errors.Is(err, apperrors.ErrConflict))
	assert.Contains(t, appErr.Message, "cart-service")
}

func TestParseResponseError_StructuredError_Unauthorized(t *testing.T) {
	resp := makeResponse(http.StatusUnauthorized, structuredError("UNAUTHORIZED", "invalid token"))
	err := ParseResponseError(resp, "auth-service")
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, http.StatusUnauthorized, appErr.Status)
	assert.True(t, errors.Is(err, apperrors.ErrUnauthorized))
	assert.Contains(t, appErr.Message, "auth-service")
}

func TestParseResponseError_StructuredError_Forbidden(t *testing.T) {
	resp := makeResponse(http.StatusForbidden, structuredError("FORBIDDEN", "insufficient permissions"))
	err := ParseResponseError(resp, "admin-service")
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, http.StatusForbidden, appErr.Status)
	assert.True(t, errors.Is(err, apperrors.ErrForbidden))
	assert.Contains(t, appErr.Message, "admin-service")
}

func TestParseResponseError_StructuredError_Gone(t *testing.T) {
	resp := makeResponse(http.StatusGone, structuredError("GONE", "reservation expired"))
	err := ParseResponseError(resp, "inventory-service")
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, http.StatusGone, appErr.Status)
	assert.True(t, errors.Is(err, apperrors.ErrGone))
	assert.Contains(t, appErr.Message, "inventory-service")
}

func TestParseResponseError_StructuredError_PaymentFailed(t *testing.T) {
	resp := makeResponse(http.StatusUnprocessableEntity, structuredError("PAYMENT_FAILED", "card declined"))
	err := ParseResponseError(resp, "payment-service")
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, http.StatusUnprocessableEntity, appErr.Status)
	assert.True(t, errors.Is(err, apperrors.ErrPaymentFailed))
	assert.Contains(t, appErr.Message, "payment-service")
}

func TestParseResponseError_StructuredError_ServiceUnavailable(t *testing.T) {
	resp := makeResponse(http.StatusServiceUnavailable, structuredError("SERVICE_UNAVAILABLE", "overloaded"))
	err := ParseResponseError(resp, "gateway")
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, http.StatusServiceUnavailable, appErr.Status)
	assert.Equal(t, "SERVICE_UNAVAILABLE", appErr.Code)
	assert.True(t, errors.Is(err, apperrors.ErrServiceUnavail))
	assert.Contains(t, appErr.Message, "gateway")
}

func TestParseResponseError_StructuredError_ServerError(t *testing.T) {
	resp := makeResponse(http.StatusInternalServerError, structuredError("INTERNAL_ERROR", "something went wrong"))
	err := ParseResponseError(resp, "order-service")
	require.Error(t, err)

	// 500 server errors produce a generic error (not AppError).
	assert.Contains(t, err.Error(), "order-service")
	assert.Contains(t, err.Error(), "500")
	assert.Contains(t, err.Error(), "something went wrong")
}

func TestParseResponseError_StructuredError_502(t *testing.T) {
	resp := makeResponse(http.StatusBadGateway, structuredError("BAD_GATEWAY", "upstream error"))
	err := ParseResponseError(resp, "gateway")
	require.Error(t, err)

	// 502 is >= 500, should produce a generic error string.
	assert.Contains(t, err.Error(), "gateway")
	assert.Contains(t, err.Error(), "502")
}

func TestParseResponseError_UnstructuredBody(t *testing.T) {
	resp := makeResponse(http.StatusBadGateway, "Bad Gateway: upstream connection refused")
	err := ParseResponseError(resp, "api-gateway")
	require.Error(t, err)

	assert.Contains(t, err.Error(), "api-gateway")
	assert.Contains(t, err.Error(), "502")
	assert.Contains(t, err.Error(), "Bad Gateway: upstream connection refused")
}

func TestParseResponseError_EmptyBody(t *testing.T) {
	resp := makeResponse(http.StatusInternalServerError, "")
	err := ParseResponseError(resp, "some-service")
	require.Error(t, err)

	assert.Contains(t, err.Error(), "some-service")
	assert.Contains(t, err.Error(), "500")
}

func TestParseResponseError_HTMLBody(t *testing.T) {
	resp := makeResponse(http.StatusBadGateway, "<html><body><h1>502 Bad Gateway</h1></body></html>")
	err := ParseResponseError(resp, "nginx")
	require.Error(t, err)

	assert.Contains(t, err.Error(), "nginx")
	assert.Contains(t, err.Error(), "502")
}

func TestParseResponseError_StructuredButNullError(t *testing.T) {
	// JSON body with error: null should fall through to the unstructured path.
	resp := makeResponse(http.StatusBadRequest, `{"error":null}`)
	err := ParseResponseError(resp, "svc")
	require.Error(t, err)

	// Should be a plain error (not AppError), since downstream.Error is nil.
	assert.Contains(t, err.Error(), "svc")
	assert.Contains(t, err.Error(), "400")
}

func TestParseResponseError_DefaultStatusCode(t *testing.T) {
	// A 4xx status not specifically handled (e.g. 429 Too Many Requests) should
	// produce a generic AppError with the original status preserved.
	resp := makeResponse(http.StatusTooManyRequests, structuredError("RATE_LIMITED", "slow down"))
	err := ParseResponseError(resp, "gateway")
	require.Error(t, err)

	var appErr *apperrors.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, http.StatusTooManyRequests, appErr.Status)
	assert.Equal(t, "RATE_LIMITED", appErr.Code)
	assert.Contains(t, appErr.Message, "gateway")
}

// --- IsClientError tests ---

func TestIsClientError_4xx(t *testing.T) {
	clientStatuses := []int{400, 401, 403, 404, 409, 410, 422, 429, 499}
	for _, status := range clientStatuses {
		assert.True(t, IsClientError(status), "status %d should be a client error", status)
	}
}

func TestIsClientError_5xx(t *testing.T) {
	serverStatuses := []int{500, 501, 502, 503, 504}
	for _, status := range serverStatuses {
		assert.False(t, IsClientError(status), "status %d should NOT be a client error", status)
	}
}

func TestIsClientError_2xx(t *testing.T) {
	successStatuses := []int{200, 201, 204, 301, 302}
	for _, status := range successStatuses {
		assert.False(t, IsClientError(status), "status %d should NOT be a client error", status)
	}
}

func TestIsClientError_Boundary(t *testing.T) {
	assert.False(t, IsClientError(399), "399 should not be a client error")
	assert.True(t, IsClientError(400), "400 should be a client error")
	assert.True(t, IsClientError(499), "499 should be a client error")
	assert.False(t, IsClientError(500), "500 should not be a client error")
}
