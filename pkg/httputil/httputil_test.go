package httputil

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/pkg/logger"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// --- WriteJSON ---

func TestWriteJSON_SetsContentType(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusOK, Response{Data: "hello"})

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWriteJSON_EncodesData(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusOK, Response{Data: map[string]string{"key": "value"}})

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotNil(t, resp.Data)
}

func TestWriteJSON_ErrorPayload(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusBadRequest, Response{
		Error: &ErrorResponse{Code: "INVALID", Message: "bad input"},
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID", resp.Error.Code)
	assert.Equal(t, "bad input", resp.Error.Message)
}

func TestWriteJSON_StatusCodes(t *testing.T) {
	codes := []int{http.StatusOK, http.StatusCreated, http.StatusNotFound, http.StatusTeapot}
	for _, code := range codes {
		rec := httptest.NewRecorder()
		WriteJSON(rec, code, Response{})
		assert.Equal(t, code, rec.Code)
	}
}

// --- WriteError ---

func TestWriteError_AppError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	appErr := apperrors.NotFound("product", "abc-123")
	WriteError(rec, req, appErr, testLogger())

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
}

func TestWriteError_SentinelNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	WriteError(rec, req, apperrors.ErrNotFound, testLogger())

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
}

func TestWriteError_SentinelAlreadyExists(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)

	WriteError(rec, req, apperrors.ErrAlreadyExists, testLogger())

	assert.Equal(t, http.StatusConflict, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ALREADY_EXISTS", resp.Error.Code)
}

func TestWriteError_SentinelInvalidInput(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)

	WriteError(rec, req, apperrors.ErrInvalidInput, testLogger())

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
}

func TestWriteError_UnknownError_Returns500(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	WriteError(rec, req, fmt.Errorf("something unexpected"), testLogger())

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "INTERNAL_ERROR", resp.Error.Code)
}

// --- WriteValidationError ---

func TestWriteValidationError_NonValidationError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteValidationError(rec, fmt.Errorf("not a validation error"))

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
}

// --- Response struct ---

func TestResponse_NilError_OmitsErrorField(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusOK, Response{Data: "ok"})

	var raw map[string]json.RawMessage
	err := json.NewDecoder(rec.Body).Decode(&raw)
	require.NoError(t, err)
	_, hasError := raw["error"]
	assert.False(t, hasError, "error field should be omitted when nil")
}

func TestResponse_NilData_OmitsDataField(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusBadRequest, Response{
		Error: &ErrorResponse{Code: "ERR", Message: "msg"},
	})

	var raw map[string]json.RawMessage
	err := json.NewDecoder(rec.Body).Decode(&raw)
	require.NoError(t, err)
	_, hasData := raw["data"]
	assert.False(t, hasData, "data field should be omitted when nil")
}

// --- PaginatedResponse ---

func TestNewPaginatedResponse_ComputesTotalPages(t *testing.T) {
	resp := NewPaginatedResponse([]string{"a", "b"}, 25, 1, 10)
	assert.Equal(t, 3, resp.TotalPages)
	assert.True(t, resp.HasNext)
	assert.Equal(t, 2, len(resp.Data))
}

func TestNewPaginatedResponse_LastPage(t *testing.T) {
	resp := NewPaginatedResponse([]string{"x"}, 21, 3, 10)
	assert.Equal(t, 3, resp.TotalPages)
	assert.False(t, resp.HasNext)
}

func TestNewPaginatedResponse_ExactDivision(t *testing.T) {
	resp := NewPaginatedResponse([]int{1, 2, 3}, 30, 2, 10)
	assert.Equal(t, 3, resp.TotalPages)
	assert.True(t, resp.HasNext)
}

func TestNewPaginatedResponse_NilDataBecomesEmptySlice(t *testing.T) {
	resp := NewPaginatedResponse[string](nil, 0, 1, 20)
	assert.NotNil(t, resp.Data)
	assert.Equal(t, 0, len(resp.Data))
	assert.Equal(t, 0, resp.TotalPages)
	assert.False(t, resp.HasNext)
}

func TestNewPaginatedResponse_JSONSerialization(t *testing.T) {
	resp := NewPaginatedResponse([]string{"hello"}, 1, 1, 10)
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusOK, resp)

	var out map[string]json.RawMessage
	err := json.NewDecoder(rec.Body).Decode(&out)
	require.NoError(t, err)

	assert.Contains(t, string(out["data"]), "hello")
	assert.Contains(t, string(out["total_count"]), "1")
	assert.Contains(t, string(out["page"]), "1")
	assert.Contains(t, string(out["per_page"]), "10")
}

// --- ParseUUID ---

func TestParseUUID_ValidUUID(t *testing.T) {
	rec := httptest.NewRecorder()
	id, ok := ParseUUID(rec, "550e8400-e29b-41d4-a716-446655440000")
	assert.True(t, ok)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", id.String())
	assert.Equal(t, http.StatusOK, rec.Code) // no response written
}

func TestParseUUID_InvalidUUID_Returns400(t *testing.T) {
	rec := httptest.NewRecorder()
	id, ok := ParseUUID(rec, "not-a-uuid")
	assert.False(t, ok)
	assert.Equal(t, "00000000-0000-0000-0000-000000000000", id.String())
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "not-a-uuid")
}

func TestParseUUID_EmptyString_Returns400(t *testing.T) {
	rec := httptest.NewRecorder()
	_, ok := ParseUUID(rec, "")
	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

func TestParseUUID_ShortString_Returns400(t *testing.T) {
	rec := httptest.NewRecorder()
	_, ok := ParseUUID(rec, "abc123")
	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestParseUUID_UppercaseUUID(t *testing.T) {
	rec := httptest.NewRecorder()
	id, ok := ParseUUID(rec, "550E8400-E29B-41D4-A716-446655440000")
	assert.True(t, ok)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", id.String())
}

// --- WriteError RequestID (correlation_id in error responses) ---

func TestWriteError_IncludesRequestID(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx := logger.WithCorrelationID(context.Background(), "corr-123")
	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)

	WriteError(rec, req, apperrors.ErrNotFound, testLogger())

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "corr-123", resp.Error.RequestID)
}

func TestWriteError_NoCorrelationID_OmitsRequestID(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	WriteError(rec, req, apperrors.ErrNotFound, testLogger())

	assert.Equal(t, http.StatusNotFound, rec.Code)

	// Verify the decoded RequestID is empty.
	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.NotNil(t, resp.Error)
	assert.Empty(t, resp.Error.RequestID)

	// Also verify "request_id" key is not present in the raw JSON (omitempty).
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	WriteError(rec2, req2, apperrors.ErrNotFound, testLogger())

	var raw map[string]json.RawMessage
	err = json.NewDecoder(rec2.Body).Decode(&raw)
	require.NoError(t, err)

	var errObj map[string]json.RawMessage
	err = json.Unmarshal(raw["error"], &errObj)
	require.NoError(t, err)
	_, hasRequestID := errObj["request_id"]
	assert.False(t, hasRequestID, "request_id should be omitted from JSON when correlation_id is not in context")
}

func TestWriteError_AppError_IncludesRequestID(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx := logger.WithCorrelationID(context.Background(), "corr-456")
	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)

	appErr := apperrors.NotFound("product", "xyz-789")
	WriteError(rec, req, appErr, testLogger())

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	assert.Equal(t, "corr-456", resp.Error.RequestID)
}

func TestErrorResponse_RequestID_JSONSerialization(t *testing.T) {
	// With RequestID set, it should appear in JSON output.
	withID := ErrorResponse{Code: "ERR", Message: "msg", RequestID: "req-abc"}
	data, err := json.Marshal(withID)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)
	_, hasRequestID := raw["request_id"]
	assert.True(t, hasRequestID, "request_id should be present in JSON when set")
	assert.Contains(t, string(raw["request_id"]), "req-abc")

	// Without RequestID, it should be omitted (omitempty).
	withoutID := ErrorResponse{Code: "ERR", Message: "msg"}
	data2, err := json.Marshal(withoutID)
	require.NoError(t, err)

	var raw2 map[string]json.RawMessage
	err = json.Unmarshal(data2, &raw2)
	require.NoError(t, err)
	_, hasRequestID2 := raw2["request_id"]
	assert.False(t, hasRequestID2, "request_id should be omitted from JSON when empty")
}
