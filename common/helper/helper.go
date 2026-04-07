package helper

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// GetTimestamp returns the current Unix timestamp in seconds.
func GetTimestamp() int64 {
	return time.Now().Unix()
}

// GetTimestampMs returns the current Unix timestamp in milliseconds.
func GetTimestampMs() int64 {
	return time.Now().UnixMilli()
}

// GetRequestID retrieves the request ID from the context, or returns empty string.
func GetRequestID(ctx context.Context) string {
	if v := ctx.Value(requestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// WithRequestID returns a new context with the request ID set.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// NewRequestID generates a new UUID-based request ID.
func NewRequestID() string {
	return uuid.New().String()
}

// CalcElapsedTime returns milliseconds elapsed since t.
func CalcElapsedTime(t time.Time) int64 {
	return time.Since(t).Milliseconds()
}
