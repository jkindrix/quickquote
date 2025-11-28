// Package database provides PostgreSQL connection management and migration support.
package database

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Migrator handles database schema migrations.
type Migrator struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewMigrator creates a new migrator instance.
func NewMigrator(pool *pgxpool.Pool, logger *zap.Logger) *Migrator {
	return &Migrator{
		pool:   pool,
		logger: logger,
	}
}

// MigrateFromFS runs all pending migrations from an embedded filesystem.
func (m *Migrator) MigrateFromFS(ctx context.Context, fs embed.FS, dir string) error {
	// Ensure schema_migrations table exists
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Read migration files from embedded FS
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Collect and sort .up.sql files
	var migrations []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") {
			migrations = append(migrations, name)
		}
	}
	sort.Strings(migrations)

	// Apply pending migrations
	for _, filename := range migrations {
		version := extractVersion(filename)
		if version == 0 {
			m.logger.Warn("skipping migration with invalid version", zap.String("file", filename))
			continue
		}

		if applied[version] {
			continue
		}

		m.logger.Info("applying migration", zap.String("file", filename), zap.Int("version", version))

		content, err := fs.ReadFile(filepath.Join(dir, filename))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", filename, err)
		}

		if err := m.applyMigration(ctx, version, filename, string(content)); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", filename, err)
		}

		m.logger.Info("migration applied successfully", zap.String("file", filename))
	}

	return nil
}

// MigrateFromDir runs all pending migrations from a directory on disk.
func (m *Migrator) MigrateFromDir(ctx context.Context, dir string) error {
	// Ensure schema_migrations table exists
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Read migration files
	pattern := filepath.Join(dir, "*.up.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob migrations: %w", err)
	}
	sort.Strings(files)

	// Apply pending migrations
	for _, path := range files {
		filename := filepath.Base(path)
		version := extractVersion(filename)
		if version == 0 {
			m.logger.Warn("skipping migration with invalid version", zap.String("file", filename))
			continue
		}

		if applied[version] {
			continue
		}

		m.logger.Info("applying migration", zap.String("file", filename), zap.Int("version", version))

		content, err := readFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", filename, err)
		}

		if err := m.applyMigration(ctx, version, filename, string(content)); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", filename, err)
		}

		m.logger.Info("migration applied successfully", zap.String("file", filename))
	}

	return nil
}

// ensureMigrationsTable creates the schema_migrations table if it doesn't exist.
func (m *Migrator) ensureMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			filename TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`
	_, err := m.pool.Exec(ctx, query)
	return err
}

// getAppliedMigrations returns a map of applied migration versions.
func (m *Migrator) getAppliedMigrations(ctx context.Context) (map[int]bool, error) {
	rows, err := m.pool.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

// applyMigration runs a single migration in a transaction.
func (m *Migrator) applyMigration(ctx context.Context, version int, filename, sql string) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Execute migration SQL
	if _, err := tx.Exec(ctx, sql); err != nil {
		return fmt.Errorf("migration SQL failed: %w", err)
	}

	// Record migration
	_, err = tx.Exec(ctx,
		"INSERT INTO schema_migrations (version, filename) VALUES ($1, $2)",
		version, filename,
	)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit(ctx)
}

// extractVersion extracts the version number from a migration filename.
// Expected format: NNN_description.up.sql (e.g., 001_initial.up.sql)
func extractVersion(filename string) int {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) < 2 {
		return 0
	}
	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return version
}

// readFile reads file content from disk (for non-embedded migrations).
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
