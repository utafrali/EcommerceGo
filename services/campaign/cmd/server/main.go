package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/utafrali/EcommerceGo/pkg/logger"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/app"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	// Load configuration from environment variables.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Initialize structured logger.
	log := logger.New("campaign-service", cfg.LogLevel)
	log.Info("starting campaign service",
		slog.String("environment", cfg.Environment),
		slog.Int("http_port", cfg.HTTPPort),
		slog.Int("grpc_port", cfg.GRPCPort),
	)

	// Create the application with all dependencies wired.
	application, err := app.NewApp(cfg, log)
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}

	// Create a context that is canceled on SIGINT or SIGTERM.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Run the application. This blocks until shutdown.
	if err := application.Run(ctx); err != nil {
		return fmt.Errorf("run application: %w", err)
	}

	log.Info("campaign service stopped")
	return nil
}
