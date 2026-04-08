package model

// GeneralOpenAIRequest is a simplified OpenAI-compatible request body.
type GeneralOpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages,omitempty"`
	Prompt      string    `json:"prompt,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
}

// Message is a chat message.
type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// Usage contains token usage for a request.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ErrorWithStatusCode wraps an error with an HTTP status code.
type ErrorWithStatusCode struct {
	Error      OpenAIError `json:"error"`
	StatusCode int         `json:"-"`
}

// OpenAIError represents the error body returned in OpenAI-compatible responses.
type OpenAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    any    `json:"code"`
}
