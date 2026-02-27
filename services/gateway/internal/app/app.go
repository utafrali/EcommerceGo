package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/utafrali/EcommerceGo/services/gateway/internal/config"
	"github.com/utafrali/EcommerceGo/services/gateway/internal/handler"
	"github.com/utafrali/EcommerceGo/services/gateway/internal/proxy"
)

// App wires together all dependencies and runs the API gateway.
type App struct {
	cfg        *config.Config
	logger     *slog.Logger
	httpServer *http.Server
}

// NewApp creates a new application instance, initializing the reverse proxy
// and HTTP router. The gateway has no database or Kafka dependencies.
func NewApp(cfg *config.Config, logger *slog.Logger) (*App, error) {
	// Initialize the service proxy with backend URLs.
	sp := proxy.NewServiceProxy(cfg, logger)

	// Build the HTTP router with middleware and proxy routes.
	router := handler.NewRouter(cfg, sp, logger)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &App{
		cfg:        cfg,
		logger:     logger,
		httpServer: httpServer,
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

// Shutdown gracefully stops the HTTP server.
func (a *App) Shutdown() error {
	a.logger.Info("shutting down application...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
		a.logger.Error("http server shutdown error", slog.String("error", err.Error()))
	}

	a.logger.Info("application shutdown complete")
	return nil
}
