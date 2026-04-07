package main

import (
	"os"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/taills/ai-gateway/clickhouse"
	"github.com/taills/ai-gateway/common/config"
	"github.com/taills/ai-gateway/common/logger"
	"github.com/taills/ai-gateway/middleware"
	"github.com/taills/ai-gateway/model"
	"github.com/taills/ai-gateway/router"
)

func main() {
	logger.SysLog("AI Gateway starting…")

	if os.Getenv("GIN_MODE") != gin.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}

	// ── PostgreSQL ───────────────────────────────────────────────────────────
	model.InitDB()
	defer func() {
		if err := model.CloseDB(); err != nil {
			logger.SysLogf("failed to close database: %v", err)
		}
	}()

	// ── ClickHouse ───────────────────────────────────────────────────────────
	clickhouse.Init()
	defer clickhouse.Close()

	// ── HTTP server ──────────────────────────────────────────────────────────
	server := gin.New()
	server.Use(gin.Recovery())
	server.Use(middleware.RequestID())
	server.Use(middleware.Logger())

	router.SetRouter(server)

	port := os.Getenv("PORT")
	if port == "" {
		port = strconv.Itoa(config.Port)
	}
	logger.SysLogf("server started on http://localhost:%s", port)
	if err := server.Run(":" + port); err != nil {
		logger.FatalLog("failed to start HTTP server: " + err.Error())
	}
}
