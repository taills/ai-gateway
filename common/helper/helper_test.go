package helper

import (
	"context"
	"testing"
)

func TestRequestID(t *testing.T) {
	id := NewRequestID()
	if id == "" {
		t.Fatal("expected non-empty request ID")
	}

	ctx := WithRequestID(context.Background(), id)
	got := GetRequestID(ctx)
	if got != id {
		t.Errorf("expected %q got %q", id, got)
	}
}

func TestGetRequestID_Missing(t *testing.T) {
	got := GetRequestID(context.Background())
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}
