package config

import (
	"os"
	"strconv"
	"time"
)

var (
	// Server
	Port           = 3000
	SessionSecret  = "ai-gateway-secret"
	DebugEnabled   = os.Getenv("DEBUG") == "true"
	DebugSQLEnabled = os.Getenv("DEBUG_SQL") == "true"

	// Features
	LogConsumeEnabled = true
	IsMasterNode      = os.Getenv("NODE_TYPE") != "slave"
	BatchUpdateEnabled  = false
	BatchUpdateInterval = 5

	// ClickHouse
	ClickHouseEnabled       = false
	ClickHouseDSN           = ""
	ClickHouseDatabase      = "ai_gateway"
	ClickHouseRetentionDays = 90
	// Max text length stored in ClickHouse (bytes); truncated if exceeded
	ClickHouseMaxTextBytes = 16 * 1024 // 16 KB
	// Max JSON payload length stored in ClickHouse (bytes)
	ClickHouseMaxJSONBytes = 64 * 1024 // 64 KB
	// Async batch settings
	ClickHouseBatchSize    = 500
	ClickHouseFlushInterval = 5 * time.Second
)

func init() {
	if v := os.Getenv("PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			Port = n
		}
	}
	if v := os.Getenv("SESSION_SECRET"); v != "" {
		SessionSecret = v
	}
	if v := os.Getenv("CLICKHOUSE_DSN"); v != "" {
		ClickHouseEnabled = true
		ClickHouseDSN = v
	}
	if v := os.Getenv("CLICKHOUSE_ENABLED"); v == "true" {
		ClickHouseEnabled = true
	}
	if v := os.Getenv("CLICKHOUSE_DATABASE"); v != "" {
		ClickHouseDatabase = v
	}
	if v := os.Getenv("CLICKHOUSE_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			ClickHouseRetentionDays = n
		}
	}
	if v := os.Getenv("CLICKHOUSE_MAX_TEXT_BYTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			ClickHouseMaxTextBytes = n
		}
	}
	if v := os.Getenv("CLICKHOUSE_MAX_JSON_BYTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			ClickHouseMaxJSONBytes = n
		}
	}
	if v := os.Getenv("CLICKHOUSE_BATCH_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			ClickHouseBatchSize = n
		}
	}
	if v := os.Getenv("CLICKHOUSE_FLUSH_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			ClickHouseFlushInterval = d
		}
	}
}
