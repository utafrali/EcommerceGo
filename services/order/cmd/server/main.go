package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/utafrali/EcommerceGo/pkg/logger"
	"github.com/utafrali/EcommerceGo/services/order/internal/app"
	"github.com/utafrali/EcommerceGo/services/order/internal/config"
)

func main() {
	// Load configuration from environment variables.
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize structured logger.
	log := logger.New("order-service", cfg.LogLevel)
	log.Info("starting order service",
		slog.String("environment", cfg.Environment),
		slog.Int("http_port", cfg.HTTPPort),
		slog.Int("grpc_port", cfg.GRPCPort),
	)

	// Create the application with all dependencies wired.
	application, err := app.NewApp(cfg, log)
	if err != nil {
		log.Error("failed to initialize application", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Create a context that is canceled on SIGINT or SIGTERM.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Run the application. This blocks until shutdown.
	if err := application.Run(ctx); err != nil {
		log.Error("application error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log.Info("order service stopped")
}
