package model

import (
	"time"
)

// WordCountLog 字数统计日志（按天/按次记录）
type WordCountLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"type:char(36);index" json:"user_id"`
	ProjectID uint      `gorm:"index" json:"project_id"`
	DocID     uint      `gorm:"index" json:"doc_id"`
	Delta     int       `json:"delta"` // 正=新增，负=删除
	Total     int       `json:"total"` // 操作后的总字数
	CreatedAt time.Time `json:"created_at"`
}

func (WordCountLog) TableName() string {
	return "word_count_logs"
}

// WritingSession 一场计时写作的记录（⏹ 结束时落库；时长为「前端累计、暂停不计时」口径）
type WritingSession struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      string    `gorm:"type:char(36);index" json:"user_id"`
	ProjectID   uint      `gorm:"index" json:"project_id"`
	StartedAt   time.Time `json:"started_at"`             // 本场开始（墙钟）
	EndedAt     time.Time `json:"ended_at"`               // 本场结束（墙钟）
	DurationSec int       `json:"duration_sec"`           // 纯码字秒数（不含暂停）
	WordsDelta  int       `json:"words_delta"`            // 本场净增字数
	CreatedAt   time.Time `json:"created_at"`
}

func (WritingSession) TableName() string {
	return "writing_sessions"
}
