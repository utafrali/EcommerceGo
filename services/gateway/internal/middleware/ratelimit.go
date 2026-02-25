package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// visitor tracks a rate limiter per client IP.
type visitor struct {
	limiter *rate.Limiter
}

// RateLimit returns middleware that enforces per-IP token bucket rate limiting.
// rps is the number of requests per second allowed, and burst is the maximum burst size.
// Returns HTTP 429 Too Many Requests when the limit is exceeded.
func RateLimit(rps, burst int, logger *slog.Logger) func(http.Handler) http.Handler {
	var (
		mu       sync.Mutex
		visitors = make(map[string]*visitor)
	)

	getVisitor := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()

		v, exists := visitors[ip]
		if !exists {
			limiter := rate.NewLimiter(rate.Limit(rps), burst)
			visitors[ip] = &visitor{limiter: limiter}
			return limiter
		}
		return v.limiter
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			limiter := getVisitor(ip)

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
