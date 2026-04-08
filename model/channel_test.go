package model

import (
	"strings"
	"testing"
)

// channelSupportsModel mirrors the LIKE-based matching logic used in
// GetChannelByModel, so we can unit-test it without a real database.
func channelSupportsModel(models, modelName string) bool {
	if modelName == "" {
		return false
	}
	// Exact match
	if models == modelName {
		return true
	}
	parts := strings.Split(models, ",")
	for _, p := range parts {
		if strings.TrimSpace(p) == modelName {
			return true
		}
	}
	return false
}

func TestChannelSupportsModel_ExactMatch(t *testing.T) {
	if !channelSupportsModel("gpt-4o-mini", "gpt-4o-mini") {
		t.Error("exact single model should match")
	}
}

func TestChannelSupportsModel_FirstInList(t *testing.T) {
	if !channelSupportsModel("gpt-4o-mini,gpt-4o,gpt-3.5-turbo", "gpt-4o-mini") {
		t.Error("first model in list should match")
	}
}

func TestChannelSupportsModel_MiddleInList(t *testing.T) {
	if !channelSupportsModel("gpt-4o,gpt-4o-mini,gpt-3.5-turbo", "gpt-4o-mini") {
		t.Error("middle model in list should match")
	}
}

func TestChannelSupportsModel_LastInList(t *testing.T) {
	if !channelSupportsModel("gpt-4o,gpt-3.5-turbo,gpt-4o-mini", "gpt-4o-mini") {
		t.Error("last model in list should match")
	}
}

func TestChannelSupportsModel_NoMatch(t *testing.T) {
	if channelSupportsModel("gpt-4o,gpt-3.5-turbo", "claude-3") {
		t.Error("model not in list should not match")
	}
}

func TestChannelSupportsModel_EmptyModel(t *testing.T) {
	if channelSupportsModel("gpt-4o,gpt-3.5-turbo", "") {
		t.Error("empty model name should not match")
	}
}

func TestChannelSupportsModel_EmptyModels(t *testing.T) {
	if channelSupportsModel("", "gpt-4o") {
		t.Error("channel with empty models field should not match")
	}
}

func TestChannelSupportsModel_PartialNameShouldNotMatch(t *testing.T) {
	// "gpt-4o" should not match "gpt-4o-mini"
	if channelSupportsModel("gpt-4o-mini", "gpt-4o") {
		t.Error("partial prefix match should not be treated as a match")
	}
}
