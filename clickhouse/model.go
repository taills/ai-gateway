package clickhouse

import "time"

// APICallLog is the canonical log entry for one LLM API call. Fields mirror
// the llm_api_call_log ClickHouse table. Identifiers from PostgreSQL business
// tables are snapshotted here so that logs remain self-contained even if
// upstream data changes.
type APICallLog struct {
	CreatedAt time.Time `ch:"created_at"`
	RequestID string    `ch:"request_id"`

	// Snapshot of PostgreSQL business identifiers
	UserID      uint64 `ch:"user_id"`
	Username    string `ch:"username"`
	TokenID     uint64 `ch:"token_id"`
	TokenName   string `ch:"token_name"`
	ChannelID   uint64 `ch:"channel_id"`
	ChannelName string `ch:"channel_name"`
	Provider    string `ch:"provider"`
	ModelID     uint64 `ch:"model_id"`
	ModelName   string `ch:"model_name"`

	// Outcome
	Status         string `ch:"status"`          // "success" | "error" | "timeout"
	HTTPStatusCode uint16 `ch:"http_status_code"`
	ErrorCode      string `ch:"error_code"`
	ErrorMessage   string `ch:"error_message"`

	// Performance
	IsStream  bool   `ch:"is_stream"`
	LatencyMs uint32 `ch:"latency_ms"`
	TTFTMs    uint32 `ch:"ttft_ms"` // time-to-first-token

	// Token usage & billing
	PromptTokens     uint32 `ch:"prompt_tokens"`
	CompletionTokens uint32 `ch:"completion_tokens"`
	TotalTokens      uint32 `ch:"total_tokens"`
	Quota            int64  `ch:"quota"`

	// Payload (truncated per config)
	RequestText  string `ch:"request_text"`
	ResponseText string `ch:"response_text"`
	RequestJSON  string `ch:"request_json"`
	ResponseJSON string `ch:"response_json"`

	// Truncation flags
	RequestTextTruncated  bool `ch:"request_text_truncated"`
	ResponseTextTruncated bool `ch:"response_text_truncated"`
	RequestJSONTruncated  bool `ch:"request_json_truncated"`
	ResponseJSONTruncated bool `ch:"response_json_truncated"`
}
