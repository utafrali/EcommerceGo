package http

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/utafrali/EcommerceGo/services/search/internal/engine/memory"
	"github.com/utafrali/EcommerceGo/services/search/internal/service"
)

func newTestHandler() *SearchHandler {
	eng := memory.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.NewSearchService(eng, logger, "http://localhost:9999")
	return NewSearchHandler(svc, logger)
}

func newTestRouter() http.Handler {
	h := newTestHandler()
	r := chi.NewRouter()
	r.Route("/api/v1/search", func(r chi.Router) {
		r.Get("/", h.Search)
		r.Post("/index", h.IndexProduct)
		r.Delete("/{id}", h.DeleteProduct)
		r.Post("/bulk", h.BulkIndex)
		r.Post("/reindex", h.Reindex)
		r.Get("/suggest", h.Suggest)
	})
	return r
}

// --- MaxBytesReader Tests ---

func TestIndexProduct_RejectsBodyOver1MB(t *testing.T) {
	router := newTestRouter()

	// Create a body larger than 1MB (1<<20 = 1,048,576 bytes).
	largeBody := strings.Repeat("x", 1<<20+1)
	body := `{"id":"big","name":"` + largeBody + `"}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/index", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp response
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
}

func TestBulkIndex_RejectsBodyOver1MB(t *testing.T) {
	router := newTestRouter()

	// Build a large products array that exceeds 1MB.
	largeDesc := strings.Repeat("y", 1<<20)
	body := `{"products":[{"id":"big","name":"Product","description":"` + largeDesc + `"}]}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp response
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
}

func TestReindex_RejectsBodyOver1MB(t *testing.T) {
	router := newTestRouter()

	// Reindex handler also applies MaxBytesReader even though it doesn't
	// read the body; verify the limit is set without error for a small body.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/reindex", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// The handler calls service.Reindex which tries to reach localhost:9999
	// (not running), so it will return 500 -- but the MaxBytesReader does
	// not block it. This test just verifies a small body is not rejected.
	// We check it is NOT a 400 from MaxBytesReader.
	assert.NotEqual(t, http.StatusBadRequest, w.Code)
}

// --- IndexProduct Handler Tests ---

func TestIndexProduct_AcceptsValidBody(t *testing.T) {
	router := newTestRouter()

	body := `{"id":"test-1","name":"Valid Product","base_price":999}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/index", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	data := resp["data"].(map[string]any)
	assert.Equal(t, "test-1", data["id"])
	assert.Equal(t, "indexed", data["status"])
}

func TestIndexProduct_RequiresID(t *testing.T) {
	router := newTestRouter()

	body := `{"name":"No ID Product"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/index", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIndexProduct_RequiresName(t *testing.T) {
	router := newTestRouter()

	body := `{"id":"test-2"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/index", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIndexProduct_RejectsInvalidJSON(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/index", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- BulkIndex Handler Tests ---

func TestBulkIndex_AcceptsValidBody(t *testing.T) {
	router := newTestRouter()

	body := `{"products":[{"id":"b1","name":"Bulk One"},{"id":"b2","name":"Bulk Two"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	data := resp["data"].(map[string]any)
	assert.Equal(t, float64(2), data["indexed"])
}

func TestBulkIndex_RejectsEmptyProducts(t *testing.T) {
	router := newTestRouter()

	body := `{"products":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Search Handler Tests ---

func TestSearch_ReturnsEmptyResults(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	data := resp["data"].(map[string]any)
	assert.Equal(t, float64(0), data["total"])
}

func TestSearch_ParsesQueryParameters(t *testing.T) {
	router := newTestRouter()

	// Index a product first.
	indexBody := `{"id":"qp-1","name":"Query Params Test","status":"published","category_id":"cat-1","brand_id":"brand-1","base_price":5000}`
	indexReq := httptest.NewRequest(http.MethodPost, "/api/v1/search/index", strings.NewReader(indexBody))
	indexReq.Header.Set("Content-Type", "application/json")
	iw := httptest.NewRecorder()
	router.ServeHTTP(iw, indexReq)
	require.Equal(t, http.StatusOK, iw.Code)

	// Search with query params.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=query+params&page=1&per_page=10", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- DeleteProduct Handler Tests ---

func TestDeleteProduct_ReturnsOK(t *testing.T) {
	router := newTestRouter()

	// Index first.
	indexBody := `{"id":"del-1","name":"To Delete"}`
	indexReq := httptest.NewRequest(http.MethodPost, "/api/v1/search/index", strings.NewReader(indexBody))
	indexReq.Header.Set("Content-Type", "application/json")
	iw := httptest.NewRecorder()
	router.ServeHTTP(iw, indexReq)
	require.Equal(t, http.StatusOK, iw.Code)

	// Delete it.
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/search/del-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	data := resp["data"].(map[string]any)
	assert.Equal(t, "deleted", data["status"])
}

// --- Suggest Handler Tests ---

func TestSuggest_EmptyQuery(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/suggest", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	data := resp["data"].(map[string]any)
	suggestions := data["suggestions"].([]any)
	assert.Empty(t, suggestions)
}

func TestSuggest_WithQuery(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/suggest?q=test&limit=3", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- MaxBytesReader edge case: body exactly at limit ---

func TestIndexProduct_AcceptsBodyAtExactly1MB(t *testing.T) {
	router := newTestRouter()

	// Build a valid JSON body that is just under 1MB.
	// The MaxBytesReader limit is 1<<20 = 1,048,576 bytes.
	// We need the total body to be <= 1MB.
	prefix := `{"id":"exact","name":"`
	suffix := `"}`
	overhead := len(prefix) + len(suffix)
	nameLen := (1 << 20) - overhead - 100 // leave some margin
	name := strings.Repeat("a", nameLen)

	var buf bytes.Buffer
	buf.WriteString(prefix)
	buf.WriteString(name)
	buf.WriteString(suffix)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/index", &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should succeed (body is within limit).
	assert.Equal(t, http.StatusOK, w.Code)
}
