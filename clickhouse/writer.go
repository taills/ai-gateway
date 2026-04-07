package clickhouse

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/taills/ai-gateway/common/logger"
)

// asyncWorker batches APICallLog entries and flushes them to ClickHouse.
// Writes are completely decoupled from the request path so that ClickHouse
// downtime or slowness never affects API latency.
type asyncWorker struct {
	client        *httpClient
	batchSize     int
	flushInterval time.Duration

	queue  chan *APICallLog
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func newAsyncWorker(client *httpClient, batchSize int, flushInterval time.Duration) *asyncWorker {
	if batchSize <= 0 {
		batchSize = 500
	}
	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}
	return &asyncWorker{
		client:        client,
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

// writeBatch converts the slice of log entries into JSONEachRow rows and sends
// them to ClickHouse via the HTTP interface.
func (w *asyncWorker) writeBatch(entries []*APICallLog) error {
	if len(entries) == 0 {
		return nil
	}

	rows := make([]map[string]any, len(entries))
	for i, e := range entries {
		isStream := uint8(0)
		if e.IsStream {
			isStream = 1
		}
		rows[i] = map[string]any{
			"created_at":              e.CreatedAt.Format("2006-01-02 15:04:05.000"),
			"request_id":              e.RequestID,
			"user_id":                 e.UserID,
			"username":                e.Username,
			"token_id":                e.TokenID,
			"token_name":              e.TokenName,
			"channel_id":              e.ChannelID,
			"channel_name":            e.ChannelName,
			"provider":                e.Provider,
			"model_id":                e.ModelID,
			"model_name":              e.ModelName,
			"status":                  e.Status,
			"http_status_code":        e.HTTPStatusCode,
			"error_code":              e.ErrorCode,
			"error_message":           e.ErrorMessage,
			"is_stream":               isStream,
			"latency_ms":              e.LatencyMs,
			"ttft_ms":                 e.TTFTMs,
			"prompt_tokens":           e.PromptTokens,
			"completion_tokens":       e.CompletionTokens,
			"total_tokens":            e.TotalTokens,
			"quota":                   e.Quota,
			"request_text":            e.RequestText,
			"response_text":           e.ResponseText,
			"request_json":            e.RequestJSON,
			"response_json":           e.ResponseJSON,
			"request_text_truncated":  boolToUint8(e.RequestTextTruncated),
			"response_text_truncated": boolToUint8(e.ResponseTextTruncated),
			"request_json_truncated":  boolToUint8(e.RequestJSONTruncated),
			"response_json_truncated": boolToUint8(e.ResponseJSONTruncated),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := w.client.insertJSONEachRow(ctx, "llm_api_call_log", rows); err != nil {
		return fmt.Errorf("insert batch: %w", err)
	}
	return nil
}

func boolToUint8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}
