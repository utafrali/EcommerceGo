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
	"github.com/utafrali/EcommerceGo/services/inventory/internal/config"
	"github.com/utafrali/EcommerceGo/services/inventory/internal/event"
	handler "github.com/utafrali/EcommerceGo/services/inventory/internal/handler/http"
	"github.com/utafrali/EcommerceGo/services/inventory/internal/repository/postgres"
	"github.com/utafrali/EcommerceGo/services/inventory/internal/service"
)

// App wires together all dependencies and runs the inventory service.
type App struct {
	cfg              *config.Config
	logger           *slog.Logger
	pool             *pgxpool.Pool
	producer         *pkgkafka.Producer
	httpServer       *http.Server
	orderConfirmed   *pkgkafka.Consumer
	orderCanceled    *pkgkafka.Consumer
	inventoryService *service.InventoryService
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
	producer := pkgkafka.NewProducer(kafkaCfg, logger)
	logger.Info("kafka producer initialized", slog.Any("brokers", cfg.KafkaBrokers))

	// Build the dependency graph.
	repo := postgres.NewInventoryRepository(pool)
	eventProducer := event.NewProducer(producer, logger)
	inventoryService := service.NewInventoryService(repo, repo, pool, eventProducer, logger, cfg.ReservationTTL)

	// Set up Kafka consumers for order events.
	eventConsumer := event.NewConsumer(inventoryService, logger)

	orderConfirmedConsumer := pkgkafka.NewConsumer(pkgkafka.ConsumerConfig{
		Brokers:  cfg.KafkaBrokers,
		GroupID:  "inventory-service-order-confirmed",
		Topic:    event.TopicOrderConfirmed,
		MinBytes: 1,
		MaxBytes: 10e6,
	}, eventConsumer.HandleOrderConfirmed, logger)

	orderCanceledConsumer := pkgkafka.NewConsumer(pkgkafka.ConsumerConfig{
		Brokers:  cfg.KafkaBrokers,
		GroupID:  "inventory-service-order-canceled",
		Topic:    event.TopicOrderCanceled,
		MinBytes: 1,
		MaxBytes: 10e6,
	}, eventConsumer.HandleOrderCanceled, logger)

	// Health checks.
	healthHandler := health.NewHandler()
	healthHandler.Register("postgres", func(ctx context.Context) error {
		return pool.Ping(ctx)
	})

	// HTTP router.
	router := handler.NewRouter(inventoryService, healthHandler, logger)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &App{
		cfg:              cfg,
		logger:           logger,
		pool:             pool,
		producer:         producer,
		httpServer:       httpServer,
		orderConfirmed:   orderConfirmedConsumer,
		orderCanceled:    orderCanceledConsumer,
		inventoryService: inventoryService,
	}, nil
}

// Run starts the HTTP server, Kafka consumers, and background jobs, then blocks until the context is canceled.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 3)

	// Start HTTP server.
	go func() {
		a.logger.Info("starting HTTP server",
			slog.String("addr", a.httpServer.Addr),
		)
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("http server: %w", err)
		}
	}()

	// Start Kafka consumers.
	go func() {
		if err := a.orderConfirmed.Start(ctx); err != nil {
			errCh <- fmt.Errorf("order confirmed consumer: %w", err)
		}
	}()

	go func() {
		if err := a.orderCanceled.Start(ctx); err != nil {
			errCh <- fmt.Errorf("order canceled consumer: %w", err)
		}
	}()

	// Start background reservation cleanup job.
	go a.runReservationCleanup(ctx)

	select {
	case <-ctx.Done():
		a.logger.Info("shutdown signal received")
	case err := <-errCh:
		return err
	}

	return a.Shutdown()
}

// runReservationCleanup periodically cleans up expired reservations.
func (a *App) runReservationCleanup(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			released, err := a.inventoryService.CleanExpiredReservations(ctx)
			if err != nil {
				a.logger.Error("reservation cleanup error", slog.String("error", err.Error()))
			} else if released > 0 {
				a.logger.Info("expired reservations cleaned", slog.Int("released", released))
			}
		}
	}
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
	if err := a.orderConfirmed.Close(); err != nil {
		a.logger.Error("order confirmed consumer close error", slog.String("error", err.Error()))
	}
	if err := a.orderCanceled.Close(); err != nil {
		a.logger.Error("order canceled consumer close error", slog.String("error", err.Error()))
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
