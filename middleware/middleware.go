package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/taills/ai-gateway/common/helper"
	"github.com/taills/ai-gateway/common/logger"
)

// RequestID injects a unique request ID into the context and response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = helper.NewRequestID()
		}
		ctx := helper.WithRequestID(c.Request.Context(), id)
		c.Request = c.Request.WithContext(ctx)
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// Logger logs basic request info.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		logger.SysLogf("%s %s %d", c.Request.Method, c.Request.URL.Path, c.Writer.Status())
	}
}

// Auth validates the Bearer token in the Authorization header and sets user
// context values. For simplicity this implementation looks up the token in
// PostgreSQL directly; a production version would add caching.
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := extractBearerToken(c)
		if key == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": gin.H{"message": "missing authorization", "type": "auth_error"}})
			return
		}

		// Import here to avoid import cycle
		token, err := lookupToken(key)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": gin.H{"message": "invalid or expired token", "type": "auth_error"}})
			return
		}

		go touchTokenAsync(token.Id)

		c.Set("user_id", token.UserId)
		c.Set("token_id", token.Id)
		c.Set("token_name", token.Name)
		c.Next()
	}
}

func extractBearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return ""
}
