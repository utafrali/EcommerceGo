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
	"github.com/utafrali/EcommerceGo/pkg/tracing"
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
	cfg            *config.Config
	logger         *slog.Logger
	consumers      []*pkgkafka.Consumer
	httpServer     *http.Server
	tracerShutdown func(context.Context) error
	cancelESRetry  context.CancelFunc
}

// NewApp creates a new application instance, initializing all dependencies.
func NewApp(cfg *config.Config, logger *slog.Logger) (*App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize OpenTelemetry tracing.
	tracerShutdown, err := tracing.InitTracer(ctx, tracing.Config{
		ServiceName:    "search",
		ServiceVersion: "0.1.0",
		Environment:    cfg.Environment,
		OTLPEndpoint:   cfg.OTELEndpoint,
		SampleRate:     cfg.OTELSampleRate,
		Enabled:        cfg.OTELEnabled,
	})
	if err != nil {
		return nil, fmt.Errorf("init tracer: %w", err)
	}

	// Initialize search engine based on configuration.
	var eng engine.SearchEngine
	var esEng *esengine.Engine
	var esDegraded bool
	switch cfg.SearchEngine {
	case "elasticsearch":
		var err error
		esEng, err = esengine.New(cfg.ElasticsearchURL, cfg.ElasticsearchIndex, logger)
		if err != nil {
			// Start in degraded mode with in-memory fallback.
			logger.Warn("elasticsearch unavailable at startup, starting in degraded mode with in-memory engine",
				slog.String("url", cfg.ElasticsearchURL),
				slog.String("error", err.Error()),
			)
			eng = memory.New()
			esDegraded = true
		} else {
			eng = esEng
			logger.Info("elasticsearch search engine initialized",
				slog.String("url", cfg.ElasticsearchURL),
				slog.String("index", cfg.ElasticsearchIndex),
			)
		}
	default:
		eng = memory.New()
		logger.Info("in-memory search engine initialized")
	}

	// Build the service layer.
	searchService := service.NewSearchService(eng, logger, cfg.ProductServiceURL)

	// If ES was configured but unavailable, launch background goroutine to retry connection.
	var cancelESRetry context.CancelFunc
	if esDegraded {
		var esCtx context.Context
		esCtx, cancelESRetry = context.WithCancel(context.Background())
		go retryElasticsearch(esCtx, cfg, searchService, logger)
	}

	// Initialize Kafka consumers for product events.
	eventConsumer := event.NewConsumer(searchService, logger)
	idempotencyStore := pkgkafka.NewMemoryIdempotencyStore(24 * time.Hour)

	topics := []string{
		event.TopicProductCreated,
		event.TopicProductUpdated,
		event.TopicProductDeleted,
	}

	var consumers []*pkgkafka.Consumer
	for _, topic := range topics {
		consumerCfg := pkgkafka.ConsumerConfig{
			Brokers:   cfg.KafkaBrokers,
			GroupID:   "search-service",
			Topic:     topic,
			MinBytes:  1,
			MaxBytes:  10e6, // 10 MB
			EnableDLQ: true,
		}
		c := pkgkafka.NewConsumer(consumerCfg, pkgkafka.IdempotentHandler(idempotencyStore, eventConsumer.Handle, logger), logger)
		consumers = append(consumers, c)
	}
	logger.Info("kafka consumers initialized",
		slog.Any("brokers", cfg.KafkaBrokers),
		slog.Int("topic_count", len(topics)),
	)

	// Health checks.
	healthHandler := health.NewHandler()
	healthHandler.RegisterNonCritical("search_engine", func(ctx context.Context) error {
		// Check the current engine type; if we're on in-memory fallback
		// because ES was configured, report the engine as degraded.
		engineType := searchService.SearchEngineType()
		if cfg.SearchEngine == "elasticsearch" && engineType != "*elasticsearch.Engine" {
			return fmt.Errorf("running on in-memory fallback (elasticsearch unavailable)")
		}
		if esEng != nil {
			return esEng.Ping(ctx)
		}
		return nil
	})
	healthHandler.RegisterNonCritical("kafka", func(ctx context.Context) error {
		return pkgkafka.PingBrokers(ctx, cfg.KafkaBrokers)
	})

	// HTTP router.
	router := handler.NewRouter(searchService, healthHandler, logger, cfg.PprofAllowedCIDRs)

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
		consumers:      consumers,
		httpServer:     httpServer,
		tracerShutdown: tracerShutdown,
		cancelESRetry:  cancelESRetry,
	}, nil
}

// retryElasticsearch runs in the background and attempts to connect to
// Elasticsearch every 30 seconds. When ES becomes available, it hot-swaps the
// engine in the search service.
func retryElasticsearch(ctx context.Context, cfg *config.Config, svc *service.SearchService, logger *slog.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logger.Info("retrying elasticsearch connection",
				slog.String("url", cfg.ElasticsearchURL),
			)
			esEng, err := esengine.New(cfg.ElasticsearchURL, cfg.ElasticsearchIndex, logger)
			if err != nil {
				logger.Warn("elasticsearch still unavailable",
					slog.String("error", err.Error()),
				)
				continue
			}
			// ES is available — hot-swap the engine.
			svc.SwapEngine(esEng)
			logger.Info("elasticsearch connected, engine hot-swapped from in-memory fallback",
				slog.String("url", cfg.ElasticsearchURL),
				slog.String("index", cfg.ElasticsearchIndex),
			)
			return
		}
	}
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

// Shutdown gracefully stops all components in the correct order:
// 1. HTTP server (drain in-flight requests)
// 2. Tracer (flush pending spans from drained requests)
// 3. Kafka consumers
// 4. ES retry goroutine
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
	for _, c := range a.consumers {
		if err := c.Close(); err != nil {
			a.logger.Error("kafka consumer close error", slog.String("error", err.Error()))
			errs = append(errs, err)
		}
	}

	// 4. Cancel ES retry goroutine if running.
	if a.cancelESRetry != nil {
		a.cancelESRetry()
	}

	a.logger.Info("application shutdown complete")
	return errors.Join(errs...)
}
