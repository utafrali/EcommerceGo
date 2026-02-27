package proxy

import (
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	pkglogger "github.com/utafrali/EcommerceGo/pkg/logger"
	"github.com/utafrali/EcommerceGo/services/gateway/internal/config"
)

// ServiceProxy manages reverse proxies to backend microservices.
type ServiceProxy struct {
	routes map[string]*httputil.ReverseProxy
	logger *slog.Logger
}

// NewServiceProxy creates a new ServiceProxy with reverse proxies for each backend service.
func NewServiceProxy(cfg *config.Config, logger *slog.Logger) *ServiceProxy {
	sp := &ServiceProxy{
		routes: make(map[string]*httputil.ReverseProxy),
		logger: logger,
	}

	serviceURLs := map[string]string{
		"product":      cfg.ProductServiceURL,
		"cart":         cfg.CartServiceURL,
		"order":        cfg.OrderServiceURL,
		"checkout":     cfg.CheckoutServiceURL,
		"payment":      cfg.PaymentServiceURL,
		"user":         cfg.UserServiceURL,
		"inventory":    cfg.InventoryServiceURL,
		"campaign":     cfg.CampaignServiceURL,
		"notification": cfg.NotificationServiceURL,
		"search":       cfg.SearchServiceURL,
		"media":        cfg.MediaServiceURL,
	}

	for name, rawURL := range serviceURLs {
		target, err := url.Parse(rawURL)
		if err != nil {
			logger.Error("invalid service URL",
				slog.String("service", name),
				slog.String("url", rawURL),
				slog.String("error", err.Error()),
			)
			continue
		}

		proxy := httputil.NewSingleHostReverseProxy(target)

		// Use a custom transport with sensible dial/idle/response timeouts
		// to prevent a single slow backend from exhausting gateway resources.
		proxy.Transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ResponseHeaderTimeout: 30 * time.Second,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
		}

		// Wrap the default Director to ensure gateway-injected headers
		// (X-User-ID, X-User-Email, X-User-Role, Authorization) are
		// explicitly forwarded to backend services, and standard proxy
		// headers (X-Forwarded-For, X-Forwarded-Host, X-Forwarded-Proto)
		// are set for request tracing.
		defaultDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			// Run the default director (sets target host/scheme/path).
			defaultDirector(req)

			// Preserve the original Host header for backends that need it.
			if req.Header.Get("X-Forwarded-Host") == "" {
				req.Header.Set("X-Forwarded-Host", req.Host)
			}

			// Set X-Forwarded-Proto based on the incoming request scheme.
			if req.Header.Get("X-Forwarded-Proto") == "" {
				proto := "http"
				if req.TLS != nil {
					proto = "https"
				}
				req.Header.Set("X-Forwarded-Proto", proto)
			}

			// Forward X-Correlation-ID to backend services. If the header
			// is not already set (e.g. by an upstream load balancer), pull
			// the ID that the gateway's RequestLogging middleware generated
			// and stored in the request context.
			if req.Header.Get("X-Correlation-ID") == "" {
				if corrID := pkglogger.CorrelationIDFromContext(req.Context()); corrID != "" {
					req.Header.Set("X-Correlation-ID", corrID)
				}
			}
		}

		proxy.ErrorHandler = sp.errorHandler(name)
		sp.routes[name] = proxy

		logger.Info("registered service proxy",
			slog.String("service", name),
			slog.String("target", rawURL),
		)
	}

	return sp
}

// Handler returns an http.Handler that proxies requests to the named backend service.
func (sp *ServiceProxy) Handler(serviceName string) http.Handler {
	proxy, ok := sp.routes[serviceName]
	if !ok {
		sp.logger.Error("no proxy registered for service", slog.String("service", serviceName))
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"code":"SERVICE_UNAVAILABLE","message":"service not configured"}`, http.StatusBadGateway)
		})
	}
	return proxy
}

// errorHandler returns an error handler for the reverse proxy that logs errors
// and writes a JSON error response.
func (sp *ServiceProxy) errorHandler(serviceName string) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		sp.logger.Error("proxy error",
			slog.String("service", serviceName),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("error", err.Error()),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"code":"BAD_GATEWAY","message":"upstream service unavailable"}`))
	}
}
