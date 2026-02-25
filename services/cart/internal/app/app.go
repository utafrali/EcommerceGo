package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/utafrali/EcommerceGo/pkg/health"
	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/cart/internal/config"
	"github.com/utafrali/EcommerceGo/services/cart/internal/event"
	handler "github.com/utafrali/EcommerceGo/services/cart/internal/handler/http"
	redisrepo "github.com/utafrali/EcommerceGo/services/cart/internal/repository/redis"
	"github.com/utafrali/EcommerceGo/services/cart/internal/service"
)

// App wires together all dependencies and runs the cart service.
type App struct {
	cfg        *config.Config
	logger     *slog.Logger
	rdb        *redis.Client
	producer   *pkgkafka.Producer
	httpServer *http.Server
}

// NewApp creates a new application instance, initializing all dependencies.
func NewApp(cfg *config.Config, logger *slog.Logger) (*App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize Redis client.
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connect to redis: %w", err)
	}
	logger.Info("connected to Redis",
		slog.String("addr", cfg.RedisAddr),
		slog.Int("db", cfg.RedisDB),
	)

	// Initialize Kafka producer.
	kafkaCfg := pkgkafka.DefaultProducerConfig(cfg.KafkaBrokers)
	producer := pkgkafka.NewProducer(kafkaCfg, logger)
	logger.Info("kafka producer initialized", slog.Any("brokers", cfg.KafkaBrokers))

	// Build the dependency graph.
	cartTTL := time.Duration(cfg.CartTTL) * time.Hour
	repo := redisrepo.NewCartRepository(rdb, cartTTL)
	eventProducer := event.NewProducer(producer, logger)
	cartService := service.NewCartService(repo, eventProducer, logger, cartTTL)

	// Health checks.
	healthHandler := health.NewHandler()
	healthHandler.Register("redis", func(ctx context.Context) error {
		return rdb.Ping(ctx).Err()
	})

	// HTTP router.
	router := handler.NewRouter(cartService, healthHandler, logger)

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
		rdb:        rdb,
		producer:   producer,
		httpServer: httpServer,
	}, nil
}

// Run starts the HTTP server and blocks until the context is canceled.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

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

// Shutdown gracefully stops all components.
func (a *App) Shutdown() error {
	a.logger.Info("shutting down application...")

	// Graceful HTTP server shutdown with a 10-second deadline.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
		a.logger.Error("http server shutdown error", slog.String("error", err.Error()))
	}

	// Close Kafka producer.
	if err := a.producer.Close(); err != nil {
		a.logger.Error("kafka producer close error", slog.String("error", err.Error()))
	}

	// Close Redis client.
	if err := a.rdb.Close(); err != nil {
		a.logger.Error("redis close error", slog.String("error", err.Error()))
	}

	a.logger.Info("application shutdown complete")
	return nil
}
