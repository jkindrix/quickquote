package database

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestTxFromContext_NoTx(t *testing.T) {
	ctx := context.Background()
	tx := TxFromContext(ctx)
	if tx != nil {
		t.Error("expected nil tx from context without transaction")
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{"serialization failure", errors.New("ERROR: 40001 serialization_failure"), true},
		{"deadlock detected", errors.New("ERROR: 40P01 deadlock_detected"), true},
		{"serialization in message", errors.New("serialization failure occurred"), true},
		{"deadlock in message", errors.New("deadlock detected"), true},
		{"normal error", errors.New("connection refused"), false},
		{"constraint violation", errors.New("unique_violation"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			if result != tt.retryable {
				t.Errorf("isRetryableError(%v) = %v, want %v", tt.err, result, tt.retryable)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "foo", false},
		{"", "foo", false},
		{"foo", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

// MockTx is a mock transaction for testing.
type MockTx struct {
	committed  bool
	rolledBack bool
}

func (m *MockTx) Commit(ctx context.Context) error {
	m.committed = true
	return nil
}

func (m *MockTx) Rollback(ctx context.Context) error {
	if m.committed {
		return pgx.ErrTxClosed
	}
	m.rolledBack = true
	return nil
}

func (m *MockTx) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (m *MockTx) Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (m *MockTx) QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row {
	return nil
}

func TestContextWithTx(t *testing.T) {
	ctx := context.Background()

	// Test that nil transaction returns nil
	txCtx := ContextWithTx(ctx, nil)
	if TxFromContext(txCtx) != nil {
		t.Error("expected nil tx from context with nil transaction")
	}
}
