// handler_extract.go — 记忆提取 API

package memory

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"xiezuojun-server/internal/ai"
)

// ExtractMemory 触发从文档内容提取记忆
// POST /api/memory/:project_id/extract
func (h *Handler) ExtractMemory(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		errorJSON(c, http.StatusUnauthorized, err.Error())
		return
	}
	pid, err := getProjectID(c, "project_id")
	if err != nil {
		errorJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errorJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	if h.aiClient == nil {
		errorJSON(c, http.StatusServiceUnavailable, "AI client not configured")
		return
	}

	charNames := DetectCharacterNames(req.Content)
	extracted := h.extractWithAI(c.Request.Context(), req.Content)

	content := buildExtractContent(charNames, extracted)
	existing, _ := h.storage.ReadWorkingMemory(userID, pid)
	_ = h.storage.WriteWorkingMemory(userID, pid, existing+"\n\n"+content)

	// 更新风格记忆
	if extracted != nil && len(extracted.StyleFeatures) > 0 {
		_ = h.storage.WriteStyleMemory(userID, pid, buildStyleContent(extracted.StyleFeatures))
	}

	// 异步创建角色卡
	go h.autoCreateCharCards(userID, pid, extracted)

	c.JSON(http.StatusOK, gin.H{
		"message":        "extracted",
		"characters":     charNames,
		"extracted_info": extracted,
	})
}

// buildExtractContent 构建提取内容文本
func buildExtractContent(charNames []string, extracted *ExtractResult) string {
	var sb strings.Builder
	sb.WriteString(time.Now().Format("## 提取于 2006-01-02 15:04\n\n"))

	if len(charNames) > 0 {
		sb.WriteString("### 检测到的角色\n")
		for _, name := range charNames {
			sb.WriteString("- " + name + "\n")
		}
		sb.WriteString("\n")
	}
	if extracted != nil {
		appendExtractedInfo(&sb, extracted)
	}
	return sb.String()
}

// appendExtractedInfo 追加提取到的结构化信息
func appendExtractedInfo(sb *strings.Builder, e *ExtractResult) {
	if len(e.Characters) > 0 {
		sb.WriteString("### 角色信息\n")
		for _, ch := range e.Characters {
			sb.WriteString("- **" + ch.Name + "**（" + ch.Role + "）: " + ch.Description + "\n")
		}
		sb.WriteString("\n")
	}
	if len(e.Events) > 0 {
		sb.WriteString("### 情节要点\n")
		for _, evt := range e.Events {
			sb.WriteString("- " + evt + "\n")
		}
		sb.WriteString("\n")
	}
	if len(e.Locations) > 0 {
		sb.WriteString("### 地点\n")
		for _, loc := range e.Locations {
			sb.WriteString("- " + loc + "\n")
		}
		sb.WriteString("\n")
	}
	if len(e.StyleFeatures) > 0 {
		sb.WriteString("### 风格特征\n")
		for _, feat := range e.StyleFeatures {
			sb.WriteString("- " + feat + "\n")
		}
	}
}

// buildStyleContent 构建风格记忆内容
func buildStyleContent(features []string) string {
	var sb strings.Builder
	sb.WriteString("# 写作风格画像\n\n最后更新: ")
	sb.WriteString(time.Now().Format("2006-01-02") + "\n\n")
	for _, feat := range features {
		sb.WriteString("- " + feat + "\n")
	}
	return sb.String()
}

// autoCreateCharCards 异步创建角色卡
func (h *Handler) autoCreateCharCards(uid, pid string, extracted *ExtractResult) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[memory] extract character cards panic: %v", r)
		}
	}()
	if extracted == nil {
		return
	}
	for _, ch := range extracted.Characters {
		if ch.Name == "" {
			continue
		}
		existing, err := h.storage.ReadCharacter(uid, pid, ch.Name)
		if err == nil && existing != "" {
			continue
		}
		card := "# " + ch.Name + "\n\n> auto_generated: true\n\n## 基本信息\n- 身份：" + ch.Role + "\n\n## 描述\n" + ch.Description + "\n"
		_ = h.storage.WriteCharacter(uid, pid, ch.Name, card)
		log.Printf("[memory] auto-created character card: %s", ch.Name)
	}
}

// ExtractResult AI 提取结果
type ExtractResult struct {
	Characters    []CharacterInfo `json:"characters"`
	Events        []string        `json:"events"`
	Locations     []string        `json:"locations"`
	StyleFeatures []string        `json:"style_features"`
}

// CharacterInfo 角色信息
type CharacterInfo struct {
	Name        string `json:"name"`
	Role        string `json:"role"`
	Description string `json:"description"`
}

// extractWithAI 使用 AI 提取结构化信息
func (h *Handler) extractWithAI(ctx context.Context, content string) *ExtractResult {
	if h.aiClient == nil {
		return nil
	}

	prompt := `分析以下文本，提取以下信息，以JSON格式返回：
{
  "characters": [{"name": "角色名", "role": "主角/配角/路人", "description": "简短描述"}],
  "events": ["情节事件1", "情节事件2"],
  "locations": ["地点1", "地点2"],
  "style_features": ["风格特征1", "风格特征2"]
}
只返回JSON，不要其他内容。如果没有某类信息，返回空数组。
分析风格时注意：句式特点、叙事视角、用词偏好、修辞手法、节奏感。`

	if len(content) > 3000 {
		content = content[:3000]
	}

	result, err := h.aiClient.Complete(ctx, "glm-4-flash", []ai.Message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: content},
	})
	if err != nil {
		log.Printf("[memory] AI extract failed: %v", err)
		return nil
	}

	jsonStr := extractJSON(result)
	var extracted ExtractResult
	if err := json.Unmarshal([]byte(jsonStr), &extracted); err != nil {
		log.Printf("[memory] parse extract JSON failed: %v, raw: %s", err, jsonStr[:min(200, len(jsonStr))])
		return nil
	}
	return &extracted
}

// extractJSON 从 AI 回复中提取 JSON
func extractJSON(s string) string {
	if idx := strings.Index(s, "```json"); idx >= 0 {
		s = s[idx+7:]
		if end := strings.Index(s, "```"); end >= 0 {
			s = s[:end]
		}
	} else if idx := strings.Index(s, "```"); idx >= 0 {
		s = s[idx+3:]
		if end := strings.Index(s, "```"); end >= 0 {
			s = s[:end]
		}
	}
	return strings.TrimSpace(s)
}
