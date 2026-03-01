package database

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresConfig holds PostgreSQL connection configuration.
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string

	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// DefaultPostgresConfig returns sensible defaults for PostgreSQL connection pool.
func DefaultPostgresConfig() PostgresConfig {
	return PostgresConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "ecommerce",
		Password:        "ecommerce_secret",
		DBName:          "ecommerce",
		SSLMode:         "disable",
		MaxConns:        25,
		MinConns:        5,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}
}

// DSN returns the PostgreSQL connection string.
func (c *PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode,
	)
}

const (
	defaultRetryAttempts = 3
	defaultRetryBaseWait = 1 * time.Second
	retryJitterFraction  = 0.25
)

// retryBackoff returns the backoff duration for the given attempt (0-indexed)
// with ±25% jitter. Base delays: 1s, 2s, 4s.
func retryBackoff(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	base := defaultRetryBaseWait << attempt                                               // 1s, 2s, 4s
	jitter := time.Duration(float64(base) * retryJitterFraction * (2*rand.Float64() - 1)) // #nosec G404 -- non-cryptographic jitter for retry backoff
	return base + jitter
}

// NewPostgresPool creates a new connection pool for PostgreSQL with startup
// retry logic (3 attempts, 1s/2s/4s exponential backoff with ±25% jitter).
func NewPostgresPool(ctx context.Context, cfg *PostgresConfig) (*pgxpool.Pool, error) {
	return NewPostgresPoolWithLogger(ctx, cfg, nil)
}

// NewPostgresPoolWithLogger is like NewPostgresPool but accepts an optional logger
// for retry warning messages.
func NewPostgresPoolWithLogger(ctx context.Context, cfg *PostgresConfig, logger *slog.Logger) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	var lastErr error
	for attempt := 0; attempt < defaultRetryAttempts; attempt++ {
		pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
		if err != nil {
			lastErr = err
			if attempt < defaultRetryAttempts-1 {
				wait := retryBackoff(attempt)
				if logger != nil {
					logger.Warn("postgres connection failed, retrying",
						slog.Int("attempt", attempt+1),
						slog.Int("max_attempts", defaultRetryAttempts),
						slog.Duration("backoff", wait),
						slog.String("error", err.Error()),
					)
				}
				select {
				case <-ctx.Done():
					return nil, fmt.Errorf("create postgres pool: context canceled during retry: %w", ctx.Err())
				case <-time.After(wait):
				}
			}
			continue
		}

		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			lastErr = err
			if attempt < defaultRetryAttempts-1 {
				wait := retryBackoff(attempt)
				if logger != nil {
					logger.Warn("postgres ping failed, retrying",
						slog.Int("attempt", attempt+1),
						slog.Int("max_attempts", defaultRetryAttempts),
						slog.Duration("backoff", wait),
						slog.String("error", err.Error()),
					)
				}
				select {
				case <-ctx.Done():
					return nil, fmt.Errorf("ping postgres: context canceled during retry: %w", ctx.Err())
				case <-time.After(wait):
				}
			}
			continue
		}

		return pool, nil
	}

	return nil, fmt.Errorf("connect to postgres after %d attempts: %w", defaultRetryAttempts, lastErr)
}
