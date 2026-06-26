package model

import (
	"time"
)

// UsageRecord AI 用量记录
type UsageRecord struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           string    `gorm:"size:50;index" json:"user_id"`
	ProjectID        string    `gorm:"size:50;index" json:"project_id"`
	ModelID          string    `gorm:"size:100" json:"model_id"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	Cost             float64   `json:"cost"`
	AIMode           string    `gorm:"size:20" json:"ai_mode"`
	DurationMs       int64     `json:"duration_ms"`
	CreatedAt        time.Time `gorm:"index" json:"created_at"`
}

func (UsageRecord) TableName() string {
	return "usage_records"
}
