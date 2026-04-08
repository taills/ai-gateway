package model

import (
	"errors"

	"github.com/taills/ai-gateway/common/helper"
	"github.com/taills/ai-gateway/common/logger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	RoleGuestUser = 0
	RoleCommonUser = 1
	RoleAdminUser = 10
	RoleRootUser  = 100
)

const (
	UserStatusEnabled  = 1
	UserStatusDisabled = 2
)

// User stores business user information in PostgreSQL.
type User struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Username    string `json:"username" gorm:"uniqueIndex;size:64;not null"`
	Password    string `json:"password,omitempty" gorm:"size:255;not null"`
	DisplayName string `json:"display_name" gorm:"size:255;default:''"`
	Role        int    `json:"role" gorm:"index;default:1"`
	Status      int    `json:"status" gorm:"default:1"`
	Email       string `json:"email" gorm:"size:255;default:''"`
	AccessToken string `json:"access_token" gorm:"uniqueIndex;size:64;default:''"`
	Quota       int64  `json:"quota" gorm:"default:0"`
	UsedQuota   int64  `json:"used_quota" gorm:"default:0"`
	RequestCount int   `json:"request_count" gorm:"default:0"`
	CreatedAt   int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func CreateRootAccountIfNeed() error {
	var user User
	if err := DB.First(&user).Error; err != nil {
		logger.SysLog("no user exists, creating a root user: username=root, password=123456")
		hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		rootUser := User{
			Username:    "root",
			Password:    string(hash),
			Role:        RoleRootUser,
			Status:      UserStatusEnabled,
			DisplayName: "Root User",
			AccessToken: helper.NewRequestID(),
			Quota:       500_000_000_000_000,
		}
		return DB.Create(&rootUser).Error
	}
	return nil
}

func GetUserById(id int) (*User, error) {
	var user User
	if err := DB.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func GetUsernameById(id int) string {
	var user User
	DB.Select("username").First(&user, id)
	return user.Username
}

func UpdateUserUsedQuotaAndRequestCount(userId int, quota int64) {
	DB.Model(&User{}).Where("id = ?", userId).Updates(map[string]any{
		"used_quota":    gorm.Expr("used_quota + ?", quota),
		"request_count": gorm.Expr("request_count + 1"),
	})
}

func ValidateUserCredentials(username, password string) (*User, error) {
	var user User
	if err := DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}
	if user.Status != UserStatusEnabled {
		return nil, errors.New("user is disabled")
	}
	return &user, nil
}
