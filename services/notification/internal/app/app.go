package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/utafrali/EcommerceGo/pkg/database"
	"github.com/utafrali/EcommerceGo/pkg/health"
	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/notification/internal/config"
	"github.com/utafrali/EcommerceGo/services/notification/internal/event"
	handler "github.com/utafrali/EcommerceGo/services/notification/internal/handler/http"
	"github.com/utafrali/EcommerceGo/services/notification/internal/repository/postgres"
	"github.com/utafrali/EcommerceGo/services/notification/internal/sender"
	mocksender "github.com/utafrali/EcommerceGo/services/notification/internal/sender/mock"
	"github.com/utafrali/EcommerceGo/services/notification/internal/service"
)

// App wires together all dependencies and runs the notification service.
type App struct {
	cfg        *config.Config
	logger     *slog.Logger
	pool       *pgxpool.Pool
	producer   *pkgkafka.Producer
	consumers  []*pkgkafka.Consumer
	httpServer *http.Server
}

// NewApp creates a new application instance, initializing all dependencies.
func NewApp(cfg *config.Config, logger *slog.Logger) (*App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize PostgreSQL connection pool.
	pgCfg := database.PostgresConfig{
		Host:            cfg.PostgresHost,
		Port:            cfg.PostgresPort,
		User:            cfg.PostgresUser,
		Password:        cfg.PostgresPass,
		DBName:          cfg.PostgresDB,
		SSLMode:         cfg.PostgresSSL,
		MaxConns:        25,
		MinConns:        5,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
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

	// Initialize Kafka producer.
	kafkaCfg := pkgkafka.DefaultProducerConfig(cfg.KafkaBrokers)
	kafkaProducer := pkgkafka.NewProducer(kafkaCfg, logger)
	logger.Info("kafka producer initialized", slog.Any("brokers", cfg.KafkaBrokers))

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
	consumerHandler := event.NewConsumerHandler(logger)
	consumers := event.NewConsumers(cfg.KafkaBrokers, consumerHandler, logger)

	// Health checks.
	healthHandler := health.NewHandler()
	healthHandler.Register("postgres", func(ctx context.Context) error {
		return pool.Ping(ctx)
	})

	// HTTP router.
	router := handler.NewRouter(notificationService, healthHandler, logger)

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
		pool:       pool,
		producer:   kafkaProducer,
		consumers:  consumers,
		httpServer: httpServer,
	}, nil
}

// Run starts the HTTP server and Kafka consumers, then blocks until the context is canceled.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	// Start Kafka consumers.
	for _, consumer := range a.consumers {
		c := consumer
		go func() {
			if err := c.Start(ctx); err != nil {
				a.logger.Error("kafka consumer error", slog.String("error", err.Error()))
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

// Shutdown gracefully stops all components.
func (a *App) Shutdown() error {
	a.logger.Info("shutting down application...")

	// Graceful HTTP server shutdown with a 10-second deadline.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
		a.logger.Error("http server shutdown error", slog.String("error", err.Error()))
	}

	// Close Kafka consumers.
	for _, consumer := range a.consumers {
		if err := consumer.Close(); err != nil {
			a.logger.Error("kafka consumer close error", slog.String("error", err.Error()))
		}
	}

	// Close Kafka producer.
	if err := a.producer.Close(); err != nil {
		a.logger.Error("kafka producer close error", slog.String("error", err.Error()))
	}

	// Close PostgreSQL pool.
	a.pool.Close()

	a.logger.Info("application shutdown complete")
	return nil
}
