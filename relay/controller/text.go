package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	chttp "github.com/taills/ai-gateway/clickhouse"
	"github.com/taills/ai-gateway/common/config"
	"github.com/taills/ai-gateway/common/helper"
	"github.com/taills/ai-gateway/common/logger"
	dbmodel "github.com/taills/ai-gateway/model"
	"github.com/taills/ai-gateway/relay/meta"
	relaymodel "github.com/taills/ai-gateway/relay/model"
)

// RelayTextHelper handles a chat/completions relay request.
// It forwards the request to the upstream provider, records PostgreSQL consume
// logs, and enqueues a ClickHouse entry asynchronously.
func RelayTextHelper(c *gin.Context) *relaymodel.ErrorWithStatusCode {
	ctx := c.Request.Context()
	m := meta.GetByContext(c)

	// ── Parse request ────────────────────────────────────────────────────────
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return wrapError(err, "read_request_body_failed", http.StatusBadRequest)
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	var textRequest relaymodel.GeneralOpenAIRequest
	if err := json.Unmarshal(body, &textRequest); err != nil {
		return wrapError(err, "invalid_request_body", http.StatusBadRequest)
	}
	m.ModelName = textRequest.Model
	m.IsStream = textRequest.Stream

	// ── Fetch channel ────────────────────────────────────────────────────────
	channel, err := dbmodel.GetChannelById(m.ChannelID)
	if err != nil {
		return wrapError(fmt.Errorf("channel not found: %d", m.ChannelID), "channel_not_found", http.StatusBadRequest)
	}
	m.ChannelName = channel.Name
	m.ChannelType = channel.Type
	m.APIKey = channel.Key
	if channel.BaseURL != "" {
		m.BaseURL = channel.BaseURL
	}

	// ── Forward to upstream ──────────────────────────────────────────────────
	upstreamURL, err := buildUpstreamURL(m.BaseURL, c.FullPath())
	if err != nil {
		return wrapError(err, "invalid_channel_base_url", http.StatusBadGateway)
	}
	upstreamResp, upstreamRespBody, upstreamErr := forwardRequest(c, upstreamURL, m.APIKey, body)

	startTime := m.StartTime
	latencyMs := uint32(time.Since(startTime).Milliseconds())

	// ── Build ClickHouse log entry ───────────────────────────────────────────
	entry := buildClickHouseEntry(m, textRequest, upstreamResp, upstreamRespBody, upstreamErr, latencyMs, body)
	chttp.Enqueue(entry)

	// ── Handle upstream error ────────────────────────────────────────────────
	if upstreamErr != nil {
		logger.Errorf(ctx, "upstream request failed: %s", upstreamErr.Error())
		return wrapError(upstreamErr, "upstream_request_failed", http.StatusBadGateway)
	}
	if upstreamResp.StatusCode != http.StatusOK {
		return &relaymodel.ErrorWithStatusCode{
			StatusCode: upstreamResp.StatusCode,
			Error: relaymodel.OpenAIError{
				Message: fmt.Sprintf("upstream returned %d", upstreamResp.StatusCode),
				Type:    "upstream_error",
			},
		}
	}

	// ── Parse usage from response ────────────────────────────────────────────
	var usage relaymodel.Usage
	var respObj struct {
		Usage *relaymodel.Usage `json:"usage"`
	}
	if err := json.Unmarshal(upstreamRespBody, &respObj); err == nil && respObj.Usage != nil {
		usage = *respObj.Usage
	}

	// ── Write response to client ─────────────────────────────────────────────
	for k, vals := range upstreamResp.Header {
		for _, v := range vals {
			c.Header(k, v)
		}
	}
	c.Status(upstreamResp.StatusCode)
	c.Writer.Write(upstreamRespBody) //nolint:errcheck

	// ── PostgreSQL quota accounting (async so it doesn't block response) ─────
	go postConsumeQuota(ctx, m, &textRequest, &usage, startTime)

	return nil
}

// forwardRequest sends the request body to the upstream URL using the provided
// API key.  It returns the upstream response, the full response body bytes, and
// any transport-level error.
func forwardRequest(c *gin.Context, url, apiKey string, body []byte) (*http.Response, []byte, error) {
	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	req.Header = c.Request.Header.Clone()
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	return resp, respBody, err
}

// buildUpstreamURL constructs the full upstream URL from the channel's base URL
// and the request path. The baseURL is validated to be a safe http/https URL
// stored in the database by an administrator; invalid schemes are rejected.
func buildUpstreamURL(baseURL, path string) (string, error) {
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid channel base_url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("channel base_url must use http or https scheme, got %q", parsed.Scheme)
	}
	// Ensure path starts with /
	if path != "" && path[0] != '/' {
		path = "/" + path
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + path
	return parsed.String(), nil
}

// buildClickHouseEntry assembles a ClickHouse log entry from all available
// context, snapshotting PostgreSQL identifiers and truncating large payloads.
func buildClickHouseEntry(
	m *meta.Meta,
	req relaymodel.GeneralOpenAIRequest,
	resp *http.Response,
	respBody []byte,
	upstreamErr error,
	latencyMs uint32,
	reqBody []byte,
) *chttp.APICallLog {
	entry := &chttp.APICallLog{
		CreatedAt: m.StartTime,
		RequestID: m.RequestID,

		UserID:      uint64(m.UserID),
		Username:    m.Username,
		TokenID:     uint64(m.TokenID),
		TokenName:   m.TokenName,
		ChannelID:   uint64(m.ChannelID),
		ChannelName: m.ChannelName,
		Provider:    channelTypeToProvider(m.ChannelType),
		ModelName:   m.ModelName,

		IsStream:  m.IsStream,
		LatencyMs: latencyMs,
	}

	if upstreamErr != nil {
		entry.Status = "error"
		entry.ErrorMessage = upstreamErr.Error()
	} else if resp != nil {
		entry.HTTPStatusCode = uint16(resp.StatusCode)
		if resp.StatusCode == http.StatusOK {
			entry.Status = "success"
		} else {
			entry.Status = "error"
		}
	}

	// Request JSON
	reqJSON := string(reqBody)
	if t, truncated := chttp.TruncateText(reqJSON, config.ClickHouseMaxJSONBytes); truncated {
		entry.RequestJSON = t
		entry.RequestJSONTruncated = true
	} else {
		entry.RequestJSON = reqJSON
	}

	// Request text (first user message content)
	if len(req.Messages) > 0 {
		for _, msg := range req.Messages {
			if s, ok := msg.Content.(string); ok {
				if t, truncated := chttp.TruncateText(s, config.ClickHouseMaxTextBytes); truncated {
					entry.RequestText = t
					entry.RequestTextTruncated = true
				} else {
					entry.RequestText = s
				}
				break
			}
		}
	} else if req.Prompt != "" {
		if t, truncated := chttp.TruncateText(req.Prompt, config.ClickHouseMaxTextBytes); truncated {
			entry.RequestText = t
			entry.RequestTextTruncated = true
		} else {
			entry.RequestText = req.Prompt
		}
	}

	// Response JSON
	if len(respBody) > 0 {
		respJSON := string(respBody)
		if t, truncated := chttp.TruncateText(respJSON, config.ClickHouseMaxJSONBytes); truncated {
			entry.ResponseJSON = t
			entry.ResponseJSONTruncated = true
		} else {
			entry.ResponseJSON = respJSON
		}

		// Response text (first assistant message)
		var respObj struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				Text string `json:"text"`
			} `json:"choices"`
			Usage *relaymodel.Usage `json:"usage"`
		}
		if err := json.Unmarshal(respBody, &respObj); err == nil {
			if len(respObj.Choices) > 0 {
				text := respObj.Choices[0].Message.Content
				if text == "" {
					text = respObj.Choices[0].Text
				}
				if t, truncated := chttp.TruncateText(text, config.ClickHouseMaxTextBytes); truncated {
					entry.ResponseText = t
					entry.ResponseTextTruncated = true
				} else {
					entry.ResponseText = text
				}
			}
			if respObj.Usage != nil {
				entry.PromptTokens = uint32(respObj.Usage.PromptTokens)
				entry.CompletionTokens = uint32(respObj.Usage.CompletionTokens)
				entry.TotalTokens = uint32(respObj.Usage.TotalTokens)
			}
		}
	}

	return entry
}

func channelTypeToProvider(channelType int) string {
	switch channelType {
	case dbmodel.ChannelTypeOpenAI:
		return "openai"
	case dbmodel.ChannelTypeAzure:
		return "azure"
	case dbmodel.ChannelTypeAnthropic:
		return "anthropic"
	default:
		return "unknown"
	}
}

func wrapError(err error, code string, status int) *relaymodel.ErrorWithStatusCode {
	return &relaymodel.ErrorWithStatusCode{
		StatusCode: status,
		Error: relaymodel.OpenAIError{
			Message: err.Error(),
			Type:    "api_error",
			Code:    code,
		},
	}
}

func postConsumeQuota(ctx context.Context, m *meta.Meta, req *relaymodel.GeneralOpenAIRequest, usage *relaymodel.Usage, startTime time.Time) {
	if usage == nil {
		return
	}
	totalTokens := usage.PromptTokens + usage.CompletionTokens
	if totalTokens == 0 {
		return
	}

	// Rough quota = total tokens (caller can scale by model ratio)
	quota := int64(totalTokens)
	dbmodel.UpdateUserUsedQuotaAndRequestCount(m.UserID, quota)
	dbmodel.UpdateChannelUsedQuota(m.ChannelID, quota)
	dbmodel.RecordConsumeLog(ctx, &dbmodel.Log{
		UserId:           m.UserID,
		ChannelId:        m.ChannelID,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		ModelName:        m.ModelName,
		TokenName:        m.TokenName,
		Quota:            quota,
		ElapsedTime:      helper.CalcElapsedTime(startTime),
		IsStream:         m.IsStream,
	})
}
