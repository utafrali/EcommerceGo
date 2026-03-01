package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/utafrali/EcommerceGo/pkg/health"
	"github.com/utafrali/EcommerceGo/pkg/tracing"
	"github.com/utafrali/EcommerceGo/services/gateway/internal/config"
	"github.com/utafrali/EcommerceGo/services/gateway/internal/handler"
	"github.com/utafrali/EcommerceGo/services/gateway/internal/proxy"
)

// App wires together all dependencies and runs the API gateway.
type App struct {
	cfg            *config.Config
	logger         *slog.Logger
	httpServer     *http.Server
	tracerShutdown func(context.Context) error
}

// NewApp creates a new application instance, initializing the reverse proxy
// and HTTP router. The gateway has no database or Kafka dependencies.
func NewApp(cfg *config.Config, logger *slog.Logger) (*App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize OpenTelemetry tracing.
	tracerShutdown, err := tracing.InitTracer(ctx, tracing.Config{
		ServiceName:    "gateway",
		ServiceVersion: "0.1.0",
		Environment:    cfg.Environment,
		OTLPEndpoint:   cfg.OTELEndpoint,
		SampleRate:     cfg.OTELSampleRate,
		Enabled:        cfg.OTELEnabled,
	})
	if err != nil {
		return nil, fmt.Errorf("init tracer: %w", err)
	}

	// Initialize the service proxy with backend URLs.
	sp := proxy.NewServiceProxy(cfg, logger)

	// Health checks with downstream service reachability.
	healthHandler := health.NewHandler()
	healthHandler.RegisterNonCritical("downstream", func(ctx context.Context) error {
		u, err := url.Parse(cfg.ProductServiceURL)
		if err != nil {
			return fmt.Errorf("parse product service URL: %w", err)
		}
		d := net.Dialer{Timeout: 2 * time.Second}
		conn, err := d.DialContext(ctx, "tcp", u.Host)
		if err != nil {
			return fmt.Errorf("downstream unreachable: %w", err)
		}
		_ = conn.Close()
		return nil
	})

	// Build the HTTP router with middleware and proxy routes.
	router := handler.NewRouter(cfg, sp, healthHandler, logger)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           router,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return &App{
		cfg:            cfg,
		logger:         logger,
		httpServer:     httpServer,
		tracerShutdown: tracerShutdown,
	}, nil
}

// Run starts the HTTP server and blocks until the context is canceled.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 2)

	go func() {
		a.logger.Info("starting HTTP server",
			slog.String("addr", a.httpServer.Addr),
		)
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("http server: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		a.logger.Info("shutdown signal received")
	case err := <-errCh:
		return err
	}

	return a.Shutdown()
}

// Shutdown gracefully stops the HTTP server in the correct order:
// 1. HTTP server (drain in-flight requests)
// 2. Tracer (flush pending spans from drained requests)
func (a *App) Shutdown() error {
	a.logger.Info("shutting down application...")

	var errs []error

	// 1. Drain in-flight HTTP requests (5s budget).
	httpCtx, httpCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer httpCancel()
	if err := a.httpServer.Shutdown(httpCtx); err != nil {
		a.logger.Error("http server shutdown error", slog.String("error", err.Error()))
		errs = append(errs, err)
	}

	// 2. Flush pending spans after HTTP drain so in-flight request spans are captured.
	if a.tracerShutdown != nil {
		tracerCtx, tracerCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer tracerCancel()
		if err := a.tracerShutdown(tracerCtx); err != nil {
			a.logger.Error("tracer shutdown error", slog.String("error", err.Error()))
			errs = append(errs, err)
		}
	}

	a.logger.Info("application shutdown complete")
	return errors.Join(errs...)
}
