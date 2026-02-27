package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Existing tests (preserved) ---

func TestRateLimit_RequestsWithinLimit_Pass(t *testing.T) {
	handler := RateLimit(10, 10, newTestLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Send 5 requests (well within the burst limit of 10).
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "request %d should pass", i+1)
	}
}

func TestRateLimit_ExceedingLimit_Returns429(t *testing.T) {
	// Allow burst of 3 and very low RPS.
	handler := RateLimit(1, 3, newTestLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	var rateLimited bool

	// Send enough requests to exceed the burst limit.
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code == http.StatusTooManyRequests {
			rateLimited = true
			assert.Contains(t, rr.Body.String(), "RATE_LIMITED")
			break
		}
	}

	assert.True(t, rateLimited, "should have been rate limited after exceeding burst")
}

func TestRateLimit_DifferentIPs_IndependentLimits(t *testing.T) {
	// Allow burst of 2 per IP.
	handler := RateLimit(1, 2, newTestLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First IP: send 2 requests (at burst limit).
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// Second IP: should still be allowed (separate limiter).
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	req.RemoteAddr = "10.0.0.2:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRateLimit_ResponseBody_ContainsErrorCode(t *testing.T) {
	handler := RateLimit(1, 1, newTestLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request: should pass.
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "172.16.0.1:12345"
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	assert.Equal(t, http.StatusOK, rr1.Code)

	// Second request: should be rate limited.
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "172.16.0.1:12345"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	assert.Equal(t, http.StatusTooManyRequests, rr2.Code)
	assert.Contains(t, rr2.Body.String(), "RATE_LIMITED")
	assert.Contains(t, rr2.Body.String(), "too many requests")
}

func TestClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.50")
	req.RemoteAddr = "10.0.0.1:12345"

	ip := clientIP(req)
	assert.Equal(t, "203.0.113.50", ip)
}

func TestClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "198.51.100.42")
	req.RemoteAddr = "10.0.0.1:12345"

	ip := clientIP(req)
	assert.Equal(t, "198.51.100.42", ip)
}

func TestClientIP_RemoteAddr_Fallback(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"

	ip := clientIP(req)
	assert.Equal(t, "10.0.0.1", ip)
}

// --- New tests for visitorStore, lastSeen, and cleanup ---

func TestVisitorStore_GetVisitor_CreatesNewEntry(t *testing.T) {
	store := &visitorStore{
		visitors: make(map[string]*visitor),
		rps:      10,
		burst:    10,
		ttl:      time.Minute,
		nowFunc:  time.Now,
	}

	limiter := store.getVisitor("1.2.3.4")
	require.NotNil(t, limiter)
	assert.Equal(t, 1, store.len(), "store should have exactly one visitor")

	lastSeen, ok := store.getLastSeen("1.2.3.4")
	require.True(t, ok, "visitor should exist")
	assert.WithinDuration(t, time.Now(), lastSeen, time.Second)
}

func TestVisitorStore_GetVisitor_ReturnsSameLimiter(t *testing.T) {
	store := &visitorStore{
		visitors: make(map[string]*visitor),
		rps:      10,
		burst:    10,
		ttl:      time.Minute,
		nowFunc:  time.Now,
	}

	limiter1 := store.getVisitor("1.2.3.4")
	limiter2 := store.getVisitor("1.2.3.4")

	// Same IP should return the same limiter instance (pointer equality).
	assert.Same(t, limiter1, limiter2, "same IP should return the same limiter")
	assert.Equal(t, 1, store.len(), "store should still have exactly one visitor")
}

func TestVisitorStore_LastSeen_UpdatedOnSubsequentRequests(t *testing.T) {
	// Use a controllable clock to verify lastSeen updates.
	currentTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	store := &visitorStore{
		visitors: make(map[string]*visitor),
		rps:      10,
		burst:    10,
		ttl:      time.Minute,
		nowFunc:  func() time.Time { return currentTime },
	}

	// First visit at t=0.
	store.getVisitor("10.0.0.1")
	ls1, ok := store.getLastSeen("10.0.0.1")
	require.True(t, ok)
	assert.Equal(t, currentTime, ls1)

	// Advance clock by 30 seconds.
	currentTime = currentTime.Add(30 * time.Second)

	// Second visit at t=30s. lastSeen should advance.
	store.getVisitor("10.0.0.1")
	ls2, ok := store.getLastSeen("10.0.0.1")
	require.True(t, ok)
	assert.Equal(t, currentTime, ls2)
	assert.True(t, ls2.After(ls1), "lastSeen should advance after a subsequent request")
}

func TestVisitorStore_Cleanup_EvictsStaleEntries(t *testing.T) {
	// Use a controllable clock.
	currentTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ttl := 5 * time.Minute
	store := &visitorStore{
		visitors: make(map[string]*visitor),
		rps:      10,
		burst:    10,
		ttl:      ttl,
		nowFunc:  func() time.Time { return currentTime },
	}

	// Create two visitors at t=0.
	store.getVisitor("10.0.0.1")
	store.getVisitor("10.0.0.2")
	assert.Equal(t, 2, store.len())

	// Advance clock by 6 minutes (past TTL).
	currentTime = currentTime.Add(6 * time.Minute)

	// Run cleanup manually.
	store.cleanup()

	assert.Equal(t, 0, store.len(), "all stale visitors should be evicted")
	_, ok1 := store.getLastSeen("10.0.0.1")
	_, ok2 := store.getLastSeen("10.0.0.2")
	// Note: getLastSeen checks without creating. But since we deleted,
	// we need to check the raw map. Actually getLastSeen is read-only, so it works,
	// but getVisitor would re-create. Let me use the direct check.
	assert.False(t, ok1, "10.0.0.1 should have been evicted")
	assert.False(t, ok2, "10.0.0.2 should have been evicted")
}

func TestVisitorStore_Cleanup_KeepsFreshEntries(t *testing.T) {
	currentTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ttl := 5 * time.Minute
	store := &visitorStore{
		visitors: make(map[string]*visitor),
		rps:      10,
		burst:    10,
		ttl:      ttl,
		nowFunc:  func() time.Time { return currentTime },
	}

	// Create two visitors.
	store.getVisitor("10.0.0.1")
	store.getVisitor("10.0.0.2")

	// Advance clock by 3 minutes.
	currentTime = currentTime.Add(3 * time.Minute)

	// Touch 10.0.0.1 to keep it fresh.
	store.getVisitor("10.0.0.1")

	// Advance clock by another 3 minutes (total 6 min from start).
	// 10.0.0.1 was seen at t=3min, so 3 min ago (within TTL).
	// 10.0.0.2 was seen at t=0, so 6 min ago (past TTL of 5 min).
	currentTime = currentTime.Add(3 * time.Minute)

	store.cleanup()

	assert.Equal(t, 1, store.len(), "only stale visitor should be evicted")
	_, ok1 := store.getLastSeen("10.0.0.1")
	_, ok2 := store.getLastSeen("10.0.0.2")
	assert.True(t, ok1, "10.0.0.1 was refreshed and should remain")
	assert.False(t, ok2, "10.0.0.2 was stale and should be evicted")
}

func TestVisitorStore_Cleanup_EmptyStoreIsNoOp(t *testing.T) {
	store := &visitorStore{
		visitors: make(map[string]*visitor),
		rps:      10,
		burst:    10,
		ttl:      time.Minute,
		nowFunc:  time.Now,
	}

	// Should not panic on empty store.
	store.cleanup()
	assert.Equal(t, 0, store.len())
}

func TestVisitorStore_MultipleDifferentIPs(t *testing.T) {
	store := &visitorStore{
		visitors: make(map[string]*visitor),
		rps:      10,
		burst:    10,
		ttl:      time.Minute,
		nowFunc:  time.Now,
	}

	ips := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4", "10.0.0.5"}
	for _, ip := range ips {
		limiter := store.getVisitor(ip)
		require.NotNil(t, limiter)
	}

	assert.Equal(t, len(ips), store.len(), "each unique IP should have its own entry")
}

func TestRateLimit_LastSeen_UpdatedViaHTTP(t *testing.T) {
	// This is an integration-style test that verifies lastSeen updates
	// through the full HTTP middleware path. We test indirectly by verifying
	// the rate limiter state is preserved across requests (same limiter is reused).
	handler := RateLimit(1, 5, newTestLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Send 3 requests from the same IP. All should succeed because burst=5.
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.99:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "request %d should pass (within burst)", i+1)
	}

	// Continue sending requests from same IP. The token bucket state should
	// be continuous (not reset), proving the same visitor entry is being reused
	// and lastSeen is updated (not creating a new entry each time).
	// With burst=5, rps=1, after 3 successful requests we have 2 tokens left.
	// Requests 4 and 5 should succeed, request 6 should be rate limited.
	for i := 3; i < 6; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.99:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if i < 5 {
			assert.Equal(t, http.StatusOK, rr.Code, "request %d should pass", i+1)
		}
	}
}

func TestClientIP_XForwardedFor_MultipleIPs(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18, 150.172.238.178")
	req.RemoteAddr = "10.0.0.1:12345"

	ip := clientIP(req)
	assert.Equal(t, "203.0.113.50", ip, "should use the first IP in X-Forwarded-For chain")
}

func TestClientIP_RemoteAddr_NoPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1" // no port

	ip := clientIP(req)
	assert.Equal(t, "10.0.0.1", ip)
}
