// Gateway 主控制器
// 协调 ContextBuilder + PromptEngineering + ModelRouter + UsageTracker

package gateway

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"

	"xiezuojun-server/internal/ai"
)

// DefaultGateway 默认网关实现
type DefaultGateway struct {
	aiClient        *ai.Client
	contextBuilder  *ContextBuilder
	modelRouter     *ModelRouter
	usageTracker    *UsageTracker
	docContentLoader func(docID uint64) (string, error)                // 加载文档正文
	settingLoader    func(projectID uint) ([]SettingDoc, error)        // 加载项目的设定文档

	// 并发控制
	semaphores map[string]chan struct{} // userID -> semaphore
	mu         sync.RWMutex
}

// SetDocContentLoader 设置文档内容加载器（从 main.go 调用）
func (g *DefaultGateway) SetDocContentLoader(fn func(docID uint64) (string, error)) {
	g.docContentLoader = fn
}

// SetSettingLoader 设置项目设定文档加载器
func (g *DefaultGateway) SetSettingLoader(fn func(projectID uint) ([]SettingDoc, error)) {
	g.settingLoader = fn
}

// NewGateway 创建网关
func NewGateway(aiClient *ai.Client, memoryReader MemoryReader) *DefaultGateway {
	return &DefaultGateway{
		aiClient:       aiClient,
		contextBuilder: NewContextBuilder(memoryReader),
		modelRouter:    NewModelRouter(),
		usageTracker:   NewUsageTracker(),
		semaphores:     make(map[string]chan struct{}),
	}
}

// BuildContext 构建上下文
func (g *DefaultGateway) BuildContext(ctx context.Context, req BuildContextRequest) (*BuildContextResult, error) {
	return g.contextBuilder.Build(ctx, req)
}

// Generate 非流式生成
func (g *DefaultGateway) Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error) {
	// 配额检查已关闭（测试阶段不限制）
	_ = req.UserTier

	// 1. 构建上下文
	ctxReq := BuildContextRequest{
		ProjectID:  req.ProjectID,
		DocumentID: req.DocumentID,
		UserID:     req.UserID,
		UserPrompt: req.UserPrompt,
		AIMode:     req.AIMode,
		SceneText:  req.SceneText,
	}
	if len(req.Context) > 0 {
		ctxReq.MemoryBlocks = req.Context
	}
	// 加载项目设定文档（大纲/人物/世界观）—— 默认开启
	if req.IncludeSettings && g.settingLoader != nil && req.ProjectID != "" {
		if pid, err := parseProjectID(req.ProjectID); err == nil && pid > 0 {
			if docs, err := g.settingLoader(pid); err == nil {
				ctxReq.SettingDocs = docs
			}
		}
	}

	ctxResult, err := g.contextBuilder.Build(ctx, ctxReq)
	if err != nil {
		return nil, fmt.Errorf("context build failed: %w", err)
	}

	// 3. 路由模型
	routeReq := RouteRequest{
		AIMode:        req.AIMode,
		ContextTokens: ctxResult.TotalTokens,
		RequireJSON:   req.AIMode == "outline" || req.AIMode == "review",
		UserTier:      "free",
	}
	routeResult := g.modelRouter.Route(routeReq)

	// 4. 获取并发许可
	if err := g.acquireSlot(ctx, req.ProjectID, routeResult.Model); err != nil {
		return nil, err
	}
	defer g.releaseSlot(req.ProjectID)

	// 5. 调用 AI
	start := time.Now()
	aiMessages := make([]ai.Message, len(ctxResult.Messages))
	for i, m := range ctxResult.Messages {
		aiMessages[i] = ai.Message{Role: m.Role, Content: m.Content}
	}
	content, err := g.aiClient.Complete(ctx, routeResult.Model.ModelID, aiMessages)
	duration := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("ai generation failed: %w", err)
	}

	// 6. 估算用量并记录
	promptTokens := ctxResult.TotalTokens
	completionTokens := len(content) / 4
	totalTokens := promptTokens + completionTokens
	cost := float64(totalTokens) / 1000 * routeResult.Model.CostPer1K

	usage := UsageRecord{
		UserID:           req.UserID,
		ProjectID:        req.ProjectID,
		ModelID:          routeResult.Model.ModelID,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		Cost:             cost,
		AIMode:           req.AIMode,
		Timestamp:        time.Now(),
		DurationMs:       duration.Milliseconds(),
	}

	if err := g.usageTracker.Track(usage); err != nil {
		fmt.Printf("[gateway] failed to track usage: %v\n", err)
	}

	return &GenerateResult{
		Content:       content,
		Usage:         usage,
		ModelUsed:     routeResult.Model.ModelID,
		ContextTokens: promptTokens,
		UsedSettings:  ctxResult.UsedSettings,
	}, nil
}

// GenerateStream 流式生成
func (g *DefaultGateway) GenerateStream(ctx context.Context, req GenerateRequest) (<-chan StreamChunk, error) {
	// 配额检查已关闭（测试阶段不限制）
	_ = req.UserTier

	// 1. 构建上下文
	ctxReq := BuildContextRequest{
		ProjectID:  req.ProjectID,
		DocumentID: req.DocumentID,
		UserID:     req.UserID,
		UserPrompt: req.UserPrompt,
		AIMode:     req.AIMode,
		SceneText:  req.SceneText,
	}
	if len(req.Context) > 0 {
		ctxReq.MemoryBlocks = req.Context
	}
	if req.IncludeSettings && g.settingLoader != nil && req.ProjectID != "" {
		if pid, err := parseProjectID(req.ProjectID); err == nil && pid > 0 {
			if docs, err := g.settingLoader(pid); err == nil {
				ctxReq.SettingDocs = docs
			}
		}
	}

	ctxResult, err := g.contextBuilder.Build(ctx, ctxReq)
	if err != nil {
		return nil, fmt.Errorf("context build failed: %w", err)
	}

	// 3. 路由模型
	routeReq := RouteRequest{
		AIMode:        req.AIMode,
		ContextTokens: ctxResult.TotalTokens,
		RequireJSON:   req.AIMode == "outline" || req.AIMode == "review",
		UserTier:      "free",
	}
	routeResult := g.modelRouter.Route(routeReq)

	// 4. 获取并发许可
	if err := g.acquireSlot(ctx, req.ProjectID, routeResult.Model); err != nil {
		return nil, err
	}

	// 5. 调用 AI 流式接口
	start := time.Now()
	aiMessages := make([]ai.Message, len(ctxResult.Messages))
	for i, m := range ctxResult.Messages {
		aiMessages[i] = ai.Message{Role: m.Role, Content: m.Content}
	}
	aiCh, err := g.aiClient.Stream(ctx, routeResult.Model.ModelID, aiMessages)
	if err != nil {
		g.releaseSlot(req.ProjectID)
		return nil, fmt.Errorf("ai stream failed: %w", err)
	}

	// 6. 包装流，在结束时记录用量
	outCh := make(chan StreamChunk, 10)
	var contentBuilder string

	go func() {
		defer close(outCh)
		defer g.releaseSlot(req.ProjectID)

		// 首帧发送 UsedSettings 元数据（不带 Content）
		if len(ctxResult.UsedSettings) > 0 {
			outCh <- StreamChunk{UsedSettings: ctxResult.UsedSettings}
		}

		for msg := range aiCh {
			if msg.Error != "" {
				outCh <- StreamChunk{Error: msg.Error}
				return
			}
			contentBuilder += msg.Content
			outCh <- StreamChunk{Content: msg.Content}
		}

		// 流结束，记录用量
		duration := time.Since(start)
		promptTokens := ctxResult.TotalTokens
		completionTokens := len(contentBuilder) / 4
		totalTokens := promptTokens + completionTokens
		cost := float64(totalTokens) / 1000 * routeResult.Model.CostPer1K

		usage := UsageRecord{
			UserID:           req.UserID,
			ProjectID:        req.ProjectID,
			ModelID:          routeResult.Model.ModelID,
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
			Cost:             cost,
			AIMode:           req.AIMode,
			Timestamp:        time.Now(),
			DurationMs:       duration.Milliseconds(),
		}

		if err := g.usageTracker.Track(usage); err != nil {
			fmt.Printf("[gateway] failed to track usage: %v\n", err)
		}

		outCh <- StreamChunk{Done: true}
	}()

	return outCh, nil
}

// CheckQuota 检查配额
func (g *DefaultGateway) CheckQuota(userID string, tier string) (*QuotaStatus, error) {
	return g.usageTracker.CheckQuota(userID, tier)
}

// TrackUsage 记录用量（手动调用）
func (g *DefaultGateway) TrackUsage(record UsageRecord) error {
	return g.usageTracker.Track(record)
}

// SetDB 设置数据库（供 UsageTracker 持久化）
func (g *DefaultGateway) SetDB(db *gorm.DB) {
	g.usageTracker.db = db
}

// acquireSlot 获取并发槽位
func (g *DefaultGateway) acquireSlot(ctx context.Context, userID string, model ModelConfig) error {
	g.mu.Lock()
	sem, ok := g.semaphores[userID]
	if !ok {
		sem = make(chan struct{}, 2)
		g.semaphores[userID] = sem
	}
	g.mu.Unlock()

	select {
	case sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// releaseSlot 释放并发槽位
func (g *DefaultGateway) releaseSlot(userID string) {
	g.mu.RLock()
	sem, ok := g.semaphores[userID]
	g.mu.RUnlock()

	if ok {
		select {
		case <-sem:
		default:
		}
	}
}


// parseProjectID 尝试把字符串 project_id 解析成 uint
func parseProjectID(s string) (uint, error) {
	if s == "" || s == "default" {
		return 0, nil
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(n), nil
}

