package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// visitor tracks a rate limiter per client IP.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// visitorStore manages per-IP rate limiters with automatic cleanup
// of stale entries.
type visitorStore struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rps      int
	burst    int
	ttl      time.Duration
	nowFunc  func() time.Time // injectable clock for testing
}

// newVisitorStore creates a store with the given rate parameters and TTL.
// It starts a background cleanup goroutine that runs every ttl interval.
func newVisitorStore(rps, burst int, ttl time.Duration) *visitorStore {
	s := &visitorStore{
		visitors: make(map[string]*visitor),
		rps:      rps,
		burst:    burst,
		ttl:      ttl,
		nowFunc:  time.Now,
	}
	go s.cleanupLoop()
	return s
}

// getVisitor returns (or creates) a rate limiter for the given IP.
// It updates lastSeen on every call.
func (s *visitorStore) getVisitor(ip string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, exists := s.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rate.Limit(s.rps), s.burst)
		s.visitors[ip] = &visitor{limiter: limiter, lastSeen: s.now()}
		return limiter
	}
	v.lastSeen = s.now()
	return v.limiter
}

// now returns the current time, using the injectable clock.
func (s *visitorStore) now() time.Time {
	return s.nowFunc()
}

// cleanupLoop runs a ticker that evicts visitors not seen within the TTL.
func (s *visitorStore) cleanupLoop() {
	ticker := time.NewTicker(s.ttl)
	defer ticker.Stop()
	for range ticker.C {
		s.cleanup()
	}
}

// cleanup evicts all visitors whose lastSeen is older than the TTL.
func (s *visitorStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	for ip, v := range s.visitors {
		if now.Sub(v.lastSeen) > s.ttl {
			delete(s.visitors, ip)
		}
	}
}

// len returns the number of tracked visitors (used in tests).
func (s *visitorStore) len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.visitors)
}

// getLastSeen returns the lastSeen time for a given IP, and whether it exists.
func (s *visitorStore) getLastSeen(ip string) (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.visitors[ip]
	if !ok {
		return time.Time{}, false
	}
	return v.lastSeen, true
}

// RateLimit returns middleware that enforces per-IP token bucket rate limiting.
// rps is the number of requests per second allowed, and burst is the maximum burst size.
// Returns HTTP 429 Too Many Requests when the limit is exceeded.
func RateLimit(rps, burst int, logger *slog.Logger) func(http.Handler) http.Handler {
	const cleanupInterval = 3 * time.Minute
	store := newVisitorStore(rps, burst, cleanupInterval)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			limiter := store.getVisitor(ip)

			if !limiter.Allow() {
				logger.Warn("rate limit exceeded",
					slog.String("ip", ip),
					slog.String("path", r.URL.Path),
				)
				writeJSONError(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the client IP address from the request.
// It checks X-Forwarded-For and X-Real-IP headers before falling back to RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain.
		if i := len(xff); i > 0 {
			parts := net.ParseIP(xff)
			if parts != nil {
				return xff
			}
			// Try splitting on comma for multiple proxies.
			for _, part := range splitFirst(xff, ",") {
				ip := net.ParseIP(trimSpace(part))
				if ip != nil {
					return ip.String()
				}
			}
		}
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if ip := net.ParseIP(xri); ip != nil {
			return ip.String()
		}
	}

	// Fall back to RemoteAddr, stripping the port.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// splitFirst splits s by sep and returns the parts. Simple helper to avoid importing strings.
func splitFirst(s, sep string) []string {
	var result []string
	for {
		i := indexOf(s, sep)
		if i < 0 {
			result = append(result, s)
			break
		}
		result = append(result, s[:i])
		s = s[i+len(sep):]
	}
	return result
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
