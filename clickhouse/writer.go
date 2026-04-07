package clickhouse

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"time"

	"github.com/taills/ai-gateway/common/logger"
)

// asyncWorker batches APICallLog entries and flushes them to ClickHouse.
// Writes are completely decoupled from the request path so that ClickHouse
// downtime or slowness never affects API latency.
type asyncWorker struct {
	db            *sql.DB
	batchSize     int
	flushInterval time.Duration

	queue  chan *APICallLog
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func newAsyncWorker(db *sql.DB, batchSize int, flushInterval time.Duration) *asyncWorker {
	if batchSize <= 0 {
		batchSize = 500
	}
	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}
	return &asyncWorker{
		db:            db,
		batchSize:     batchSize,
		flushInterval: flushInterval,
		queue:          make(chan *APICallLog, batchSize*10),
		stopCh:        make(chan struct{}),
	}
}

func (w *asyncWorker) start() {
	w.wg.Add(1)
	go w.loop()
}

func (w *asyncWorker) enqueue(entry *APICallLog) {
	select {
	case w.queue <- entry:
	default:
		// Drop rather than block the caller if the queue is full.
		logger.SysLog("clickhouse: write queue full, dropping log entry")
	}
}

func (w *asyncWorker) stop() {
	close(w.stopCh)
	w.wg.Wait()
}

func (w *asyncWorker) loop() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	var buf []*APICallLog

	flush := func() {
		if len(buf) == 0 {
			return
		}
		if err := w.writeBatch(buf); err != nil {
			logger.SysLogf("clickhouse: batch write failed: %v", err)
		}
		buf = buf[:0]
	}

	for {
		select {
		case entry := <-w.queue:
			buf = append(buf, entry)
			if len(buf) >= w.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-w.stopCh:
			// Drain remaining items
		drain:
			for {
				select {
				case entry := <-w.queue:
					buf = append(buf, entry)
				default:
					break drain
				}
			}
			flush()
			return
		}
	}
}

// writeBatch inserts a slice of log entries into ClickHouse using a single
// batch INSERT statement.
func (w *asyncWorker) writeBatch(entries []*APICallLog) error {
	if len(entries) == 0 {
		return nil
	}

	const insertSQL = `INSERT INTO llm_api_call_log
(created_at, request_id,
 user_id, username, token_id, token_name, channel_id, channel_name, provider, model_id, model_name,
 status, http_status_code, error_code, error_message,
 is_stream, latency_ms, ttft_ms,
 prompt_tokens, completion_tokens, total_tokens, quota,
 request_text, response_text, request_json, response_json,
 request_text_truncated, response_text_truncated, request_json_truncated, response_json_truncated)
VALUES `

	placeholders := make([]string, len(entries))
	args := make([]any, 0, len(entries)*30)

	for i, e := range entries {
		placeholders[i] = "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"
		isStream := uint8(0)
		if e.IsStream {
			isStream = 1
		}
		reqTrunc := uint8(0)
		if e.RequestTextTruncated {
			reqTrunc = 1
		}
		respTrunc := uint8(0)
		if e.ResponseTextTruncated {
			respTrunc = 1
		}
		reqJSONTrunc := uint8(0)
		if e.RequestJSONTruncated {
			reqJSONTrunc = 1
		}
		respJSONTrunc := uint8(0)
		if e.ResponseJSONTruncated {
			respJSONTrunc = 1
		}
		args = append(args,
			e.CreatedAt, e.RequestID,
			e.UserID, e.Username, e.TokenID, e.TokenName, e.ChannelID, e.ChannelName, e.Provider, e.ModelID, e.ModelName,
			e.Status, e.HTTPStatusCode, e.ErrorCode, e.ErrorMessage,
			isStream, e.LatencyMs, e.TTFTMs,
			e.PromptTokens, e.CompletionTokens, e.TotalTokens, e.Quota,
			e.RequestText, e.ResponseText, e.RequestJSON, e.ResponseJSON,
			reqTrunc, respTrunc, reqJSONTrunc, respJSONTrunc,
		)
	}

	query := insertSQL + strings.Join(placeholders, ",")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := w.db.ExecContext(ctx, query, args...)
	return err
}
