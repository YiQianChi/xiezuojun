package model

import (
	"time"
)

// WordCountLog 字数统计日志（按天/按次记录）
type WordCountLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	ProjectID uint      `gorm:"index" json:"project_id"`
	DocID     uint      `gorm:"index" json:"doc_id"`
	Delta     int       `json:"delta"` // 正=新增，负=删除
	Total     int       `json:"total"` // 操作后的总字数
	CreatedAt time.Time `json:"created_at"`
}

func (WordCountLog) TableName() string {
	return "word_count_logs"
}
