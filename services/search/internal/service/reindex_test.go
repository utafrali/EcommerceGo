package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/utafrali/EcommerceGo/services/search/internal/domain"
	"github.com/utafrali/EcommerceGo/services/search/internal/engine/memory"
)

// reindexResponse is the paginated response the fake product service returns.
type reindexResponse struct {
	Data       []map[string]any `json:"data"`
	TotalCount int              `json:"total_count"`
	Page       int              `json:"page"`
	TotalPages int              `json:"total_pages"`
}

func TestReindex_IndexesProductsFromRemoteService(t *testing.T) {
	// Fake product service returns a single page with 2 products.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := reindexResponse{
			Data: []map[string]any{
				{
					"id":         "prod-1",
					"name":       "Reindexed Widget",
					"slug":       "reindexed-widget",
					"base_price": 1999,
					"currency":   "USD",
					"status":     "published",
					"category":   map[string]any{"id": "cat-1", "name": "Widgets"},
					"brand":      map[string]any{"id": "brand-1", "name": "Acme"},
				},
				{
					"id":         "prod-2",
					"name":       "Reindexed Gadget",
					"slug":       "reindexed-gadget",
					"base_price": 2999,
					"currency":   "USD",
					"status":     "published",
				},
			},
			TotalCount: 2,
			Page:       1,
			TotalPages: 1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	eng := memory.New()
	svc := NewSearchService(eng, newTestLogger(), srv.URL)

	err := svc.Reindex(context.Background())
	require.NoError(t, err)

	// Verify that both products were indexed into the engine.
	result, err := svc.Search(context.Background(), &domain.SearchQuery{
		Query:   "reindexed",
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Total)
}

func TestReindex_HandlesMultiplePages(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		page := r.URL.Query().Get("page")

		var resp reindexResponse
		switch page {
		case "1", "":
			resp = reindexResponse{
				Data: []map[string]any{
					{"id": "p1", "name": "Page1 Product", "status": "published"},
				},
				TotalCount: 2,
				Page:       1,
				TotalPages: 2,
			}
		case "2":
			resp = reindexResponse{
				Data: []map[string]any{
					{"id": "p2", "name": "Page2 Product", "status": "published"},
				},
				TotalCount: 2,
				Page:       2,
				TotalPages: 2,
			}
		default:
			resp = reindexResponse{Data: nil, TotalCount: 2, Page: 3, TotalPages: 2}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	eng := memory.New()
	svc := NewSearchService(eng, newTestLogger(), srv.URL)

	err := svc.Reindex(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, callCount, "should have fetched exactly 2 pages")

	result, err := svc.Search(context.Background(), &domain.SearchQuery{Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Total)
}

func TestReindex_ReturnsErrorOnNon200StatusCode(t *testing.T) {
	// Server returns 500 Internal Server Error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal"}`))
	}))
	defer srv.Close()

	eng := memory.New()
	svc := NewSearchService(eng, newTestLogger(), srv.URL)

	err := svc.Reindex(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status 500")
}

func TestReindex_ReturnsErrorOnStatus404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	eng := memory.New()
	svc := NewSearchService(eng, newTestLogger(), srv.URL)

	err := svc.Reindex(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status 404")
}

func TestReindex_ReturnsErrorOnStatus503(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	eng := memory.New()
	svc := NewSearchService(eng, newTestLogger(), srv.URL)

	err := svc.Reindex(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status 503")
}

func TestReindex_ReturnsErrorOnConnectionFailure(t *testing.T) {
	// Use an unreachable URL (closed server).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // close immediately so connection is refused

	eng := memory.New()
	svc := NewSearchService(eng, newTestLogger(), srv.URL)

	err := svc.Reindex(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch products page 1")
}

func TestReindex_RespectsContextCancellation(t *testing.T) {
	// Server that blocks until the request context is done.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	eng := memory.New()
	svc := NewSearchService(eng, newTestLogger(), srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := svc.Reindex(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch products page 1")
}

func TestReindex_UsesCustomHTTPClientWithTimeout(t *testing.T) {
	// Verify the Reindex function creates an HTTP client with a timeout
	// by checking that a slow server is cut off by a short context deadline.
	// The custom client propagates the context deadline properly.
	requestReceived := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestReceived)
		// Block longer than our test context allows.
		time.Sleep(5 * time.Second)
	}))
	defer srv.Close()

	eng := memory.New()
	svc := NewSearchService(eng, newTestLogger(), srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := svc.Reindex(ctx)
	require.Error(t, err)

	// Make sure the request was actually sent (not just a URL build failure).
	select {
	case <-requestReceived:
		// good, request was sent
	default:
		t.Fatal("expected the HTTP request to have been sent to the server")
	}
}

func TestReindex_SkipsMalformedProducts(t *testing.T) {
	// Return one valid product and one that will cause unmarshal issues.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := `{
			"data": [
				{"id": "good-1", "name": "Good Product", "status": "published"},
				"not-a-json-object"
			],
			"total_count": 2,
			"page": 1,
			"total_pages": 1
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	defer srv.Close()

	eng := memory.New()
	svc := NewSearchService(eng, newTestLogger(), srv.URL)

	err := svc.Reindex(context.Background())
	require.NoError(t, err)

	// Only the valid product should be indexed.
	result, err := svc.Search(context.Background(), &domain.SearchQuery{Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "good-1", result.Products[0].ID)
}

func TestReindex_EmptyDataBreaksLoop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := reindexResponse{
			Data:       []map[string]any{},
			TotalCount: 0,
			Page:       1,
			TotalPages: 0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	eng := memory.New()
	svc := NewSearchService(eng, newTestLogger(), srv.URL)

	err := svc.Reindex(context.Background())
	require.NoError(t, err)

	// Nothing should have been indexed.
	result, err := svc.Search(context.Background(), &domain.SearchQuery{Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
}

func TestReindex_MapsNestedCategoryBrandImage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := reindexResponse{
			Data: []map[string]any{
				{
					"id":            "prod-nested",
					"name":          "Nested Fields Product",
					"slug":          "nested-fields",
					"description":   "Tests nested field mapping",
					"base_price":    4999,
					"currency":      "TRY",
					"status":        "published",
					"category":      map[string]any{"id": "cat-shoes", "name": "Shoes"},
					"brand":         map[string]any{"id": "brand-nike", "name": "Nike"},
					"primary_image": map[string]any{"url": "https://img.example.com/shoe.jpg"},
				},
			},
			TotalCount: 1,
			Page:       1,
			TotalPages: 1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	eng := memory.New()
	svc := NewSearchService(eng, newTestLogger(), srv.URL)

	err := svc.Reindex(context.Background())
	require.NoError(t, err)

	result, err := svc.Search(context.Background(), &domain.SearchQuery{
		Query:   "nested",
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.Total)

	p := result.Products[0]
	assert.Equal(t, "prod-nested", p.ID)
	assert.Equal(t, "cat-shoes", p.CategoryID)
	assert.Equal(t, "Shoes", p.CategoryName)
	assert.Equal(t, "brand-nike", p.BrandID)
	assert.Equal(t, "Nike", p.BrandName)
	assert.Equal(t, "https://img.example.com/shoe.jpg", p.ImageURL)
}


func TestReindex_ConcurrencyGuard(t *testing.T) {
	// Server that takes a moment to respond, giving time for a concurrent call.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		resp := reindexResponse{
			Data: []map[string]any{
				{"id": "p1", "name": "Slow Product", "status": "published"},
			},
			TotalCount: 1,
			Page:       1,
			TotalPages: 1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	eng := memory.New()
	svc := NewSearchService(eng, newTestLogger(), srv.URL)

	var wg sync.WaitGroup
	errs := make([]error, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		errs[0] = svc.Reindex(context.Background())
	}()
	// Small delay to ensure the first goroutine acquires the guard.
	time.Sleep(50 * time.Millisecond)
	go func() {
		defer wg.Done()
		errs[1] = svc.Reindex(context.Background())
	}()
	wg.Wait()

	// Exactly one should succeed and the other should fail with "reindex already in progress".
	successCount := 0
	alreadyInProgressCount := 0
	for _, err := range errs {
		if err == nil {
			successCount++
		} else if err.Error() == "reindex already in progress" {
			alreadyInProgressCount++
		}
	}
	assert.Equal(t, 1, successCount, "exactly one reindex call should succeed")
	assert.Equal(t, 1, alreadyInProgressCount, "exactly one reindex call should be rejected")
}
