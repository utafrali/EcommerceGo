package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLivenessHandler_AlwaysReturns200(t *testing.T) {
	h := NewHandler()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	h.LivenessHandler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, StatusUp, resp.Status)
	assert.False(t, resp.Timestamp.IsZero())
}

func TestReadinessHandler_AllHealthy(t *testing.T) {
	h := NewHandler()
	h.Register("db", func(ctx context.Context) error { return nil })
	h.Register("redis", func(ctx context.Context) error { return nil })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	h.ReadinessHandler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, StatusUp, resp.Status)
	assert.Equal(t, StatusUp, resp.Checks["db"].Status)
	assert.Equal(t, StatusUp, resp.Checks["redis"].Status)
}

func TestReadinessHandler_OneDown(t *testing.T) {
	h := NewHandler()
	h.Register("db", func(ctx context.Context) error { return nil })
	h.Register("redis", func(ctx context.Context) error { return fmt.Errorf("connection refused") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	h.ReadinessHandler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, StatusDown, resp.Status)
	assert.Equal(t, StatusUp, resp.Checks["db"].Status)
	assert.Equal(t, StatusDown, resp.Checks["redis"].Status)
	assert.Equal(t, "connection refused", resp.Checks["redis"].Error)
}

func TestReadinessHandler_NoCheckers(t *testing.T) {
	h := NewHandler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	h.ReadinessHandler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, StatusUp, resp.Status)
}

func TestNewHandler_RegisterOverwrite(t *testing.T) {
	h := NewHandler()
	h.Register("db", func(ctx context.Context) error { return fmt.Errorf("fail") })
	h.Register("db", func(ctx context.Context) error { return nil }) // overwrite

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	h.ReadinessHandler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, StatusUp, resp.Checks["db"].Status)
}

// --- Tests for critical/non-critical and degraded status ---

func TestReadinessHandler_NonCriticalDown_ReturnsDegraded200(t *testing.T) {
	h := NewHandler()
	h.RegisterCritical("postgres", func(ctx context.Context) error { return nil })
	h.RegisterNonCritical("kafka", func(ctx context.Context) error { return fmt.Errorf("broker unreachable") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	h.ReadinessHandler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, StatusDegraded, resp.Status)
	assert.Equal(t, StatusUp, resp.Checks["postgres"].Status)
	assert.True(t, resp.Checks["postgres"].Critical)
	assert.Equal(t, StatusDown, resp.Checks["kafka"].Status)
	assert.False(t, resp.Checks["kafka"].Critical)
	assert.Equal(t, "broker unreachable", resp.Checks["kafka"].Error)
}

func TestReadinessHandler_CriticalDown_Returns503(t *testing.T) {
	h := NewHandler()
	h.RegisterCritical("postgres", func(ctx context.Context) error { return fmt.Errorf("connection refused") })
	h.RegisterNonCritical("kafka", func(ctx context.Context) error { return nil })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	h.ReadinessHandler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, StatusDown, resp.Status)
	assert.Equal(t, StatusDown, resp.Checks["postgres"].Status)
	assert.True(t, resp.Checks["postgres"].Critical)
}

func TestReadinessHandler_BothCriticalAndNonCriticalDown_Returns503(t *testing.T) {
	h := NewHandler()
	h.RegisterCritical("postgres", func(ctx context.Context) error { return fmt.Errorf("db down") })
	h.RegisterNonCritical("redis", func(ctx context.Context) error { return fmt.Errorf("redis down") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	h.ReadinessHandler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, StatusDown, resp.Status)
}

func TestReadinessHandler_MultipleNonCriticalDown_ReturnsDegraded(t *testing.T) {
	h := NewHandler()
	h.RegisterCritical("postgres", func(ctx context.Context) error { return nil })
	h.RegisterNonCritical("kafka", func(ctx context.Context) error { return fmt.Errorf("kafka down") })
	h.RegisterNonCritical("redis", func(ctx context.Context) error { return fmt.Errorf("redis down") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	h.ReadinessHandler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, StatusDegraded, resp.Status)
	assert.Equal(t, StatusUp, resp.Checks["postgres"].Status)
	assert.Equal(t, StatusDown, resp.Checks["kafka"].Status)
	assert.Equal(t, StatusDown, resp.Checks["redis"].Status)
}

func TestRegister_IsCriticalByDefault(t *testing.T) {
	h := NewHandler()
	h.Register("db", func(ctx context.Context) error { return fmt.Errorf("fail") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	h.ReadinessHandler().ServeHTTP(rec, req)

	// Register defaults to critical, so a failure should produce 503.
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, StatusDown, resp.Status)
	assert.True(t, resp.Checks["db"].Critical)
}

func TestReadinessHandler_AllCriticalAndNonCriticalUp(t *testing.T) {
	h := NewHandler()
	h.RegisterCritical("postgres", func(ctx context.Context) error { return nil })
	h.RegisterNonCritical("kafka", func(ctx context.Context) error { return nil })
	h.RegisterNonCritical("redis", func(ctx context.Context) error { return nil })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	h.ReadinessHandler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, StatusUp, resp.Status)
}
