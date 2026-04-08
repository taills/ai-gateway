package meta

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Meta carries per-request metadata through the relay pipeline.
type Meta struct {
	// Identifiers from PostgreSQL
	UserID      int
	Username    string
	TokenID     int
	TokenName   string
	ChannelID   int
	ChannelName string
	ChannelType int

	// Request info
	ModelName  string
	IsStream   bool
	StartTime  time.Time
	RequestID  string

	// Provider info
	APIKey  string
	BaseURL string
}

// GetByContext builds a Meta from values stored in the gin Context.
func GetByContext(c *gin.Context) *Meta {
	m := &Meta{
		StartTime: time.Now(),
	}
	if v, exists := c.Get("user_id"); exists {
		if id, ok := v.(int); ok {
			m.UserID = id
		}
	}
	if v, exists := c.Get("username"); exists {
		if s, ok := v.(string); ok {
			m.Username = s
		}
	}
	if v, exists := c.Get("token_id"); exists {
		if id, ok := v.(int); ok {
			m.TokenID = id
		}
	}
	if v, exists := c.Get("token_name"); exists {
		if s, ok := v.(string); ok {
			m.TokenName = s
		}
	}
	if v, exists := c.Get("channel_id"); exists {
		if id, ok := v.(int); ok {
			m.ChannelID = id
		}
	}
	if v, exists := c.Get("channel_name"); exists {
		if s, ok := v.(string); ok {
			m.ChannelName = s
		}
	}
	if v, exists := c.Get("channel_type"); exists {
		if t, ok := v.(int); ok {
			m.ChannelType = t
		}
	}
	if v, exists := c.Get("request_id"); exists {
		if s, ok := v.(string); ok {
			m.RequestID = s
		}
	}
	if v, exists := c.Get("api_key"); exists {
		if s, ok := v.(string); ok {
			m.APIKey = s
		}
	}
	if v, exists := c.Get("base_url"); exists {
		if s, ok := v.(string); ok {
			m.BaseURL = s
		}
	}
	return m
}
