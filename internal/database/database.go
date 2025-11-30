// Package database provides PostgreSQL connection management using pgx.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/config"
)

// DB wraps the pgx connection pool with additional functionality.
type DB struct {
	Pool        *pgxpool.Pool
	TxManager   *TxManager
	QueryLogger *QueryLogger
	logger      *zap.Logger
}

// New creates a new database connection pool.
func New(ctx context.Context, cfg *config.DatabaseConfig, logger *zap.Logger) (*DB, error) {
	return NewWithQueryLogger(ctx, cfg, nil, logger)
}

// NewWithQueryLogger creates a new database connection pool with optional query logging.
func NewWithQueryLogger(ctx context.Context, cfg *config.DatabaseConfig, queryLoggerCfg *QueryLoggerConfig, logger *zap.Logger) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configure pool settings
	poolConfig.MaxConns = int32(cfg.MaxConnections)
	poolConfig.MinConns = int32(cfg.MaxIdleConnections)
	poolConfig.MaxConnLifetime = cfg.ConnectionMaxLifetime
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	// Create query logger and attach to pool config
	var queryLogger *QueryLogger
	if queryLoggerCfg != nil {
		queryLogger = NewQueryLogger(queryLoggerCfg, logger)
		poolConfig.ConnConfig.Tracer = queryLogger
		logger.Info("query logging enabled",
			zap.Duration("slow_threshold", queryLoggerCfg.SlowQueryThreshold),
			zap.Duration("very_slow_threshold", queryLoggerCfg.VerySlowQueryThreshold),
			zap.Bool("log_all_queries", queryLoggerCfg.LogAllQueries),
		)
	}

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("database connection established",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Name),
		zap.Int("max_connections", cfg.MaxConnections),
	)

	db := &DB{
		Pool:        pool,
		QueryLogger: queryLogger,
		logger:      logger,
	}
	db.TxManager = NewTxManager(pool, logger)

	return db, nil
}

// Close closes the database connection pool.
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
		db.logger.Info("database connection closed")
	}
}

// Health checks the database connection health.
func (db *DB) Health(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Ping checks the database connection (implements handler.HealthChecker interface).
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Stats returns current pool statistics.
func (db *DB) Stats() *pgxpool.Stat {
	return db.Pool.Stat()
}
