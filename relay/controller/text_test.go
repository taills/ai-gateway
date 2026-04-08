package controller

import (
	"errors"
	"net/http"
	"testing"
)

// ── buildUpstreamURL ─────────────────────────────────────────────────────────

func TestBuildUpstreamURL_DefaultBase(t *testing.T) {
	got, err := buildUpstreamURL("", "/v1/chat/completions")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://api.openai.com/v1/chat/completions"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestBuildUpstreamURL_CustomBase(t *testing.T) {
	got, err := buildUpstreamURL("http://127.0.0.1:8080", "/v1/chat/completions")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "http://127.0.0.1:8080/v1/chat/completions"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestBuildUpstreamURL_TrailingSlashOnBase(t *testing.T) {
	// Base URL with trailing slash should not produce double slash.
	got, err := buildUpstreamURL("https://proxy.example.com/", "/v1/completions")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://proxy.example.com/v1/completions"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestBuildUpstreamURL_PathWithoutLeadingSlash(t *testing.T) {
	got, err := buildUpstreamURL("https://api.openai.com", "v1/chat/completions")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://api.openai.com/v1/chat/completions"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestBuildUpstreamURL_InvalidScheme(t *testing.T) {
	_, err := buildUpstreamURL("ftp://bad.example.com", "/v1/chat/completions")
	if err == nil {
		t.Error("expected error for non-http/https scheme")
	}
}

func TestBuildUpstreamURL_InvalidURL(t *testing.T) {
	_, err := buildUpstreamURL("://bad-url", "/v1/chat/completions")
	if err == nil {
		t.Error("expected error for unparseable URL")
	}
}

// ── channelTypeToProvider ────────────────────────────────────────────────────

func TestChannelTypeToProvider(t *testing.T) {
	cases := []struct {
		input int
		want  string
	}{
		{1, "openai"},
		{3, "azure"},
		{14, "anthropic"},
		{0, "unknown"},
		{99, "unknown"},
	}
	for _, tc := range cases {
		got := channelTypeToProvider(tc.input)
		if got != tc.want {
			t.Errorf("channelTypeToProvider(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ── wrapError ────────────────────────────────────────────────────────────────

func TestWrapError(t *testing.T) {
	err := errors.New("something went wrong")
	wrapped := wrapError(err, "test_code", http.StatusBadGateway)
	if wrapped == nil {
		t.Fatal("expected non-nil result")
	}
	if wrapped.StatusCode != http.StatusBadGateway {
		t.Errorf("expected status %d got %d", http.StatusBadGateway, wrapped.StatusCode)
	}
	if wrapped.Error.Code != "test_code" {
		t.Errorf("expected code %q got %q", "test_code", wrapped.Error.Code)
	}
	if wrapped.Error.Message != "something went wrong" {
		t.Errorf("unexpected message: %q", wrapped.Error.Message)
	}
}
