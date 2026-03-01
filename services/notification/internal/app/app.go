package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/utafrali/EcommerceGo/pkg/database"
	"github.com/utafrali/EcommerceGo/pkg/health"
	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/pkg/tracing"
	"github.com/utafrali/EcommerceGo/services/notification/internal/config"
	"github.com/utafrali/EcommerceGo/services/notification/internal/event"
	handler "github.com/utafrali/EcommerceGo/services/notification/internal/handler/http"
	"github.com/utafrali/EcommerceGo/services/notification/internal/repository/postgres"
	"github.com/utafrali/EcommerceGo/services/notification/internal/sender"
	mocksender "github.com/utafrali/EcommerceGo/services/notification/internal/sender/mock"
	"github.com/utafrali/EcommerceGo/services/notification/internal/service"
	"github.com/utafrali/EcommerceGo/services/notification/migrations"
)

// notificationSenderAdapter adapts *service.NotificationService to the
// event.NotificationSender interface, breaking the import cycle between
// the event and service packages.
type notificationSenderAdapter struct {
	svc *service.NotificationService
}

func (a *notificationSenderAdapter) SendNotification(ctx context.Context, input *event.SendInput) error {
	_, err := a.svc.SendNotification(ctx, &service.SendNotificationInput{
		UserID:   input.UserID,
		Type:     input.Type,
		Channel:  input.Channel,
		Subject:  input.Subject,
		Body:     input.Body,
		Priority: input.Priority,
		Metadata: input.Metadata,
	})
	return err
}

// App wires together all dependencies and runs the notification service.
type App struct {
	cfg            *config.Config
	logger         *slog.Logger
	pool           *pgxpool.Pool
	producer       *pkgkafka.Producer
	consumers      []*pkgkafka.Consumer
	httpServer     *http.Server
	tracerShutdown func(context.Context) error
}

// NewApp creates a new application instance, initializing all dependencies.
func NewApp(cfg *config.Config, logger *slog.Logger) (*App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize OpenTelemetry tracing.
	tracerShutdown, err := tracing.InitTracer(ctx, tracing.Config{
		ServiceName:    "notification",
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
	database.RegisterPoolMetrics(pool, "notification")

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

	// Initialize Kafka producer with connection validation and retry.
	kafkaCfg := pkgkafka.DefaultProducerConfig(cfg.KafkaBrokers)
	kafkaProducer := pkgkafka.NewProducer(kafkaCfg, logger)
	if err := pingKafkaWithRetry(ctx, kafkaProducer, logger); err != nil {
		logger.Warn("kafka producer ping failed after retries, continuing in degraded mode",
			slog.String("error", err.Error()),
		)
	} else {
		logger.Info("kafka producer initialized", slog.Any("brokers", cfg.KafkaBrokers))
	}

	// Build the dependency graph.
	repo := postgres.NewNotificationRepository(pool)
	eventProducer := event.NewProducer(kafkaProducer, logger)

	// Initialize senders (mock senders for all channels).
	senders := map[string]sender.Sender{
		"email": mocksender.NewMockSender("email", logger),
		"sms":   mocksender.NewMockSender("sms", logger),
		"push":  mocksender.NewMockSender("push", logger),
	}

	notificationService := service.NewNotificationService(repo, senders, eventProducer, logger)

	// Kafka event consumers.
	senderAdapter := &notificationSenderAdapter{svc: notificationService}
	consumerHandler := event.NewConsumerHandler(senderAdapter, logger)
	idempotencyStore := pkgkafka.NewMemoryIdempotencyStore(24 * time.Hour)
	consumers := event.NewConsumers(cfg.KafkaBrokers, consumerHandler, idempotencyStore, logger)

	// Health checks.
	healthHandler := health.NewHandler()
	healthHandler.RegisterCritical("postgres", func(ctx context.Context) error {
		return pool.Ping(ctx)
	})
	healthHandler.RegisterNonCritical("kafka", func(ctx context.Context) error {
		return kafkaProducer.Ping(ctx)
	})

	// HTTP router.
	router := handler.NewRouter(notificationService, healthHandler, logger, cfg.PprofAllowedCIDRs)

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
		producer:       kafkaProducer,
		consumers:      consumers,
		httpServer:     httpServer,
		tracerShutdown: tracerShutdown,
	}, nil
}

// Run starts the HTTP server and Kafka consumers, then blocks until the context is canceled.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1+len(a.consumers))

	// Start Kafka consumers.
	for _, consumer := range a.consumers {
		c := consumer
		go func() {
			if err := c.Start(ctx); err != nil {
				errCh <- fmt.Errorf("kafka consumer: %w", err)
			}
		}()
	}

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
// 3. Kafka consumers
// 4. Kafka producer
// 5. PostgreSQL pool
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

	// 3. Close Kafka consumers (2s budget).
	for _, consumer := range a.consumers {
		if err := consumer.Close(); err != nil {
			a.logger.Error("kafka consumer close error", slog.String("error", err.Error()))
			errs = append(errs, err)
		}
	}

	// 4. Close Kafka producer (2s budget).
	if err := a.producer.Close(); err != nil {
		a.logger.Error("kafka producer close error", slog.String("error", err.Error()))
		errs = append(errs, err)
	}

	// 5. Close PostgreSQL pool.
	a.pool.Close()

	a.logger.Info("application shutdown complete")
	return errors.Join(errs...)
}

// pingKafkaWithRetry attempts to ping the Kafka producer with exponential
// backoff (3 attempts, 1s/2s/4s with ±25% jitter).
func pingKafkaWithRetry(ctx context.Context, producer *pkgkafka.Producer, logger *slog.Logger) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if err := producer.Ping(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if attempt < 2 {
			base := time.Duration(1<<uint(attempt)) * time.Second
			jitter := time.Duration(float64(base) * 0.25 * (2*rand.Float64() - 1)) // #nosec G404 -- non-cryptographic jitter for retry backoff
			wait := base + jitter
			logger.Warn("kafka producer ping failed, retrying",
				slog.Int("attempt", attempt+1),
				slog.Int("max_attempts", 3),
				slog.Duration("backoff", wait),
				slog.String("error", lastErr.Error()),
			)
			select {
			case <-ctx.Done():
				return fmt.Errorf("kafka ping: context canceled during retry: %w", ctx.Err())
			case <-time.After(wait):
			}
		}
	}
	return fmt.Errorf("kafka producer ping failed after 3 attempts: %w", lastErr)
}
