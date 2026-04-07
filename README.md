# AI Gateway

An OpenAI-compatible LLM API gateway inspired by [one-api](https://github.com/songquanpeng/one-api).

**Business data** is stored in **PostgreSQL** (users, tokens, channels).  
**Model API request/response logs** are stored in **ClickHouse** for high-throughput analytics.

---

## Features

- OpenAI-compatible `/v1/chat/completions` and `/v1/completions` relay
- Bearer token authentication
- PostgreSQL for transactional business data (GORM, auto-migrate)
- ClickHouse for LLM call logs with:
  - Async / batched writes (never blocks request handling)
  - Automatic schema creation (table + materialized daily-aggregate view)
  - Configurable TTL / retention
  - Per-field text/JSON truncation safeguards
- Single-node `docker-compose` for local development

---

## Quick Start (Docker Compose)

```bash
# Clone & start
git clone https://github.com/taills/ai-gateway.git
cd ai-gateway
docker compose up -d
```

The gateway will be available at `http://localhost:3000`.

Default root credentials (auto-created on first start): `root / 123456`

---

## Configuration

Copy `.env.example` to `.env` and adjust values.

| Variable | Default | Description |
|---|---|---|
| `PORT` | `3000` | HTTP listen port |
| `SQL_DSN` | *(SQLite)* | PostgreSQL DSN (`postgres://...`); falls back to SQLite if unset |
| `CLICKHOUSE_ENABLED` | `false` | Enable ClickHouse logging |
| `CLICKHOUSE_DSN` | `` | ClickHouse DSN (`clickhouse://user:pass@host:9000/db`) |
| `CLICKHOUSE_DATABASE` | `ai_gateway` | ClickHouse database name |
| `CLICKHOUSE_RETENTION_DAYS` | `90` | TTL for log rows (days) |
| `CLICKHOUSE_MAX_TEXT_BYTES` | `16384` | Max bytes for request/response text fields |
| `CLICKHOUSE_MAX_JSON_BYTES` | `65536` | Max bytes for request/response JSON fields |
| `CLICKHOUSE_BATCH_SIZE` | `500` | Number of rows per ClickHouse INSERT batch |
| `CLICKHOUSE_FLUSH_INTERVAL` | `5s` | How often to flush partial batches |
| `DEBUG` | `false` | Verbose application logging |
| `DEBUG_SQL` | `false` | Log SQL queries |
| `GIN_MODE` | `release` | Gin framework mode (`release`/`debug`) |

---

## Architecture

```
[Client] ──► POST /v1/chat/completions
                │
                ▼
        [Auth Middleware]  ◄── PostgreSQL: token lookup
                │
                ▼
        [RelayTextHelper]
          ├── Read channel from PostgreSQL
          ├── Forward to upstream LLM provider
          ├── Write response to client
          ├── Enqueue ClickHouse log entry (async, non-blocking)
          └── Update PostgreSQL quota / request count (async)
```

### ClickHouse Schema

Two tables are created automatically on startup:

#### `llm_api_call_log`

One row per API call. Partitioned by month, TTL-deleted after the configured retention period.

Key columns: `request_id`, `user_id`, `username`, `token_id`, `token_name`, `channel_id`, `channel_name`, `provider`, `model_name`, `status`, `latency_ms`, `prompt_tokens`, `completion_tokens`, `total_tokens`, `quota`, `request_text`, `response_text`, `request_json`, `response_json`.

#### `llm_api_call_daily_stats`

Pre-aggregated daily statistics (populated via materialized view `mv_llm_api_call_daily_stats`).

Key columns: `stat_date`, `user_id`, `channel_id`, `model_name`, `request_count`, `success_count`, `error_count`, `total_tokens`, `total_quota`, `avg_latency_ms`, `p95_latency_ms`.

---

## Development

```bash
# Run tests
go test ./...

# Build binary
go build -o ai-gateway .

# Run locally (SQLite + no ClickHouse)
./ai-gateway
```

---

## Project Structure

```
.
├── main.go                  Entry point
├── common/
│   ├── config/              All configuration (env-driven)
│   ├── helper/              Request ID, timestamp utilities
│   └── logger/              Structured logging
├── model/                   PostgreSQL models (GORM)
│   ├── main.go              DB init / migration
│   ├── user.go              User model
│   ├── token.go             API token model
│   ├── channel.go           Upstream channel model
│   └── log.go               Consume log model
├── clickhouse/              ClickHouse logging package
│   ├── clickhouse.go        Init, schema creation
│   ├── model.go             APICallLog struct
│   ├── writer.go            Async batch writer
│   └── truncate.go          Text/JSON truncation helpers
├── relay/
│   ├── meta/                Per-request metadata
│   ├── model/               OpenAI-compatible request/response types
│   └── controller/          Relay handler with ClickHouse integration
├── middleware/              Gin middleware (auth, request ID, logger)
├── router/                  HTTP route definitions
├── docker-compose.yml       Single-node local setup
├── Dockerfile
└── .env.example
```
