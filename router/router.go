package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/taills/ai-gateway/middleware"
	"github.com/taills/ai-gateway/relay/controller"
	relaymodel "github.com/taills/ai-gateway/relay/model"
)

// SetRouter wires all HTTP routes to the gin engine.
func SetRouter(r *gin.Engine) {
	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// OpenAI-compatible relay endpoints (require auth)
	v1 := r.Group("/v1", middleware.Auth())
	{
		v1.POST("/chat/completions", relayChatCompletions)
		v1.POST("/completions", relayCompletions)
	}
}

func relayChatCompletions(c *gin.Context) {
	handleRelay(c)
}

func relayCompletions(c *gin.Context) {
	handleRelay(c)
}

func handleRelay(c *gin.Context) {
	bizErr := controller.RelayTextHelper(c)
	if bizErr == nil {
		return
	}
	c.JSON(bizErr.StatusCode, gin.H{"error": bizErr.Error})
}

// ensure relaymodel is used (for compile check only)
var _ = (*relaymodel.ErrorWithStatusCode)(nil)
