package model

import (
	"time"
)

// Project 写作项目（一本书/一个剧本）
type Project struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"type:char(36);index" json:"user_id"`
	Name      string    `gorm:"size:200" json:"name"`
	CoverURL  string    `gorm:"size:500" json:"cover_url"`
	Genre     string    `gorm:"size:50" json:"genre"`    // 类型：小说/剧本/散文
	Tags      string    `gorm:"size:500" json:"tags"`    // 逗号分隔的标签
	Status    int       `gorm:"default:1" json:"status"` // 1=正常 0=删除
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Project) TableName() string {
	return "projects"
}

// Document 文档/章节
type Document struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ProjectID  uint      `gorm:"index" json:"project_id"`
	ParentID   *uint     `gorm:"index" json:"parent_id"` // 支持卷/章嵌套
	Type       string    `gorm:"size:20" json:"type"`    // volume | chapter | note
	Title      string    `gorm:"size:200" json:"title"`
	SortOrder  int       `gorm:"default:0" json:"sort_order"`
	WordCount  int       `gorm:"default:0" json:"word_count"`
	SearchText string    `gorm:"type:mediumtext" json:"-"` // 搜索缓存；正文权威源仍是文件
	Status     int       `gorm:"default:1" json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (Document) TableName() string {
	return "documents"
}

// DocumentVersion 历史快照
type DocumentVersion struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	DocumentID      uint      `gorm:"index" json:"document_id"`
	ContentSnapshot string    `gorm:"type:longtext" json:"content_snapshot"`
	WordCount       int       `json:"word_count"`
	CreatedAt       time.Time `json:"created_at"`
}

func (DocumentVersion) TableName() string {
	return "document_versions"
}
