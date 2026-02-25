package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"

	"github.com/utafrali/EcommerceGo/pkg/health"
	"github.com/utafrali/EcommerceGo/services/search/internal/config"
	"github.com/utafrali/EcommerceGo/services/search/internal/engine/memory"
	"github.com/utafrali/EcommerceGo/services/search/internal/event"
	handler "github.com/utafrali/EcommerceGo/services/search/internal/handler/http"
	"github.com/utafrali/EcommerceGo/services/search/internal/service"
)

// App wires together all dependencies and runs the search service.
type App struct {
	cfg        *config.Config
	logger     *slog.Logger
	consumers  []*pkgkafka.Consumer
	httpServer *http.Server
}

// NewApp creates a new application instance, initializing all dependencies.
func NewApp(cfg *config.Config, logger *slog.Logger) (*App, error) {
	// Initialize in-memory search engine.
	eng := memory.New()
	logger.Info("in-memory search engine initialized")

	// Build the service layer.
	searchService := service.NewSearchService(eng, logger)

	// Initialize Kafka consumers for product events.
	eventConsumer := event.NewConsumer(searchService, logger)

	topics := []string{
		event.TopicProductCreated,
		event.TopicProductUpdated,
		event.TopicProductDeleted,
	}

	var consumers []*pkgkafka.Consumer
	for _, topic := range topics {
		consumerCfg := pkgkafka.ConsumerConfig{
			Brokers:  cfg.KafkaBrokers,
			GroupID:  "search-service",
			Topic:    topic,
			MinBytes: 1,
			MaxBytes: 10e6, // 10 MB
		}
		c := pkgkafka.NewConsumer(consumerCfg, eventConsumer.Handle, logger)
		consumers = append(consumers, c)
	}
	logger.Info("kafka consumers initialized",
		slog.Any("brokers", cfg.KafkaBrokers),
		slog.Int("topic_count", len(topics)),
	)

	// Health checks.
	healthHandler := health.NewHandler()
	// No database to check; the in-memory engine is always ready.

	// HTTP router.
	router := handler.NewRouter(searchService, healthHandler, logger)

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
		consumers:  consumers,
		httpServer: httpServer,
	}, nil
}

// Run starts the HTTP server and Kafka consumers, blocking until the context is canceled.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1+len(a.consumers))

	// Start Kafka consumers in background goroutines.
	for _, c := range a.consumers {
		c := c
		go func() {
			if err := c.Start(ctx); err != nil {
				errCh <- fmt.Errorf("kafka consumer: %w", err)
			}
		}()
	}

	// Start HTTP server.
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
	for _, c := range a.consumers {
		if err := c.Close(); err != nil {
			a.logger.Error("kafka consumer close error", slog.String("error", err.Error()))
		}
	}

	a.logger.Info("application shutdown complete")
	return nil
}
