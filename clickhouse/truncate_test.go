package clickhouse

import (
	"strings"
	"testing"
)

func TestTruncateText_NoTruncation(t *testing.T) {
	s := "hello world"
	out, truncated := TruncateText(s, 100)
	if truncated {
		t.Errorf("expected no truncation for short string")
	}
	if out != s {
		t.Errorf("expected %q got %q", s, out)
	}
}

func TestTruncateText_Truncated(t *testing.T) {
	s := "hello world"
	out, truncated := TruncateText(s, 5)
	if !truncated {
		t.Errorf("expected truncation")
	}
	if out != "hello" {
		t.Errorf("expected %q got %q", "hello", out)
	}
}

func TestTruncateText_ZeroMax(t *testing.T) {
	s := "hello"
	out, truncated := TruncateText(s, 0)
	if truncated {
		t.Errorf("zero max should be a no-op")
	}
	if out != s {
		t.Errorf("expected original string")
	}
}

func TestTruncateText_MultiByte(t *testing.T) {
	// "日本語" is 9 bytes (3 bytes per rune)
	s := "日本語"
	// Truncate to 7 bytes – must not split "語" and must return "日本" (6 bytes)
	out, truncated := TruncateText(s, 7)
	if !truncated {
		t.Errorf("expected truncation")
	}
	if out != "日本" {
		t.Errorf("expected %q got %q", "日本", out)
	}
}

func TestTruncateText_LargeString(t *testing.T) {
	s := strings.Repeat("a", 1_000_000)
	out, truncated := TruncateText(s, 16*1024)
	if !truncated {
		t.Errorf("expected truncation for 1M string")
	}
	if len(out) != 16*1024 {
		t.Errorf("expected length %d got %d", 16*1024, len(out))
	}
}
