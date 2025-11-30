package repository

import (
	"context"
	"errors"
	"time"
)

// Common repository errors.
var (
	ErrNotFound = errors.New("record not found")
)

// Default query timeouts.
const (
	// DefaultQueryTimeout is the default timeout for simple queries (SELECT by ID, etc.)
	DefaultQueryTimeout = 5 * time.Second

	// DefaultListQueryTimeout is the timeout for list/paginated queries.
	DefaultListQueryTimeout = 10 * time.Second

	// DefaultWriteTimeout is the timeout for write operations (INSERT, UPDATE, DELETE).
	DefaultWriteTimeout = 10 * time.Second

	// DefaultTransactionTimeout is the timeout for multi-statement transactions.
	DefaultTransactionTimeout = 30 * time.Second
)

// WithQueryTimeout returns a context with the default query timeout.
// If the context already has a deadline shorter than the timeout, the original context is returned.
func WithQueryTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return withTimeout(ctx, DefaultQueryTimeout)
}

// WithListQueryTimeout returns a context with the default list query timeout.
func WithListQueryTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return withTimeout(ctx, DefaultListQueryTimeout)
}

// WithWriteTimeout returns a context with the default write timeout.
func WithWriteTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return withTimeout(ctx, DefaultWriteTimeout)
}

// WithTransactionTimeout returns a context with the default transaction timeout.
func WithTransactionTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return withTimeout(ctx, DefaultTransactionTimeout)
}

// withTimeout adds a timeout to a context, respecting existing deadlines.
func withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	// If context already has a deadline, respect it
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			// Context deadline is sooner, use it
			return ctx, func() {} // no-op cancel since parent controls deadline
		}
	}
	return context.WithTimeout(ctx, timeout)
}
