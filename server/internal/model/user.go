package model

import (
	"time"
)

// User 用户主表：只放用户资料和主联系方式，登录方式放 user_auth_identities
// 后续手机号、微信扫码登录都不需要改 users 主结构
type User struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       string    `gorm:"type:char(36);uniqueIndex;size:36" json:"user_id"`
	Email        string    `gorm:"size:255;index" json:"email"`
	Phone        string    `gorm:"size:20;index" json:"phone"`
	Nickname     string    `gorm:"size:50" json:"nickname"`
	PasswordHash string    `gorm:"size:255" json:"-"` // 兼容旧账号；新登录密码同步写入 identity.credential_hash
	Avatar       string    `gorm:"size:500" json:"avatar"`
	Plan         string    `gorm:"size:20;default:'free'" json:"plan"` // free / vip / svip
	Status       int       `gorm:"default:1" json:"status"`            // 1=正常 0=禁用
	IsAdmin      bool      `gorm:"default:false" json:"is_admin"`      // 后台管理员标记
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

// RefreshToken 刷新令牌表（支持多端+踢设备）
type RefreshToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"type:char(36);index" json:"user_id"`
	Token     string    `gorm:"uniqueIndex;size:512" json:"token"`
	DeviceID  string    `gorm:"size:100" json:"device_id"`
	IP        string    `gorm:"size:45" json:"ip"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// AuthIdentity 登录身份表：邮箱、手机号、微信等统一放这里
// Provider: email / phone / wechat
// Identifier: 邮箱地址 / 手机号 / openid或unionid
// CredentialHash: 邮箱密码登录的 bcrypt hash；验证码/微信类可为空
type AuthIdentity struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         string     `gorm:"type:char(36);index" json:"user_id"`
	Provider       string     `gorm:"size:30;index:idx_provider_identifier,unique" json:"provider"`
	Identifier     string     `gorm:"size:255;index:idx_provider_identifier,unique" json:"identifier"`
	CredentialHash string     `gorm:"size:255" json:"-"`
	VerifiedAt     *time.Time `json:"verified_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (AuthIdentity) TableName() string {
	return "user_auth_identities"
}
