package model

import (
	"time"
)

// User 用户表
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex;size:255" json:"email"`
	Phone        string    `gorm:"size:20;index" json:"phone"`
	Nickname     string    `gorm:"size:50" json:"nickname"`
	PasswordHash string    `gorm:"size:255" json:"-"`
	Avatar       string    `gorm:"size:500" json:"avatar"`
	Plan         string    `gorm:"size:20;default:'free'" json:"plan"` // free / vip / svip
	Status       int       `gorm:"default:1" json:"status"`              // 1=正常 0=禁用
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

// RefreshToken 刷新令牌表（支持多端+踢设备）
type RefreshToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Token     string    `gorm:"uniqueIndex;size:512" json:"token"`
	DeviceID  string    `gorm:"size:100" json:"device_id"`
	IP        string    `gorm:"size:45" json:"ip"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
