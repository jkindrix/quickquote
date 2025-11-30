// Package database provides PostgreSQL connection management using pgx.
package database

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// QueryLoggerConfig configures query logging behavior.
type QueryLoggerConfig struct {
	// SlowQueryThreshold is the duration above which queries are logged as slow.
	// Queries taking longer than this will be logged at WARN level.
	SlowQueryThreshold time.Duration

	// VerySlowQueryThreshold is the duration above which queries are logged as very slow.
	// Queries taking longer than this will be logged at ERROR level.
	VerySlowQueryThreshold time.Duration

	// LogAllQueries enables logging of all queries (at DEBUG level).
	// When false, only slow queries are logged.
	LogAllQueries bool

	// SampleRate is the fraction of queries to log when LogAllQueries is true.
	// 1.0 = all queries, 0.1 = 10% of queries. Only applies to non-slow queries.
	SampleRate float64
}

// DefaultQueryLoggerConfig returns sensible defaults for query logging.
func DefaultQueryLoggerConfig() *QueryLoggerConfig {
	return &QueryLoggerConfig{
		SlowQueryThreshold:     100 * time.Millisecond,
		VerySlowQueryThreshold: 500 * time.Millisecond,
		LogAllQueries:          false,
		SampleRate:             0.1, // Log 10% of queries when LogAllQueries is true
	}
}

// QueryStats tracks query statistics.
type QueryStats struct {
	TotalQueries     int64
	SlowQueries      int64
	VerySlowQueries  int64
	FailedQueries    int64
	TotalDuration    time.Duration
	mu               sync.RWMutex
	slowestQuery     string
	slowestDuration  time.Duration
}

// GetStats returns a copy of the current stats.
func (qs *QueryStats) GetStats() (total, slow, verySlow, failed int64, avgDuration time.Duration) {
	total = atomic.LoadInt64(&qs.TotalQueries)
	slow = atomic.LoadInt64(&qs.SlowQueries)
	verySlow = atomic.LoadInt64(&qs.VerySlowQueries)
	failed = atomic.LoadInt64(&qs.FailedQueries)

	if total > 0 {
		qs.mu.RLock()
		avgDuration = qs.TotalDuration / time.Duration(total)
		qs.mu.RUnlock()
	}
	return
}

// GetSlowestQuery returns the slowest query seen and its duration.
func (qs *QueryStats) GetSlowestQuery() (query string, duration time.Duration) {
	qs.mu.RLock()
	defer qs.mu.RUnlock()
	return qs.slowestQuery, qs.slowestDuration
}

// QueryLogger implements pgx query tracing for monitoring and logging.
type QueryLogger struct {
	config *QueryLoggerConfig
	logger *zap.Logger
	stats  *QueryStats
	sample uint64 // Counter for sampling
}

// NewQueryLogger creates a new query logger.
func NewQueryLogger(cfg *QueryLoggerConfig, logger *zap.Logger) *QueryLogger {
	if cfg == nil {
		cfg = DefaultQueryLoggerConfig()
	}
	return &QueryLogger{
		config: cfg,
		logger: logger.Named("query"),
		stats:  &QueryStats{},
	}
}

// Stats returns the query statistics.
func (ql *QueryLogger) Stats() *QueryStats {
	return ql.stats
}

// queryTraceData stores timing data across trace calls.
type queryTraceData struct {
	startTime time.Time
	sql       string
	args      []any
}

// ctxKey is the context key type for storing trace data.
type ctxKey struct{}

// TraceQueryStart is called at the beginning of query execution.
// It implements pgx.QueryTracer interface.
func (ql *QueryLogger) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, ctxKey{}, &queryTraceData{
		startTime: time.Now(),
		sql:       data.SQL,
		args:      data.Args,
	})
}

// TraceQueryEnd is called at the end of query execution.
// It implements pgx.QueryTracer interface.
func (ql *QueryLogger) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	traceData, ok := ctx.Value(ctxKey{}).(*queryTraceData)
	if !ok {
		return
	}

	duration := time.Since(traceData.startTime)
	atomic.AddInt64(&ql.stats.TotalQueries, 1)

	// Update total duration
	ql.stats.mu.Lock()
	ql.stats.TotalDuration += duration
	if duration > ql.stats.slowestDuration {
		ql.stats.slowestDuration = duration
		ql.stats.slowestQuery = truncateSQL(traceData.sql, 200)
	}
	ql.stats.mu.Unlock()

	// Track failures
	if data.Err != nil {
		atomic.AddInt64(&ql.stats.FailedQueries, 1)
		ql.logger.Error("query failed",
			zap.String("sql", truncateSQL(traceData.sql, 500)),
			zap.Duration("duration", duration),
			zap.Error(data.Err),
		)
		return
	}

	// Determine logging level based on duration
	isVerySlow := duration >= ql.config.VerySlowQueryThreshold
	isSlow := duration >= ql.config.SlowQueryThreshold

	if isVerySlow {
		atomic.AddInt64(&ql.stats.VerySlowQueries, 1)
		atomic.AddInt64(&ql.stats.SlowQueries, 1)
		ql.logger.Error("very slow query detected",
			zap.String("sql", truncateSQL(traceData.sql, 500)),
			zap.Duration("duration", duration),
			zap.Duration("threshold", ql.config.VerySlowQueryThreshold),
			zap.String("command_tag", data.CommandTag.String()),
		)
	} else if isSlow {
		atomic.AddInt64(&ql.stats.SlowQueries, 1)
		ql.logger.Warn("slow query detected",
			zap.String("sql", truncateSQL(traceData.sql, 500)),
			zap.Duration("duration", duration),
			zap.Duration("threshold", ql.config.SlowQueryThreshold),
			zap.String("command_tag", data.CommandTag.String()),
		)
	} else if ql.config.LogAllQueries {
		// Sample queries if not slow
		if ql.shouldSample() {
			ql.logger.Debug("query executed",
				zap.String("sql", truncateSQL(traceData.sql, 200)),
				zap.Duration("duration", duration),
				zap.String("command_tag", data.CommandTag.String()),
			)
		}
	}
}

// shouldSample determines if a query should be sampled for logging.
func (ql *QueryLogger) shouldSample() bool {
	if ql.config.SampleRate >= 1.0 {
		return true
	}
	if ql.config.SampleRate <= 0 {
		return false
	}

	// Simple sampling: increment counter and check modulo
	count := atomic.AddUint64(&ql.sample, 1)
	threshold := uint64(1.0 / ql.config.SampleRate)
	return count%threshold == 0
}

// truncateSQL truncates SQL to a maximum length for logging.
func truncateSQL(sql string, maxLen int) string {
	if len(sql) <= maxLen {
		return sql
	}
	return sql[:maxLen-3] + "..."
}

// LogStats logs current query statistics.
func (ql *QueryLogger) LogStats() {
	total, slow, verySlow, failed, avgDuration := ql.stats.GetStats()
	slowest, slowestDuration := ql.stats.GetSlowestQuery()

	ql.logger.Info("query statistics",
		zap.Int64("total_queries", total),
		zap.Int64("slow_queries", slow),
		zap.Int64("very_slow_queries", verySlow),
		zap.Int64("failed_queries", failed),
		zap.Duration("avg_duration", avgDuration),
		zap.String("slowest_query", slowest),
		zap.Duration("slowest_duration", slowestDuration),
	)
}

// ResetStats resets the query statistics.
func (ql *QueryLogger) ResetStats() {
	atomic.StoreInt64(&ql.stats.TotalQueries, 0)
	atomic.StoreInt64(&ql.stats.SlowQueries, 0)
	atomic.StoreInt64(&ql.stats.VerySlowQueries, 0)
	atomic.StoreInt64(&ql.stats.FailedQueries, 0)

	ql.stats.mu.Lock()
	ql.stats.TotalDuration = 0
	ql.stats.slowestQuery = ""
	ql.stats.slowestDuration = 0
	ql.stats.mu.Unlock()
}
