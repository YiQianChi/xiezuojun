// AI Gateway 核心类型定义
// 参考 ARCHITECTURE.md v1.1 第六章设计

package gateway

import (
	"context"
	"time"
)

// ========== 上下文金字塔 ==========

// ContextLevel 上下文层级
type ContextLevel int

const (
	LevelMemory   ContextLevel = iota // 作者记忆层
	LevelProject                      // 项目层（设定、角色卡）
	LevelScene                        // 场景层（前文）
	LevelTask                         // 任务层（当前指令）
)

// ContextBlock 上下文块
type ContextBlock struct {
	Level    ContextLevel
	Content  string
	Tokens   int     // 预估 token 数
	Priority float64 // 0-1，越重要越优先保留
	Source   string  // 来源标识，如 "characters.md", "preceding-chapters"
}

// BuildContextRequest 构建上下文请求
type BuildContextRequest struct {
	ProjectID    string
	DocumentID   string
	UserID       string         // 用户 ID（用于读取记忆文件）
	UserPrompt   string         // 用户输入
	MemoryBlocks []ContextBlock // 作者记忆层
	ProjectMeta  string         // 项目设定（PROMPT.md）
	SceneText    string         // 前文（自动截断）
	AIMode       string         // 模式：continue | rewrite | polish | expand | outline
}

// BuildContextResult 构建结果
type BuildContextResult struct {
	SystemPrompt string         // 系统提示
	Messages     []Message      // 完整消息列表
	ContextMap   []ContextBlock // 用了哪些上下文块（用于调试/分析）
	TotalTokens  int            // 总 token 估算
}

// ========== AI 模式 ==========

// ModeTemplate 模式模板
type ModeTemplate struct {
	Name        string
	SystemPrompt string
	Temperature float64
	MaxTokens   int
	StyleHints  []string // 风格提示词
}

var ModeTemplates = map[string]ModeTemplate{
	"continue": {
		Name:         "续写",
		SystemPrompt: "你是一位资深小说编辑和创作助手。请根据前文的情节和文风，自然地续写后续内容。保持角色行为一致性，注意伏笔呼应。",
		Temperature:  0.85,
		MaxTokens:    2000,
		StyleHints:   []string{"保持文风一致", "注意情节连贯"},
	},
	"rewrite": {
		Name:         "改写",
		SystemPrompt: "你是一位资深小说编辑。请改写以下内容，保持核心情节不变，但提升文笔质量、优化句式结构、增强画面感。",
		Temperature:  0.7,
		MaxTokens:    2000,
		StyleHints:   []string{"提升文笔", "优化结构"},
	},
	"polish": {
		Name:         "润色",
		SystemPrompt: "你是一位资深文字编辑。请润色以下内容，修正语法错误、消除冗余、提升表达精准度，保持原意不变。",
		Temperature:  0.5,
		MaxTokens:    2000,
		StyleHints:   []string{"精准表达", "消除冗余"},
	},
	"expand": {
		Name:         "扩写",
		SystemPrompt: "你是一位擅长细节描写的作家。请扩写以下内容，增加环境渲染、心理描写、动作细节，让场景更丰满生动。",
		Temperature:  0.9,
		MaxTokens:    3000,
		StyleHints:   []string{"增加细节", "渲染氛围"},
	},
	"outline": {
		Name:         "大纲",
		SystemPrompt: "你是一位资深小说策划。请根据标题和简介，生成详细的小说大纲。按「卷→章」结构输出，包含每章的标题和概要。",
		Temperature:  0.8,
		MaxTokens:    4000,
		StyleHints:   []string{"结构清晰", "逻辑自洽"},
	},
}

// ========== 模型路由 ==========

// ModelConfig 模型配置
type ModelConfig struct {
	Name       string
	Provider   string // zhipu | openai | anthropic
	ModelID    string // glm-5-turbo | gpt-4 | claude-3-opus
	MaxTokens  int
	CostPer1K  float64 // 每 1K token 成本（元）
	Priority   int     // 优先级，数字越小越优先
	Capability ModelCapability
}

// ModelCapability 模型能力
type ModelCapability struct {
	ContextWindow int    // 上下文窗口大小
	SupportsJSON  bool   // 是否支持 JSON 模式
	SupportsVision bool  // 是否支持图片
	Quality       string // high | medium | low
}

var DefaultModels = []ModelConfig{
	{
		Name:     "GLM-5.2",
		Provider: "volcengine",
		ModelID:  "GLM-5.2",
		MaxTokens: 8192,
		CostPer1K: 0.006,
		Priority:  -1,
		Capability: ModelCapability{
			ContextWindow: 128000,
			SupportsJSON:  true,
			Quality:       "high",
		},
	},
	{
		Name:     "GLM-4.7",
		Provider: "volcengine",
		ModelID:  "GLM-4.7",
		MaxTokens: 8192,
		CostPer1K: 0.005,
		Priority:  0,
		Capability: ModelCapability{
			ContextWindow: 128000,
			SupportsJSON:  true,
			Quality:       "high",
		},
	},
	{
		Name:     "DeepSeek-V3.2",
		Provider: "volcengine",
		ModelID:  "DeepSeek-V3.2",
		MaxTokens: 8192,
		CostPer1K: 0.005,
		Priority:  1,
		Capability: ModelCapability{
			ContextWindow: 128000,
			SupportsJSON:  true,
			Quality:       "high",
		},
	},
	{
		Name:     "Doubao-Seed-2.0-Code",
		Provider: "volcengine",
		ModelID:  "Doubao-Seed-2.0-Code",
		MaxTokens: 8192,
		CostPer1K: 0.003,
		Priority:  2,
		Capability: ModelCapability{
			ContextWindow: 128000,
			SupportsJSON:  false,
			Quality:       "medium",
		},
	},
}

// RouteRequest 路由请求
type RouteRequest struct {
	AIMode        string
	ContextTokens int
	RequireJSON   bool
	UserTier      string // free | pro | enterprise
}

// RouteResult 路由结果
type RouteResult struct {
	Model       ModelConfig
	Reason      string // 选择原因
	EstimatedCost float64
}

// ========== 用量追踪 ==========

// UsageRecord 用量记录
type UsageRecord struct {
	UserID      string
	ProjectID   string
	ModelID     string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	Cost       float64
	AIMode     string
	Timestamp  time.Time
	DurationMs int64
}

// QuotaConfig 配额配置
type QuotaConfig struct {
	FreeTier      Quota // 免费版
	ProTier       Quota // 专业版
	EnterpriseTier Quota // 企业版
}

type Quota struct {
	DailyTokens    int     // 每日 token 限额
	MonthlyTokens  int     // 每月 token 限额
	ConcurrentReqs int     // 并发请求数
	MaxContextLen  int     // 最大上下文长度
}

var DefaultQuota = QuotaConfig{
	FreeTier: Quota{
		DailyTokens:    10000,
		MonthlyTokens:  300000,
		ConcurrentReqs: 2,
		MaxContextLen:  4000,
	},
	ProTier: Quota{
		DailyTokens:    100000,
		MonthlyTokens:  3000000,
		ConcurrentReqs: 5,
		MaxContextLen:  16000,
	},
	EnterpriseTier: Quota{
		DailyTokens:    1000000,
		MonthlyTokens:  30000000,
		ConcurrentReqs: 20,
		MaxContextLen:  128000,
	},
}

// ========== 通用消息类型 ==========

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ========== Gateway 接口 ==========

type Gateway interface {
	// BuildContext 构建上下文
	BuildContext(ctx context.Context, req BuildContextRequest) (*BuildContextResult, error)

	// Generate 生成内容（非流式）
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error)

	// GenerateStream 流式生成
	GenerateStream(ctx context.Context, req GenerateRequest) (<-chan StreamChunk, error)

	// CheckQuota 检查配额
	CheckQuota(userID string, tier string) (*QuotaStatus, error)

	// TrackUsage 记录用量
	TrackUsage(record UsageRecord) error
}

type GenerateRequest struct {
	ProjectID  string
	DocumentID string
	UserID     string         // 用户 ID（用于读取记忆）
	AIMode     string
	UserPrompt string
	SceneText  string         // 文档内容/前文（由 API handler 填入）
	Context    []ContextBlock // 可选：预构建的上下文
	Style      string         // 风格偏好
}

type GenerateResult struct {
	Content      string
	Usage        UsageRecord
	ModelUsed    string
	ContextTokens int
}

type StreamChunk struct {
	Content string
	Error   string
	Done    bool
}

type QuotaStatus struct {
	Allowed       bool
	Remaining     int
	UsedToday     int
	LimitToday    int
	UsedMonth     int
	LimitMonth    int
	Reason        string
}
