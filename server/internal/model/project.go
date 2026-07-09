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

// StoryBranch 故事线/剧情分支。同一本书可从某一章分出新故事线。
type StoryBranch struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ProjectID      uint      `gorm:"index" json:"project_id"`
	Name           string    `gorm:"size:100" json:"name"`
	BaseBranchID   *uint     `gorm:"index" json:"base_branch_id"`
	ForkDocumentID *uint     `gorm:"index" json:"fork_document_id"`
	Status         int       `gorm:"default:1" json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (StoryBranch) TableName() string {
	return "story_branches"
}

// Document 文档/章节
type Document struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ProjectID  uint      `gorm:"index" json:"project_id"`
	BranchID   *uint     `gorm:"index" json:"branch_id"` // nil=全故事线共享资料；有值=属于某条故事线
	ParentID   *uint     `gorm:"index" json:"parent_id"` // 支持卷/章嵌套
	Type       string    `gorm:"size:20" json:"type"`    // volume | chapter | note | outline | characters | worldbuilding | storyline | chapter_outline
	TabScope   string    `gorm:"size:50;default:chapter" json:"tab_scope"` // 逗号分隔: chapter,setting — 文件夹在哪些 Tab 显示
	Icon       string    `gorm:"size:10" json:"icon"`     // 用户自选 emoji 图标，空则按 type 默认
	Title      string    `gorm:"size:200" json:"title"`
	SortOrder  int       `gorm:"default:0" json:"sort_order"`
	WordCount  int       `gorm:"default:0" json:"word_count"`
	SearchText string    `gorm:"type:mediumtext" json:"-"` // 搜索缓存；正文权威源仍是文件
	AIInclude  *bool     `gorm:"default:true" json:"ai_include"` // 是否参与 AI 上下文（设定文档用），nil=true
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
	Label           string    `gorm:"size:50" json:"label"`
	ContentSnapshot string    `gorm:"type:longtext" json:"content_snapshot"`
	WordCount       int       `json:"word_count"`
	CreatedAt       time.Time `json:"created_at"`
}

func (DocumentVersion) TableName() string {
	return "document_versions"
}

// StoryThread 伏笔/钩子/承诺/悬念追踪
type StoryThread struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ProjectID       uint      `gorm:"index" json:"project_id"`
	Type            string    `gorm:"size:20" json:"type"`              // foreshadowing | hook | promise | suspense
	Title           string    `gorm:"size:200" json:"title"`
	Description     string    `gorm:"type:text" json:"description"`
	Status          string    `gorm:"size:20;default:open" json:"status"` // open | resolved | abandoned
	PlantedChapter  *int      `json:"planted_chapter"`
	ResolvedChapter *int      `json:"resolved_chapter"`
	Urgency         int       `gorm:"default:50" json:"urgency"` // 0-100
	SortOrder       int       `gorm:"default:0" json:"sort_order"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (StoryThread) TableName() string {
	return "story_threads"
}
