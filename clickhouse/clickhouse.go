// Package clickhouse provides asynchronous ClickHouse-backed logging for LLM
// API request/response events. It is designed to be non-blocking: log entries
// are queued in memory and flushed in batches so that the hot request path is
// never delayed by ClickHouse availability.
package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/taills/ai-gateway/common/config"
	"github.com/taills/ai-gateway/common/logger"

	_ "github.com/ClickHouse/clickhouse-go/v2" // register driver
)

var (
	db     *sql.DB
	once   sync.Once
	worker *asyncWorker
)

// Init opens the ClickHouse connection and starts the background flush worker.
// It is safe to call multiple times; subsequent calls are no-ops.
func Init() {
	if !config.ClickHouseEnabled {
		return
	}
	once.Do(func() {
		var err error
		db, err = sql.Open("clickhouse", config.ClickHouseDSN)
		if err != nil {
			logger.SysLogf("clickhouse: failed to open connection: %v", err)
			return
		}
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(time.Minute * 5)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			logger.SysLogf("clickhouse: ping failed: %v", err)
			db = nil
			return
		}

		if err = ensureSchema(db, config.ClickHouseDatabase, config.ClickHouseRetentionDays); err != nil {
			logger.SysLogf("clickhouse: schema setup failed: %v", err)
			db = nil
			return
		}

		worker = newAsyncWorker(db, config.ClickHouseBatchSize, config.ClickHouseFlushInterval)
		worker.start()
		logger.SysLog("clickhouse: connection established and worker started")
	})
}

// Enabled reports whether ClickHouse logging is active.
func Enabled() bool {
	return db != nil && worker != nil
}

// Enqueue adds an APICallLog to the async write queue. It is non-blocking.
// If ClickHouse is not enabled the call is a no-op.
func Enqueue(entry *APICallLog) {
	if !Enabled() {
		return
	}
	worker.enqueue(entry)
}

// Close flushes pending entries and closes the ClickHouse connection.
func Close() {
	if worker != nil {
		worker.stop()
	}
	if db != nil {
		_ = db.Close()
	}
}

// ensureSchema creates the database, main log table, daily aggregate table and
// the materialized view that keeps the aggregate up to date.
func ensureSchema(db *sql.DB, database string, retentionDays int) error {
	ttl := fmt.Sprintf("toDate(created_at) + INTERVAL %d DAY", retentionDays)

	stmts := []string{
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", database),

		// ── Main request/response log table ─────────────────────────────────
		fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s.llm_api_call_log
(
    event_date        Date     DEFAULT toDate(created_at),
    created_at        DateTime64(3),
    request_id        String,

    user_id           UInt64   DEFAULT 0,
    username          String   DEFAULT '',
    token_id          UInt64   DEFAULT 0,
    token_name        String   DEFAULT '',
    channel_id        UInt64   DEFAULT 0,
    channel_name      String   DEFAULT '',
    provider          LowCardinality(String) DEFAULT '',
    model_id          UInt64   DEFAULT 0,
    model_name        LowCardinality(String) DEFAULT '',

    status            LowCardinality(String) DEFAULT '',  -- success / error / timeout
    http_status_code  UInt16   DEFAULT 0,
    error_code        String   DEFAULT '',
    error_message     String   DEFAULT '',

    is_stream         UInt8    DEFAULT 0,
    latency_ms        UInt32   DEFAULT 0,
    ttft_ms           UInt32   DEFAULT 0,

    prompt_tokens     UInt32   DEFAULT 0,
    completion_tokens UInt32   DEFAULT 0,
    total_tokens      UInt32   DEFAULT 0,
    quota             Int64    DEFAULT 0,

    request_text      String   DEFAULT '',
    response_text     String   DEFAULT '',
    request_json      String   DEFAULT '',
    response_json     String   DEFAULT '',

    request_text_truncated  UInt8 DEFAULT 0,
    response_text_truncated UInt8 DEFAULT 0,
    request_json_truncated  UInt8 DEFAULT 0,
    response_json_truncated UInt8 DEFAULT 0
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (event_date, user_id, model_name, created_at, request_id)
TTL %s DELETE
SETTINGS index_granularity = 8192
`, database, ttl),

		// ── Daily aggregate table ────────────────────────────────────────────
		fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s.llm_api_call_daily_stats
(
    stat_date               Date,
    user_id                 UInt64,
    channel_id              UInt64,
    model_name              LowCardinality(String),

    request_count           UInt64,
    success_count           UInt64,
    error_count             UInt64,

    total_prompt_tokens     UInt64,
    total_completion_tokens UInt64,
    total_tokens            UInt64,
    total_quota             Int64,

    avg_latency_ms          Float64,
    p95_latency_ms          Float64
)
ENGINE = ReplacingMergeTree
ORDER BY (stat_date, user_id, channel_id, model_name)
`, database),

		// ── Materialized view → daily stats ──────────────────────────────────
		fmt.Sprintf(`
CREATE MATERIALIZED VIEW IF NOT EXISTS %s.mv_llm_api_call_daily_stats
TO %s.llm_api_call_daily_stats
AS
SELECT
    event_date                        AS stat_date,
    user_id,
    channel_id,
    model_name,
    count()                           AS request_count,
    countIf(status = 'success')       AS success_count,
    countIf(status != 'success')      AS error_count,
    sum(prompt_tokens)                AS total_prompt_tokens,
    sum(completion_tokens)            AS total_completion_tokens,
    sum(total_tokens)                 AS total_tokens,
    sum(quota)                        AS total_quota,
    avg(latency_ms)                   AS avg_latency_ms,
    quantile(0.95)(latency_ms)        AS p95_latency_ms
FROM %s.llm_api_call_log
GROUP BY stat_date, user_id, channel_id, model_name
`, database, database, database),
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("exec schema stmt: %w\nSQL: %s", err, stmt)
		}
	}
	return nil
}
