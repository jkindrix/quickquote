// Package database provides database connectivity and utilities.
package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// TxManager provides transaction management capabilities.
type TxManager struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewTxManager creates a new transaction manager.
func NewTxManager(pool *pgxpool.Pool, logger *zap.Logger) *TxManager {
	return &TxManager{
		pool:   pool,
		logger: logger,
	}
}

// TxFunc is a function that runs within a transaction.
// If it returns an error, the transaction is rolled back.
// If it returns nil, the transaction is committed.
type TxFunc func(ctx context.Context, tx pgx.Tx) error

// WithTransaction executes the given function within a transaction.
// If the function returns an error, the transaction is rolled back.
// If the function completes successfully, the transaction is committed.
func (tm *TxManager) WithTransaction(ctx context.Context, fn TxFunc) error {
	return tm.WithTransactionOptions(ctx, pgx.TxOptions{}, fn)
}

// WithTransactionOptions executes the given function within a transaction with custom options.
func (tm *TxManager) WithTransactionOptions(ctx context.Context, opts pgx.TxOptions, fn TxFunc) error {
	tx, err := tm.pool.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Defer rollback - this will be a no-op if commit succeeds
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			tm.logger.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	// Execute the function
	if err := fn(ctx, tx); err != nil {
		tm.logger.Debug("transaction rolling back due to error", zap.Error(err))
		return err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WithReadOnlyTransaction executes the given function within a read-only transaction.
// This is useful for consistent reads across multiple queries.
func (tm *TxManager) WithReadOnlyTransaction(ctx context.Context, fn TxFunc) error {
	return tm.WithTransactionOptions(ctx, pgx.TxOptions{
		AccessMode: pgx.ReadOnly,
	}, fn)
}

// WithSerializableTransaction executes the given function within a serializable transaction.
// This provides the strongest isolation level but may have performance implications.
func (tm *TxManager) WithSerializableTransaction(ctx context.Context, fn TxFunc) error {
	return tm.WithTransactionOptions(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	}, fn)
}

// txContextKey is the context key for transactions.
type txContextKey struct{}

// ContextWithTx adds a transaction to the context.
func ContextWithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

// TxFromContext retrieves a transaction from the context.
// Returns nil if no transaction is present.
func TxFromContext(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(txContextKey{}).(pgx.Tx); ok {
		return tx
	}
	return nil
}

// Querier is an interface that both pgx.Tx and *pgxpool.Pool implement.
// This allows functions to work with either a transaction or direct pool access.
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
}

// GetQuerier returns the transaction from context if present, otherwise the pool.
func (tm *TxManager) GetQuerier(ctx context.Context) Querier {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return tm.pool
}

// WithTransactionContext executes fn within a transaction and stores the tx in context.
// This allows nested functions to participate in the same transaction.
func (tm *TxManager) WithTransactionContext(ctx context.Context, fn func(ctx context.Context) error) error {
	// Check if already in a transaction
	if tx := TxFromContext(ctx); tx != nil {
		// Already in a transaction, just execute the function
		return fn(ctx)
	}

	// Start a new transaction
	return tm.WithTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		txCtx := ContextWithTx(ctx, tx)
		return fn(txCtx)
	})
}

// RetryableTransaction executes a transaction with automatic retry on serialization failures.
// This is useful for high-contention scenarios.
func (tm *TxManager) RetryableTransaction(ctx context.Context, maxRetries int, fn TxFunc) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			tm.logger.Debug("retrying transaction",
				zap.Int("attempt", attempt),
				zap.Error(lastErr),
			)
		}

		err := tm.WithTransaction(ctx, fn)
		if err == nil {
			return nil
		}

		// Check if this is a retryable error (serialization failure)
		if !isRetryableError(err) {
			return err
		}

		lastErr = err
	}

	return fmt.Errorf("transaction failed after %d retries: %w", maxRetries, lastErr)
}

// isRetryableError checks if an error is a serialization failure that can be retried.
func isRetryableError(err error) bool {
	// PostgreSQL error codes for serialization failures
	// 40001 = serialization_failure
	// 40P01 = deadlock_detected
	errStr := err.Error()
	return contains(errStr, "40001") || contains(errStr, "40P01") ||
		contains(errStr, "serialization") || contains(errStr, "deadlock")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
