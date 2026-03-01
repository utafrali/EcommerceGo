package httpclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
)

// DownstreamErrorResponse mirrors the httputil.ErrorResponse structure returned
// by EcommerceGo services. It is used to parse structured error bodies from
// downstream HTTP calls.
type DownstreamErrorResponse struct {
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// ParseResponseError reads the body of a non-2xx HTTP response and translates
// it into an appropriate AppError. If the response body matches the standard
// ErrorResponse format, the code and message are preserved. Otherwise a generic
// error is returned with the status code and raw body.
//
// The caller should only invoke this when resp.StatusCode indicates an error
// (i.e., not 2xx). The response body is fully consumed and closed.
func ParseResponseError(resp *http.Response, serviceName string) error {
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB limit
	if err != nil {
		return fmt.Errorf("%s returned status %d (failed to read body: %w)", serviceName, resp.StatusCode, err)
	}

	// Try to parse structured error response.
	var downstream DownstreamErrorResponse
	if json.Unmarshal(bodyBytes, &downstream) == nil && downstream.Error != nil {
		return mapDownstreamError(resp.StatusCode, downstream.Error.Code, downstream.Error.Message, serviceName)
	}

	// Fallback: unstructured error body.
	return fmt.Errorf("%s returned status %d: %s", serviceName, resp.StatusCode, string(bodyBytes))
}

// mapDownstreamError translates a downstream service's HTTP status code and
// error code into an AppError that preserves the error semantics.
func mapDownstreamError(status int, code, message, serviceName string) error {
	qualifiedMsg := fmt.Sprintf("%s: %s", serviceName, message)

	switch {
	case status == http.StatusNotFound:
		return apperrors.NotFound(serviceName, message)
	case status == http.StatusBadRequest:
		return apperrors.InvalidInput(qualifiedMsg)
	case status == http.StatusConflict:
		return apperrors.Conflict(qualifiedMsg)
	case status == http.StatusUnauthorized:
		return apperrors.Unauthorized(qualifiedMsg)
	case status == http.StatusForbidden:
		return apperrors.Forbidden(qualifiedMsg)
	case status == http.StatusGone:
		return apperrors.Gone(qualifiedMsg)
	case status == http.StatusUnprocessableEntity:
		return apperrors.PaymentFailed(qualifiedMsg)
	case status == http.StatusServiceUnavailable:
		return &apperrors.AppError{
			Code:    code,
			Message: qualifiedMsg,
			Status:  http.StatusServiceUnavailable,
			Err:     apperrors.ErrServiceUnavail,
		}
	case status >= 500:
		return fmt.Errorf("%s server error (%d/%s): %s", serviceName, status, code, message)
	default:
		return &apperrors.AppError{
			Code:    code,
			Message: qualifiedMsg,
			Status:  status,
		}
	}
}

// IsClientError returns true if the HTTP status code is a 4xx client error.
// This is useful for saga compensation logic: client errors (e.g., validation)
// typically should not trigger compensating actions since the request was invalid.
func IsClientError(status int) bool {
	return status >= 400 && status < 500
}
