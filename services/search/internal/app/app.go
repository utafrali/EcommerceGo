package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"

	"github.com/utafrali/EcommerceGo/pkg/health"
	"github.com/utafrali/EcommerceGo/services/search/internal/config"
	"github.com/utafrali/EcommerceGo/services/search/internal/engine"
	esengine "github.com/utafrali/EcommerceGo/services/search/internal/engine/elasticsearch"
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
	// Initialize search engine based on configuration.
	var eng engine.SearchEngine
	var esEng *esengine.Engine
	switch cfg.SearchEngine {
	case "elasticsearch":
		var err error
		esEng, err = esengine.New(cfg.ElasticsearchURL, cfg.ElasticsearchIndex, logger)
		if err != nil {
			return nil, fmt.Errorf("init elasticsearch engine: %w", err)
		}
		eng = esEng
		logger.Info("elasticsearch search engine initialized",
			slog.String("url", cfg.ElasticsearchURL),
			slog.String("index", cfg.ElasticsearchIndex),
		)
	default:
		eng = memory.New()
		logger.Info("in-memory search engine initialized")
	}

	// Build the service layer.
	searchService := service.NewSearchService(eng, logger, cfg.ProductServiceURL)

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
	if esEng != nil {
		healthHandler.Register("elasticsearch", esEng.Ping)
	}
	healthHandler.Register("kafka", func(ctx context.Context) error {
		return pkgkafka.PingBrokers(ctx, cfg.KafkaBrokers)
	})

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

	var errs []error

	// Graceful HTTP server shutdown with a 10-second deadline.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
		a.logger.Error("http server shutdown error", slog.String("error", err.Error()))
		errs = append(errs, err)
	}

	// Close Kafka consumers.
	for _, c := range a.consumers {
		if err := c.Close(); err != nil {
			a.logger.Error("kafka consumer close error", slog.String("error", err.Error()))
			errs = append(errs, err)
		}
	}

	a.logger.Info("application shutdown complete")
	return errors.Join(errs...)
}
