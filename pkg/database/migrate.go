package database

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrations executes all .up.sql files from the embedded filesystem in sorted order.
// It creates a schema_migrations tracking table and skips already-applied migrations.
// This provides a simple, dependency-free migration runner.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrations embed.FS, logger *slog.Logger) error {
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
	// The embed.go in each service's migrations/ package uses //go:embed *.sql
	// which places files at the root of the FS (not under a subdirectory).
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

		// Read and execute
		content, err := migrations.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		_, err = pool.Exec(ctx, string(content))
		if err != nil {
			return fmt.Errorf("execute migration %s: %w", name, err)
		}

		// Record as applied
		_, err = pool.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", name)
		if err != nil {
			return fmt.Errorf("record migration %s: %w", name, err)
		}

		logger.Info("migration applied successfully", slog.String("version", name))
	}

	return nil
}
