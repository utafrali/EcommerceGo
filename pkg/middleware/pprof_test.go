package middleware

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestIPAllowlist_AllowedIP(t *testing.T) {
	mw := IPAllowlist([]string{"127.0.0.0/8"}, discardLogger())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestIPAllowlist_DeniedIP(t *testing.T) {
	mw := IPAllowlist([]string{"10.0.0.0/8"}, discardLogger())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Contains(t, body, "error")
}

func TestIPAllowlist_MultipleCIDRs(t *testing.T) {
	cidrs := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	mw := IPAllowlist(cidrs, discardLogger())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name   string
		ip     string
		status int
	}{
		{"10.x allowed", "10.1.2.3:1234", http.StatusOK},
		{"172.16.x allowed", "172.16.5.5:1234", http.StatusOK},
		{"192.168.x allowed", "192.168.1.1:1234", http.StatusOK},
		{"8.8.8.8 denied", "8.8.8.8:1234", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.ip
			handler.ServeHTTP(rec, req)
			assert.Equal(t, tt.status, rec.Code)
		})
	}
}

func TestIPAllowlist_InvalidCIDR_Skipped(t *testing.T) {
	// Invalid CIDR should be skipped; valid one still works.
	mw := IPAllowlist([]string{"not-a-cidr", "127.0.0.0/8"}, discardLogger())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestIPAllowlist_IPv6(t *testing.T) {
	mw := IPAllowlist([]string{"::1/128"}, discardLogger())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "[::1]:1234"

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestIPAllowlist_NoPort(t *testing.T) {
	mw := IPAllowlist([]string{"127.0.0.0/8"}, discardLogger())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1" // no port

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestIPAllowlist_EmptyCIDRs_DeniesAll(t *testing.T) {
	mw := IPAllowlist(nil, discardLogger())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRegisterPprof_RoutesExist(t *testing.T) {
	r := chi.NewRouter()
	RegisterPprof(r, []string{"127.0.0.0/8"}, discardLogger())

	// Test that the pprof index is accessible.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "pprof")
}

func TestRegisterPprof_DeniedIP(t *testing.T) {
	r := chi.NewRouter()
	RegisterPprof(r, []string{"10.0.0.0/8"}, discardLogger())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRegisterPprof_CmdlineRoute(t *testing.T) {
	r := chi.NewRouter()
	RegisterPprof(r, []string{"127.0.0.0/8"}, discardLogger())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/cmdline", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	r.ServeHTTP(rec, req)

	// cmdline returns the command line arguments.
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRegisterPprof_SymbolRoute(t *testing.T) {
	r := chi.NewRouter()
	RegisterPprof(r, []string{"127.0.0.0/8"}, discardLogger())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/symbol", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRegisterPprof_HeapProfile(t *testing.T) {
	r := chi.NewRouter()
	RegisterPprof(r, []string{"127.0.0.0/8"}, discardLogger())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/heap", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	r.ServeHTTP(rec, req)

	// heap profile is served by pprof.Index via catch-all.
	assert.Equal(t, http.StatusOK, rec.Code)
}
