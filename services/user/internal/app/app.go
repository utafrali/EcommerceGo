package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/utafrali/EcommerceGo/pkg/database"
	"github.com/utafrali/EcommerceGo/pkg/health"
	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/pkg/tracing"
	"github.com/utafrali/EcommerceGo/services/user/internal/auth"
	"github.com/utafrali/EcommerceGo/services/user/internal/config"
	"github.com/utafrali/EcommerceGo/services/user/internal/event"
	handler "github.com/utafrali/EcommerceGo/services/user/internal/handler/http"
	"github.com/utafrali/EcommerceGo/services/user/internal/repository/postgres"
	"github.com/utafrali/EcommerceGo/services/user/internal/service"
	"github.com/utafrali/EcommerceGo/services/user/migrations"
)

// App wires together all dependencies and runs the user service.
type App struct {
	cfg            *config.Config
	logger         *slog.Logger
	pool           *pgxpool.Pool
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
		ServiceName:    "user",
		ServiceVersion: "0.1.0",
		Environment:    cfg.Environment,
		OTLPEndpoint:   cfg.OTELEndpoint,
		SampleRate:     cfg.OTELSampleRate,
		Enabled:        cfg.OTELEnabled,
	})
	if err != nil {
		return nil, fmt.Errorf("init tracer: %w", err)
	}

	// Initialize PostgreSQL connection pool.
	pgCfg := database.PostgresConfig{
		Host:            cfg.PostgresHost,
		Port:            cfg.PostgresPort,
		User:            cfg.PostgresUser,
		Password:        cfg.PostgresPass,
		DBName:          cfg.PostgresDB,
		SSLMode:         cfg.PostgresSSL,
		MaxConns:        cfg.DBMaxConns,
		MinConns:        cfg.DBMinConns,
		MaxConnLifetime: time.Duration(cfg.DBMaxConnLifetimeMins) * time.Minute,
		MaxConnIdleTime: time.Duration(cfg.DBMaxConnIdleTimeMins) * time.Minute,
	}

	pool, err := database.NewPostgresPool(ctx, &pgCfg)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	logger.Info("connected to PostgreSQL",
		slog.String("host", cfg.PostgresHost),
		slog.Int("port", cfg.PostgresPort),
		slog.String("database", cfg.PostgresDB),
	)
	database.RegisterPoolMetrics(pool, "user")

	// Run database migrations.
	if err := database.RunMigrations(ctx, pool, migrations.FS, logger); err != nil {
		pool.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	logger.Info("database migrations completed")

	// Configure slow query logging.
	if cfg.SlowQueryThresholdMs > 0 {
		database.SetSlowQueryLogging(time.Duration(cfg.SlowQueryThresholdMs)*time.Millisecond, logger)
	}

	// Initialize Kafka producer.
	kafkaCfg := pkgkafka.DefaultProducerConfig(cfg.KafkaBrokers)
	producer := pkgkafka.NewProducer(kafkaCfg, logger)
	logger.Info("kafka producer initialized", slog.Any("brokers", cfg.KafkaBrokers))

	// Parse JWT expiry durations.
	accessExpiry, err := time.ParseDuration(cfg.JWTAccessExpiry)
	if err != nil {
		return nil, fmt.Errorf("parse JWT access expiry %q: %w", cfg.JWTAccessExpiry, err)
	}
	refreshExpiry, err := time.ParseDuration(cfg.JWTRefreshExpiry)
	if err != nil {
		return nil, fmt.Errorf("parse JWT refresh expiry %q: %w", cfg.JWTRefreshExpiry, err)
	}

	// Build the dependency graph.
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, accessExpiry, refreshExpiry)
	userRepo := postgres.NewUserRepository(pool)
	addressRepo := postgres.NewAddressRepository(pool)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(pool)
	wishlistRepo := postgres.NewWishlistRepository(pool)
	eventProducer := event.NewProducer(producer, logger)
	userService := service.NewUserService(userRepo, addressRepo, refreshTokenRepo, jwtManager, eventProducer, logger)

	// Health checks.
	healthHandler := health.NewHandler()
	healthHandler.RegisterCritical("postgres", func(ctx context.Context) error {
		return pool.Ping(ctx)
	})
	healthHandler.RegisterNonCritical("kafka", func(ctx context.Context) error {
		return producer.Ping(ctx)
	})

	// HTTP router.
	router := handler.NewRouter(userService, wishlistRepo, jwtManager, healthHandler, logger, cfg.PprofAllowedCIDRs)

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
		pool:           pool,
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
// 4. PostgreSQL pool
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

	// 4. Close PostgreSQL pool.
	a.pool.Close()

	a.logger.Info("application shutdown complete")
	return errors.Join(errs...)
}
