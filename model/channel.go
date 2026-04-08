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

// GetChannelByModel returns the first enabled channel that lists modelName in
// its comma-separated models field, or — if no channel has that model listed —
// the first enabled channel overall.  This gives a sensible default without
// requiring the caller to know the channel ID.
func GetChannelByModel(modelName string) (*Channel, error) {
	var channel Channel
	// Try exact model match first (models field is comma-separated)
	if modelName != "" {
		if err := DB.Where("status = ? AND (models LIKE ? OR models LIKE ? OR models LIKE ? OR models = ?)",
			ChannelStatusEnabled,
			modelName+",%",
			"%,"+modelName+",%",
			"%,"+modelName,
			modelName,
		).First(&channel).Error; err == nil {
			return &channel, nil
		}
	}
	// Fallback: first enabled channel
	if err := DB.Where("status = ?", ChannelStatusEnabled).First(&channel).Error; err != nil {
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
