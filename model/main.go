package model

import (
	"fmt"
	"os"
	"time"

	"github.com/taills/ai-gateway/common/config"
	"github.com/taills/ai-gateway/common/logger"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

// InitDB opens the PostgreSQL (or SQLite fallback) database and runs migrations.
func InitDB() {
	var err error
	dsn := os.Getenv("SQL_DSN")
	if dsn != "" {
		logger.SysLog("using PostgreSQL as database")
		DB, err = gorm.Open(postgres.New(postgres.Config{
			DSN:                  dsn,
			PreferSimpleProtocol: true,
		}), &gorm.Config{PrepareStmt: true})
	} else {
		logger.SysLog("SQL_DSN not set, using SQLite as database")
		DB, err = gorm.Open(sqlite.Open("ai-gateway.db"), &gorm.Config{PrepareStmt: true})
	}
	if err != nil {
		logger.FatalLog("failed to initialize database: " + err.Error())
	}

	sqlDB, err2 := DB.DB()
	if err2 != nil {
		logger.FatalLog("failed to get underlying sql.DB: " + err2.Error())
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Second * 60)

	if config.DebugSQLEnabled {
		DB = DB.Debug()
	}

	if config.IsMasterNode {
		logger.SysLog("database migration started")
		if err := migrateDB(); err != nil {
			logger.FatalLog("failed to migrate database: " + err.Error())
		}
		logger.SysLog("database migrated")
		if err := CreateRootAccountIfNeed(); err != nil {
			logger.FatalLog("database init error: " + err.Error())
		}
	}
}

func migrateDB() error {
	if err := DB.AutoMigrate(&User{}); err != nil {
		return fmt.Errorf("migrate User: %w", err)
	}
	if err := DB.AutoMigrate(&Token{}); err != nil {
		return fmt.Errorf("migrate Token: %w", err)
	}
	if err := DB.AutoMigrate(&Channel{}); err != nil {
		return fmt.Errorf("migrate Channel: %w", err)
	}
	if err := DB.AutoMigrate(&Log{}); err != nil {
		return fmt.Errorf("migrate Log: %w", err)
	}
	return nil
}

// CloseDB closes the database connection.
func CloseDB() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
