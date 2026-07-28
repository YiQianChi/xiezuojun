// AI Chat handler — multi-turn conversation with project context

package gateway

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"xiezuojun-server/internal/ai"
)

// ChatMessage 单条对话消息
type ChatMessage struct {
	Role    string `json:"role" binding:"required"` // user | assistant
	Content string `json:"content" binding:"required"`
}

// ChatHTTPRequest 对话接口请求体
type ChatHTTPRequest struct {
	ProjectID string        `json:"project_id"`
	Messages  []ChatMessage `json:"messages" binding:"required,min=1"`
	Stream    bool          `json:"stream"`
}

// HandleChat AI 对话接口 — 支持多轮对话 + SSE 流式输出
func (g *DefaultGateway) HandleChat(c *gin.Context) {
	var req ChatHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr := getUserIDStr(c)
	if req.ProjectID == "" {
		req.ProjectID = "default"
	}

	// 1. 加载项目上下文（设定文档 + 伏笔），复用 Generate 的逻辑
	ctxBlocks := g.loadChatContext(req.ProjectID)

	// 2. 构建系统提示
	systemPrompt := "你是一个专业的小说创作助手。根据以下小说设定回答问题或提供建议。"
	if len(ctxBlocks) > 0 {
		systemPrompt += "\n\n--- 小说设定 ---\n" + ctxBlocks
	}
	systemPrompt += "\n\n请基于以上设定，回答作者的提问或提供创作建议。保持专业、有建设性。"

	// 3. 组装 AI 消息
	aiMessages := make([]ai.Message, 0, len(req.Messages)+1)
	aiMessages = append(aiMessages, ai.Message{Role: "system", Content: systemPrompt})
	for _, m := range req.Messages {
		aiMessages = append(aiMessages, ai.Message{Role: m.Role, Content: m.Content})
	}

	// 4. 路由模型（复用 ModelRouter，用 chat 模式）
	routeReq := RouteRequest{
		AIMode:        "chat",
		ContextTokens: estimateTokens(systemPrompt) + sumMessageTokens(req.Messages),
		RequireJSON:   false,
		UserTier:      "free",
	}
	routeResult := g.modelRouter.Route(routeReq)

	if err := g.acquireSlot(c.Request.Context(), userIDStr, routeResult.Model); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !req.Stream {
		g.chatComplete(c, aiMessages, routeResult, userIDStr, req.ProjectID)
		return
	}

	g.chatStream(c, aiMessages, routeResult, userIDStr, req.ProjectID)
}

// chatComplete 非流式对话
func (g *DefaultGateway) chatComplete(c *gin.Context, messages []ai.Message, route RouteResult, userID, projectID string) {
	defer g.releaseSlot(userID)

	start := time.Now()
	content, err := g.aiClient.Complete(c.Request.Context(), route.Model.ModelID, messages)
	duration := time.Since(start)
	if err != nil {
		fmt.Printf("[gateway] Chat Complete error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	promptTokens := sumAIMessageTokens(messages)
	completionTokens := len(content) / 4
	totalTokens := promptTokens + completionTokens
	g.trackChatUsage(userID, projectID, route.Model.ModelID, promptTokens, completionTokens, totalTokens, route.Model.CostPer1K, duration)

	c.JSON(http.StatusOK, gin.H{
		"content": content,
		"role":    "assistant",
	})
}

// chatStream SSE 流式对话
func (g *DefaultGateway) chatStream(c *gin.Context, messages []ai.Message, route RouteResult, userID, projectID string) {
	start := time.Now()
	aiCh, err := g.aiClient.Stream(c.Request.Context(), route.Model.ModelID, messages)
	if err != nil {
		g.releaseSlot(userID)
		fmt.Printf("[gateway] Chat Stream error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	var contentBuilder strings.Builder
	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			return false
		case msg, ok := <-aiCh:
			if !ok {
				// 流结束，记录用量
				duration := time.Since(start)
				promptTokens := sumAIMessageTokens(messages)
				completionTokens := contentBuilder.Len() / 4
				totalTokens := promptTokens + completionTokens
				g.trackChatUsage(userID, projectID, route.Model.ModelID, promptTokens, completionTokens, totalTokens, route.Model.CostPer1K, duration)
				g.releaseSlot(userID)
				c.SSEvent("done", "")
				return false
			}
			if msg.Error != "" {
				g.releaseSlot(userID)
				c.SSEvent("error", gin.H{"error": msg.Error})
				return false
			}
			if msg.Content != "" {
				contentBuilder.WriteString(msg.Content)
				c.SSEvent("content", gin.H{"content": msg.Content})
			}
			return true
		}
	})
}

// loadChatContext 加载项目设定文档 + 未回收伏笔，拼成纯文本
func (g *DefaultGateway) loadChatContext(projectID string) string {
	var sb strings.Builder

	pid, err := parseProjectID(projectID)
	if err != nil || pid == 0 {
		return ""
	}

	// 设定文档
	if g.settingLoader != nil {
		if docs, err := g.settingLoader(pid); err == nil {
			for _, d := range docs {
				if d.Content == "" {
					continue
				}
				sb.WriteString(fmt.Sprintf("【%s】%s\n%s\n\n", settingTypeLabel(d.Type), d.Title, d.Content))
			}
		}
	}

	// 未回收伏笔
	if g.threadLoader != nil {
		if text, err := g.threadLoader(pid); err == nil && text != "" {
			sb.WriteString("【未回收伏笔】\n")
			sb.WriteString(text)
			sb.WriteString("\n\n")
		}
	}

	return sb.String()
}

// trackChatUsage 记录对话用量
func (g *DefaultGateway) trackChatUsage(userID, projectID, modelID string, prompt, completion, total int, costPer1K float64, duration time.Duration) {
	cost := float64(total) / 1000 * costPer1K
	usage := UsageRecord{
		UserID:           userID,
		ProjectID:        projectID,
		ModelID:          modelID,
		PromptTokens:     prompt,
		CompletionTokens: completion,
		TotalTokens:      total,
		Cost:             cost,
		AIMode:           "chat",
		Timestamp:        time.Now(),
		DurationMs:       duration.Milliseconds(),
	}
	if err := g.usageTracker.Track(usage); err != nil {
		fmt.Printf("[gateway] failed to track chat usage: %v\n", err)
	}
}

// getUserIDStr 从 gin.Context 提取 userID 字符串
func getUserIDStr(c *gin.Context) string {
	if val, exists := c.Get("userID"); exists {
		switch v := val.(type) {
		case uint:
			return strconv.FormatUint(uint64(v), 10)
		case string:
			return v
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

func sumMessageTokens(msgs []ChatMessage) int {
	total := 0
	for _, m := range msgs {
		total += estimateTokens(m.Content)
	}
	return total
}

func sumAIMessageTokens(msgs []ai.Message) int {
	total := 0
	for _, m := range msgs {
		total += estimateTokens(m.Content)
	}
	return total
}

func settingTypeLabel(t string) string {
	switch t {
	case "outline":
		return "大纲"
	case "characters":
		return "人物"
	case "worldbuilding":
		return "世界观"
	case "storyline":
		return "故事主线"
	case "chapter_outline":
		return "章节大纲"
	default:
		return t
	}
}
