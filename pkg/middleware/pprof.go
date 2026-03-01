package middleware

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"

	"github.com/go-chi/chi/v5"
)

// RegisterPprof adds pprof debug endpoints (/debug/pprof/*) to the router,
// protected by a CIDR-based IP allowlist. Only requests from IPs within the
// allowed CIDRs can access the profiling endpoints.
func RegisterPprof(r chi.Router, allowedCIDRs []string, logger *slog.Logger) {
	r.Group(func(r chi.Router) {
		r.Use(IPAllowlist(allowedCIDRs, logger))
		r.HandleFunc("/debug/pprof/*", pprof.Index)
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/debug/pprof/trace", pprof.Trace)
	})
}

// IPAllowlist returns middleware that restricts access to requests from IPs
// within the configured CIDR ranges. Invalid CIDRs are logged and skipped.
func IPAllowlist(cidrs []string, logger *slog.Logger) func(http.Handler) http.Handler {
	var nets []*net.IPNet
	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			logger.Warn("invalid allowlist CIDR, skipping",
				slog.String("cidr", cidr),
				slog.String("error", err.Error()),
			)
			continue
		}
		nets = append(nets, ipNet)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			ip := net.ParseIP(host)

			allowed := false
			if ip != nil {
				for _, n := range nets {
					if n.Contains(ip) {
						allowed = true
						break
					}
				}
			}

			if !allowed {
				logger.Warn("access denied by IP allowlist",
					slog.String("ip", host),
					slog.String("path", r.URL.Path),
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]string{
						"code":    "FORBIDDEN",
						"message": "access restricted by IP allowlist",
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
