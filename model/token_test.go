package model

import (
	"testing"
	"time"
)

// ── Token expiry helper ──────────────────────────────────────────────────────

// tokenIsExpired mirrors the expiry logic used in GetTokenByKey.
// -1 means never expires; anything > 0 is a Unix timestamp.
func tokenIsExpired(expiredTime int64, now int64) bool {
	if expiredTime == -1 {
		return false
	}
	return expiredTime <= now
}

func TestTokenIsExpired_NeverExpires(t *testing.T) {
	now := time.Now().Unix()
	if tokenIsExpired(-1, now) {
		t.Error("token with expired_time=-1 should never expire")
	}
}

func TestTokenIsExpired_NotYetExpired(t *testing.T) {
	future := time.Now().Unix() + 3600
	if tokenIsExpired(future, time.Now().Unix()) {
		t.Error("token with future expiry should not be expired")
	}
}

func TestTokenIsExpired_AlreadyExpired(t *testing.T) {
	past := time.Now().Unix() - 1
	if !tokenIsExpired(past, time.Now().Unix()) {
		t.Error("token with past expiry should be expired")
	}
}

func TestTokenIsExpired_ExpiresNow(t *testing.T) {
	now := time.Now().Unix()
	// exactly at expiry boundary: expired_time == now means expired
	if !tokenIsExpired(now, now) {
		t.Error("token expiring exactly at 'now' should be treated as expired")
	}
}
