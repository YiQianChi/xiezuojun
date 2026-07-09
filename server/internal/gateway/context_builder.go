// ContextBuilder — 上下文金字塔构建器
// 核心能力：将多层上下文智能组装，适配模型窗口

package gateway

import (
	"context"
	"fmt"
	"log"
	"strings"

	"xiezuojun-server/internal/memory"
)

// MemoryReader 记忆读取接口（由 storage 实现）
type MemoryReader interface {
	ReadCharacter(uid, pid, name string) (string, error)
	ReadMemoryFile(uid, pid, path string) (string, error)
}

// ContextBuilder 上下文构建器
type ContextBuilder struct {
	// 各层级最大 token 分配（可配置）
	MaxTokensPerLevel map[ContextLevel]int
	// 系统提示模板
	SystemTemplate string
	// 记忆读取器（可选，nil 时跳过记忆读取）
	memoryReader MemoryReader
}

// NewContextBuilder 创建构建器
func NewContextBuilder(reader MemoryReader) *ContextBuilder {
	return &ContextBuilder{
		MaxTokensPerLevel: map[ContextLevel]int{
			LevelMemory:  2000, // 作者记忆层最多 2000 tokens
			LevelProject: 4000, // 项目设定最多 4000 tokens（含大纲/人物/世界观）
			LevelScene:   6000, // 前文最多 6000 tokens
			LevelTask:    1000, // 任务指令最多 1000 tokens
		},
		SystemTemplate: `你是「写作菌」，一位专业的小说创作助手。

当前任务模式：{{MODE}}

创作原则：
1. 保持角色行为一致性，不 OOC
2. 注意伏笔呼应，前后文逻辑自洽
3. 文风与原文保持一致
4. 情节推进自然，避免突兀转折
5. 对话符合角色性格设定

{{STYLE_HINT}}`,
		memoryReader: reader,
	}
}

// Build 构建完整上下文
func (b *ContextBuilder) Build(ctx context.Context, req BuildContextRequest) (*BuildContextResult, error) {
	// 1. 获取模式模板
	modeTemplate, ok := ModeTemplates[req.AIMode]
	if !ok {
		modeTemplate = ModeTemplates["continue"] // 默认续写
	}

	// 2. 组装系统提示
	systemPrompt := b.buildSystemPrompt(modeTemplate, req)

	// 3. 收集并截断各层级上下文
	blocks := b.collectBlocks(req)
	truncated := b.truncateBlocks(blocks, modeTemplate)

	// 4. 构建消息列表
	// 注入优先级：PREFERENCES.md（强制）> 前文 > 角色卡 > STYLE.md > WORLD.md
	messages := []Message{
		{Role: "system", Content: systemPrompt},
	}

	// 按优先级排序：Memory > Project > Scene > Task
	ordered := []ContextLevel{LevelMemory, LevelProject, LevelScene, LevelTask}
	for _, level := range ordered {
		for _, block := range truncated {
			if block.Level == level && block.Content != "" {
				messages = append(messages, Message{
					Role:    "user",
					Content: fmt.Sprintf("[%s]\n%s", block.Source, block.Content),
				})
			}
		}
	}

	// 最后添加用户当前任务
	if req.UserPrompt != "" {
		messages = append(messages, Message{
			Role:    "user",
			Content: req.UserPrompt,
		})
	}

	// 5. 计算总 token（粗略估算）
	totalTokens := 0
	for _, m := range messages {
		totalTokens += estimateTokens(m.Content)
	}

	// 6. 生成 UsedSettings 列表（供前端展示）
	usedSettings := b.buildUsedSettings(req.SettingDocs, truncated)

	return &BuildContextResult{
		SystemPrompt: systemPrompt,
		Messages:     messages,
		ContextMap:   truncated,
		TotalTokens:  totalTokens,
		UsedSettings: usedSettings,
	}, nil
}

// buildUsedSettings 根据 truncatedBlocks 判断每张设定卡是否真的用了
func (b *ContextBuilder) buildUsedSettings(docs []SettingDoc, truncated []ContextBlock) []UsedSetting {
	if len(docs) == 0 {
		return nil
	}
	// 建立 source → tokens 索引
	usedMap := make(map[string]int)
	for _, blk := range truncated {
		if blk.Level != LevelProject {
			continue
		}
		usedMap[blk.Source] = estimateTokens(blk.Content)
	}
	result := make([]UsedSetting, 0, len(docs))
	for _, d := range docs {
		source := settingSourceKey(d)
		tokens, ok := usedMap[source]
		result = append(result, UsedSetting{
			ID:     d.ID,
			Title:  d.Title,
			Type:   d.Type,
			Used:   ok,
			Tokens: tokens,
		})
	}
	return result
}

// settingSourceKey 统一设定文档的 Source 标识
func settingSourceKey(d SettingDoc) string {
	typeLabel := map[string]string{
		"outline":         "大纲",
		"characters":      "人物",
		"worldbuilding":   "世界观",
		"storyline":       "故事主线",
		"chapter_outline": "章节大纲",
	}[d.Type]
	if typeLabel == "" {
		typeLabel = d.Type
	}
	return fmt.Sprintf("设定·%s·%s", typeLabel, d.Title)
}

// buildSystemPrompt 构建系统提示
func (b *ContextBuilder) buildSystemPrompt(mode ModeTemplate, req BuildContextRequest) string {
	prompt := mode.SystemPrompt

	// 替换风格提示
	styleHint := ""
	if mode.StyleHints != nil && len(mode.StyleHints) > 0 {
		styleHint = fmt.Sprintf("\n风格要求：%s", strings.Join(mode.StyleHints, "，"))
	}
	prompt = strings.ReplaceAll(prompt, "{{STYLE_HINT}}", styleHint)
	prompt = strings.ReplaceAll(prompt, "{{MODE}}", mode.Name)

	return prompt
}

// collectBlocks 收集所有上下文块
func (b *ContextBuilder) collectBlocks(req BuildContextRequest) []ContextBlock {
	blocks := make([]ContextBlock, 0)

	// ===== 记忆层（优先级最高） =====

	// PREFERENCES.md — 强制注入
	if b.memoryReader != nil && req.UserID != "" && req.ProjectID != "" {
		if prefs, err := b.memoryReader.ReadMemoryFile(req.UserID, req.ProjectID, "PREFERENCES.md"); err == nil && prefs != "" {
			blocks = append(blocks, ContextBlock{
				Level:    LevelMemory,
				Content:  prefs,
				Priority: 1.0, // 强制最高优先级
				Source:   "PREFERENCES.md",
			})
		}

		// STYLE.md
		if style, err := b.memoryReader.ReadMemoryFile(req.UserID, req.ProjectID, "STYLE.md"); err == nil && style != "" {
			blocks = append(blocks, ContextBlock{
				Level:    LevelMemory,
				Content:  style,
				Priority: 0.85,
				Source:   "STYLE.md",
			})
		}

		// WORLD.md
		if world, err := b.memoryReader.ReadMemoryFile(req.UserID, req.ProjectID, "WORLD.md"); err == nil && world != "" {
			blocks = append(blocks, ContextBlock{
				Level:    LevelMemory,
				Content:  world,
				Priority: 0.8,
				Source:   "WORLD.md",
			})
		}

		// 角色卡 — 根据前文内容自动识别人名
		if req.SceneText != "" {
			charNames := memory.DetectCharacterNames(req.SceneText)
			for _, name := range charNames {
				if card, err := b.memoryReader.ReadCharacter(req.UserID, req.ProjectID, name); err == nil && card != "" {
					blocks = append(blocks, ContextBlock{
						Level:    LevelMemory,
						Content:  card,
						Priority: 0.9,
						Source:   "角色卡:" + name,
					})
				}
			}
			if len(charNames) > 0 {
				log.Printf("[context] detected %d character names: %v", len(charNames), charNames)
			}
		}
	}

	// 作者记忆层（从请求传入）
	for _, mem := range req.MemoryBlocks {
		blocks = append(blocks, mem)
	}

	// 项目层
	if req.ProjectMeta != "" {
		blocks = append(blocks, ContextBlock{
			Level:    LevelProject,
			Content:  req.ProjectMeta,
			Priority: 0.8,
			Source:   "project-meta",
		})
	}

	// 项目层：设定文档（大纲/人物/世界观）
	// 优先级：大纲 > 人物 > 世界观（大纲直接决定情节，人物次之，世界观兜底）
	if len(req.SettingDocs) > 0 {
		typePriority := map[string]float64{
			"outline":         0.95,
			"storyline":       0.93,
			"chapter_outline": 0.91,
			"characters":      0.90,
			"worldbuilding":   0.80,
		}
		for _, doc := range req.SettingDocs {
			content := strings.TrimSpace(doc.Content)
			if content == "" {
				continue
			}
			p, ok := typePriority[doc.Type]
			if !ok {
				p = 0.75
			}
			blocks = append(blocks, ContextBlock{
				Level:    LevelProject,
				Content:  content,
				Priority: p,
				Source:   settingSourceKey(doc),
			})
		}
	}

	// 场景层（前文）
	if req.SceneText != "" {
		blocks = append(blocks, ContextBlock{
			Level:    LevelScene,
			Content:  req.SceneText,
			Priority: 0.9,
			Source:   "preceding-text",
		})
	}

	// 任务层
	if req.UserPrompt != "" {
		blocks = append(blocks, ContextBlock{
			Level:    LevelTask,
			Content:  req.UserPrompt,
			Priority: 1.0, // 任务层优先级最高
			Source:   "user-task",
		})
	}

	return blocks
}

// truncateBlocks 截断上下文块以适应窗口
func (b *ContextBuilder) truncateBlocks(blocks []ContextBlock, mode ModeTemplate) []ContextBlock {
	result := make([]ContextBlock, 0, len(blocks))

	// 计算各层级已用 token
	usedTokens := make(map[ContextLevel]int)

	for _, block := range blocks {
		maxForLevel := b.MaxTokensPerLevel[block.Level]
		if maxForLevel == 0 {
			maxForLevel = 2000 // 默认
		}

		// 估算当前块 token
		blockTokens := estimateTokens(block.Content)

		// 如果该层级已超限，截断或跳过
		if usedTokens[block.Level]+blockTokens > maxForLevel {
			remaining := maxForLevel - usedTokens[block.Level]
			if remaining <= 0 {
				continue // 该层级已满，跳过
			}
			// 截断内容
			block.Content = truncateToTokens(block.Content, remaining)
			blockTokens = remaining
		}

		usedTokens[block.Level] += blockTokens
		result = append(result, block)
	}

	return result
}

// estimateTokens 粗略估算 token 数（中文：1 token ≈ 1-1.5 字，英文：1 token ≈ 4 字符）
func estimateTokens(text string) int {
	// 简单估算：中文字符按 1:1，英文按 1:4
	chineseCount := 0
	otherCount := 0
	for _, r := range text {
		if r >= '\u4e00' && r <= '\u9fff' {
			chineseCount++
		} else {
			otherCount++
		}
	}
	return chineseCount + otherCount/4 + 1
}

// truncateToTokens 按估算 token 截断文本
func truncateToTokens(text string, maxTokens int) string {
	if estimateTokens(text) <= maxTokens {
		return text
	}

	// 从末尾截断，保留开头
	lines := strings.Split(text, "\n")
	var result strings.Builder
	tokens := 0

	// 从后往前取
	for i := len(lines) - 1; i >= 0; i-- {
		lineTokens := estimateTokens(lines[i]) + 1 // +1 for newline
		if tokens+lineTokens > maxTokens {
			break
		}
		// prepend
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(lines[i])
		tokens += lineTokens
	}

	// 反转回来
	parts := strings.Split(result.String(), "\n")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}

	return strings.Join(parts, "\n")
}
