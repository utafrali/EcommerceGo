package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/utafrali/EcommerceGo/pkg/health"
	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/pkg/tracing"
	"github.com/utafrali/EcommerceGo/services/cart/internal/config"
	"github.com/utafrali/EcommerceGo/services/cart/internal/event"
	handler "github.com/utafrali/EcommerceGo/services/cart/internal/handler/http"
	redisrepo "github.com/utafrali/EcommerceGo/services/cart/internal/repository/redis"
	"github.com/utafrali/EcommerceGo/services/cart/internal/service"
)

// App wires together all dependencies and runs the cart service.
type App struct {
	cfg            *config.Config
	logger         *slog.Logger
	rdb            *redis.Client
	producer       *pkgkafka.Producer
	httpServer     *http.Server
	tracerShutdown func(context.Context) error
}

// NewApp creates a new application instance, initializing all dependencies.
func NewApp(cfg *config.Config, logger *slog.Logger) (*App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize OpenTelemetry tracing.
	tracerShutdown, err := tracing.InitTracer(ctx, tracing.Config{
		ServiceName:    "cart",
		ServiceVersion: "0.1.0",
		Environment:    cfg.Environment,
		OTLPEndpoint:   cfg.OTELEndpoint,
		SampleRate:     cfg.OTELSampleRate,
		Enabled:        cfg.OTELEnabled,
	})
	if err != nil {
		return nil, fmt.Errorf("init tracer: %w", err)
	}

	// Initialize Redis client with retry logic (3 attempts, 1s/2s/4s backoff).
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
	})

	redisConnected := false
	for attempt := 0; attempt < 3; attempt++ {
		if err := rdb.Ping(ctx).Err(); err == nil {
			redisConnected = true
			break
		} else if attempt < 2 {
			base := time.Duration(1<<uint(attempt)) * time.Second
			jitter := time.Duration(float64(base) * 0.25 * (2*rand.Float64() - 1)) // #nosec G404 -- non-cryptographic jitter for retry backoff
			wait := base + jitter
			logger.Warn("redis connection failed, retrying",
				slog.Int("attempt", attempt+1),
				slog.Int("max_attempts", 3),
				slog.Duration("backoff", wait),
				slog.String("error", err.Error()),
			)
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("connect to redis: context cancelled during retry: %w", ctx.Err())
			case <-time.After(wait):
			}
		} else {
			logger.Warn("redis unavailable after 3 attempts, starting in degraded mode",
				slog.String("addr", cfg.RedisAddr),
				slog.String("error", err.Error()),
			)
		}
	}

	if redisConnected {
		logger.Info("connected to Redis",
			slog.String("addr", cfg.RedisAddr),
			slog.Int("db", cfg.RedisDB),
		)
	}

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
	healthHandler.RegisterCritical("redis", func(ctx context.Context) error {
		return rdb.Ping(ctx).Err()
	})
	healthHandler.RegisterNonCritical("kafka", func(ctx context.Context) error {
		return producer.Ping(ctx)
	})

	// HTTP router.
	router := handler.NewRouter(cartService, healthHandler, logger, cfg.PprofAllowedCIDRs)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return &App{
		cfg:            cfg,
		logger:         logger,
		rdb:            rdb,
		producer:       producer,
		httpServer:     httpServer,
		tracerShutdown: tracerShutdown,
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

// Shutdown gracefully stops all components in the correct order:
// 1. HTTP server (drain in-flight requests)
// 2. Tracer (flush pending spans from drained requests)
// 3. Kafka producer
// 4. Redis client
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

	// 3. Close Kafka producer (2s budget).
	if err := a.producer.Close(); err != nil {
		a.logger.Error("kafka producer close error", slog.String("error", err.Error()))
		errs = append(errs, err)
	}

	// 4. Close Redis client.
	if err := a.rdb.Close(); err != nil {
		a.logger.Error("redis close error", slog.String("error", err.Error()))
		errs = append(errs, err)
	}

	a.logger.Info("application shutdown complete")
	return errors.Join(errs...)
}
