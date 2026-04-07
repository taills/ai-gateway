package model

import "gorm.io/gorm"

// Channel represents an upstream LLM provider channel.
type Channel struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name         string `json:"name" gorm:"size:255;not null;default:''"`
	Type         int    `json:"type" gorm:"default:1;index"`
	Key          string `json:"key" gorm:"size:4096;not null;default:''"`
	BaseURL      string `json:"base_url" gorm:"size:512;default:''"`
	Models       string `json:"models" gorm:"size:4096;default:''"`
	Status       int    `json:"status" gorm:"default:1;index"`
	UsedQuota    int64  `json:"used_quota" gorm:"default:0"`
	RequestCount int    `json:"request_count" gorm:"default:0"`
}

const (
	ChannelTypeOpenAI    = 1
	ChannelTypeAzure     = 3
	ChannelTypeAnthropic = 14
)

const (
	ChannelStatusEnabled  = 1
	ChannelStatusDisabled = 2
)

func GetChannelById(id int) (*Channel, error) {
	var channel Channel
	if err := DB.First(&channel, id).Error; err != nil {
		return nil, err
	}
	return &channel, nil
}

func UpdateChannelUsedQuota(channelId int, quota int64) {
	DB.Model(&Channel{}).Where("id = ?", channelId).Updates(map[string]any{
		"used_quota":    gorm.Expr("used_quota + ?", quota),
		"request_count": gorm.Expr("request_count + 1"),
	})
}
