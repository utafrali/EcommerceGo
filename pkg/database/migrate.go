package database

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// isConnectionError returns true if the error looks like a transient connection
// problem rather than a SQL syntax or constraint error. Only connection errors
// are retried; SQL errors are returned immediately.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	connPatterns := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"no such host",
		"i/o timeout",
		"connect: connection",
		"dial tcp",
		"EOF",
		"connection timed out",
		"server closed the connection unexpectedly",
		"could not connect",
	}
	for _, p := range connPatterns {
		if strings.Contains(msg, p) {
			return true
		}
	}
	return false
}

// RunMigrations executes all .up.sql files from the embedded filesystem in sorted order.
// It creates a schema_migrations tracking table and skips already-applied migrations.
// Transient connection errors are retried (3 attempts, exponential backoff);
// SQL errors are returned immediately.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrations embed.FS, logger *slog.Logger) error {
	if err := runMigrationsOnce(ctx, pool, migrations, logger); err != nil {
		if !isConnectionError(err) {
			return err
		}
		// Retry on connection errors.
		for attempt := 0; attempt < defaultRetryAttempts-1; attempt++ {
			wait := retryBackoff(attempt)
			logger.Warn("migration failed due to connection error, retrying",
				slog.Int("attempt", attempt+2),
				slog.Int("max_attempts", defaultRetryAttempts),
				slog.Duration("backoff", wait),
				slog.String("error", err.Error()),
			)
			select {
			case <-ctx.Done():
				return fmt.Errorf("run migrations: context cancelled during retry: %w", ctx.Err())
			case <-time.After(wait):
			}
			if err = runMigrationsOnce(ctx, pool, migrations, logger); err == nil {
				return nil
			}
			if !isConnectionError(err) {
				return err
			}
		}
		return fmt.Errorf("run migrations after %d attempts: %w", defaultRetryAttempts, err)
	}
	return nil
}

// runMigrationsOnce executes one attempt of the migration sequence.
func runMigrationsOnce(ctx context.Context, pool *pgxpool.Pool, migrations embed.FS, logger *slog.Logger) error {
	// Create tracking table
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	// Read all migration files from the root of the embedded FS.
	entries, err := migrations.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	// Sort by filename
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		name := entry.Name()
		// Only process .up.sql files
		if entry.IsDir() || len(name) < 7 || name[len(name)-7:] != ".up.sql" {
			continue
		}

		// Check if already applied
		var exists bool
		err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", name).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if exists {
			logger.Info("migration already applied, skipping", slog.String("version", name))
			continue
		}

		// Read migration file.
		content, err := migrations.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		// Execute migration and record version inside a transaction so
		// multi-statement migrations are atomic.
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx for migration %s: %w", name, err)
		}

		if _, err := tx.Exec(ctx, string(content)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("execute migration %s: %w", name, err)
		}

		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", name); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", name, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}

		logger.Info("migration applied successfully", slog.String("version", name))
	}

	return nil
}
