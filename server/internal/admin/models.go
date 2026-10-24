package admin

import (
	"time"
)

// Announcement 运营公告（status=1 显示，0=删除）
type Announcement struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"size:200" json:"title"`
	Content   string    `gorm:"type:text" json:"content"`
	Status    int       `gorm:"default:1" json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (Announcement) TableName() string { return "announcements" }

// Feedback 用户反馈
type Feedback struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"type:char(36);index" json:"user_id"`
	Content   string    `gorm:"type:text" json:"content"`
	Contact   string    `gorm:"size:255" json:"contact"` // 可选联系方式
	Resolved  int       `gorm:"default:0" json:"resolved"` // 0=待处理 1=已处理
	CreatedAt time.Time `json:"created_at"`
}

func (Feedback) TableName() string { return "feedbacks" }
