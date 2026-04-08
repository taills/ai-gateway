package model

import (
	"github.com/taills/ai-gateway/common/helper"
	"gorm.io/gorm"
)

const (
	TokenStatusEnabled  = 1
	TokenStatusDisabled = 2
)

// Token is an API key / access token that belongs to a user.
type Token struct {
	Id             int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId         int    `json:"user_id" gorm:"index;not null"`
	Key            string `json:"key" gorm:"uniqueIndex;size:64;not null"`
	Status         int    `json:"status" gorm:"default:1"`
	Name           string `json:"name" gorm:"size:255;default:''"`
	CreatedTime    int64  `json:"created_time" gorm:"default:0"`
	AccessedTime   int64  `json:"accessed_time" gorm:"default:0"`
	ExpiredTime    int64  `json:"expired_time" gorm:"default:-1"` // -1 means never
	RemainQuota    int64  `json:"remain_quota" gorm:"default:0"`
	UnlimitedQuota bool   `json:"unlimited_quota" gorm:"default:false"`
}

func GetTokenByKey(key string) (*Token, error) {
	var token Token
	now := helper.GetTimestamp()
	// expired_time == -1 means never expires; otherwise reject if past expiry.
	if err := DB.Where(
		"key = ? AND status = ? AND (expired_time = -1 OR expired_time > ?)",
		key, TokenStatusEnabled, now,
	).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func PreConsumeTokenQuota(tokenId int, quota int64) error {
	return DB.Model(&Token{}).Where("id = ? AND (unlimited_quota = true OR remain_quota >= ?)", tokenId, quota).
		Update("remain_quota", gorm.Expr("remain_quota - ?", quota)).Error
}

func PostConsumeTokenQuota(tokenId int, quotaDelta int64) error {
	// Only deduct from tokens that have a finite quota; unlimited tokens are skipped.
	return DB.Model(&Token{}).Where("id = ? AND unlimited_quota = false", tokenId).
		Update("remain_quota", gorm.Expr("remain_quota - ?", quotaDelta)).Error
}

func UpdateTokenAccessedTime(tokenId int) {
	DB.Model(&Token{}).Where("id = ?", tokenId).Update("accessed_time", helper.GetTimestamp())
}
