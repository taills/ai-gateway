package clickhouse

import (
	"testing"
)

func TestParseDSN_ClickHouseScheme(t *testing.T) {
	base, db, user, pass, err := parseDSN("clickhouse://user:pass@localhost:9000/mydb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if base != "http://localhost:8123" {
		t.Errorf("expected base %q got %q", "http://localhost:8123", base)
	}
	if db != "mydb" {
		t.Errorf("expected db %q got %q", "mydb", db)
	}
	if user != "user" {
		t.Errorf("expected user %q got %q", "user", user)
	}
	if pass != "pass" {
		t.Errorf("expected pass %q got %q", "pass", pass)
	}
}

func TestParseDSN_HTTP(t *testing.T) {
	base, db, user, pass, err := parseDSN("http://u:p@clickhouse-host:8123/analytics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if base != "http://clickhouse-host:8123" {
		t.Errorf("unexpected base: %q", base)
	}
	if db != "analytics" {
		t.Errorf("unexpected db: %q", db)
	}
	if user != "u" || pass != "p" {
		t.Errorf("unexpected credentials: %q %q", user, pass)
	}
}

func TestParseDSN_InvalidScheme(t *testing.T) {
	_, _, _, _, err := parseDSN("ftp://host/db")
	if err == nil {
		t.Error("expected error for unsupported scheme")
	}
}

func TestParseDSN_MissingDB(t *testing.T) {
	base, db, _, _, err := parseDSN("http://host:8123/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if base != "http://host:8123" {
		t.Errorf("unexpected base: %q", base)
	}
	if db != "" {
		t.Errorf("expected empty db, got %q", db)
	}
}
