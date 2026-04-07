package model

import (
	"context"

	"github.com/taills/ai-gateway/common/config"
	"github.com/taills/ai-gateway/common/helper"
	"github.com/taills/ai-gateway/common/logger"
)

const (
	LogTypeUnknown = 0
	LogTypeTopup   = 1
	LogTypeConsume = 2
	LogTypeManage  = 3
	LogTypeSystem  = 4
)

// Log stores consume/topup/manage events in PostgreSQL.
type Log struct {
	Id               int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId           int    `json:"user_id" gorm:"index"`
	CreatedAt        int64  `json:"created_at" gorm:"index"`
	Type             int    `json:"type" gorm:"index"`
	Content          string `json:"content" gorm:"size:4096;default:''"`
	Username         string `json:"username" gorm:"size:64;default:''"`
	TokenName        string `json:"token_name" gorm:"size:64;default:''"`
	ModelName        string `json:"model_name" gorm:"size:128;index;default:''"`
	Quota            int64  `json:"quota" gorm:"default:0"`
	PromptTokens     int    `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens int    `json:"completion_tokens" gorm:"default:0"`
	ChannelId        int    `json:"channel_id" gorm:"index"`
	RequestId        string `json:"request_id" gorm:"size:64;default:''"`
	ElapsedTime      int64  `json:"elapsed_time" gorm:"default:0"`
	IsStream         bool   `json:"is_stream" gorm:"default:false"`
}

func RecordConsumeLog(ctx context.Context, log *Log) {
	if !config.LogConsumeEnabled {
		return
	}
	log.Username = GetUsernameById(log.UserId)
	log.CreatedAt = helper.GetTimestamp()
	log.Type = LogTypeConsume
	log.RequestId = helper.GetRequestID(ctx)
	if err := DB.Create(log).Error; err != nil {
		logger.Errorf(ctx, "failed to record consume log: %s", err.Error())
	}
}

func GetAllLogs(logType int, startTimestamp, endTimestamp int64, modelName, username, tokenName string, startIdx, num, channel int) ([]*Log, error) {
	var logs []*Log
	tx := DB.Model(&Log{})
	if logType != LogTypeUnknown {
		tx = tx.Where("type = ?", logType)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if channel != 0 {
		tx = tx.Where("channel_id = ?", channel)
	}
	err := tx.Order("id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	return logs, err
}
