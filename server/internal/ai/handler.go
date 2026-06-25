package ai

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler AI 相关 API
type Handler struct {
	db     *gorm.DB
	client *Client
}

func NewHandler(db *gorm.DB, client *Client) *Handler {
	return &Handler{db: db, client: client}
}

// Request AI 流式生成请求
type Request struct {
	Messages  []Message `json:"messages" binding:"required"`
	Model     string    `json:"model"`     // optional, e.g. "glm-5-turbo"
	ProjectID uint      `json:"project_id"` // for context
}

type Message struct {
	Role    string `json:"role"`    // user | assistant | system
	Content string `json:"content"`
}

func (h *Handler) ChatStream(c *gin.Context) {
	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	model := req.Model
	if model == "" {
		model = "glm-5-turbo"
	}

	// 设置 SSE 头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ch, err := h.client.Stream(c.Request.Context(), model, req.Messages)
	if err != nil {
		c.SSEvent("error", gin.H{"error": err.Error()})
		return
	}

	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			return false
		case msg, ok := <-ch:
			if !ok {
				c.SSEvent("done", "")
				return false
			}
			if msg.Error != "" {
				c.SSEvent("error", gin.H{"error": msg.Error})
				return false
			}
			c.SSEvent("message", gin.H{"content": msg.Content})
			return true
		}
	})
}

// Outlines 生成大纲
type OutlineRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

func (h *Handler) GenerateOutline(c *gin.Context) {
	var req OutlineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prompt := buildOutlinePrompt(req.Title, req.Description)
	content, err := h.client.Complete(c.Request.Context(), "glm-5-turbo", []Message{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": content})
}

// Plot 续写/润色
type PlotRequest struct {
	Context  string `json:"context" binding:"required"` // 前文
	Command  string `json:"command" binding:"required"` // continue | polish | expand | shorten
	Style    string `json:"style"`    // 风格提示
}

func (h *Handler) PlotContinue(c *gin.Context) {
	var req PlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prompt := buildPlotPrompt(req.Context, req.Command, req.Style)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ch, err := h.client.Stream(c.Request.Context(), "glm-5-turbo", []Message{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		c.SSEvent("error", gin.H{"error": err.Error()})
		return
	}

	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			return false
		case msg, ok := <-ch:
			if !ok {
				c.SSEvent("done", "")
				return false
			}
			if msg.Error != "" {
				c.SSEvent("error", gin.H{"error": msg.Error})
				return false
			}
			c.SSEvent("message", gin.H{"content": msg.Content})
			return true
		}
	})
}

func buildOutlinePrompt(title, desc string) string {
	return `请为小说《` + title + `》生成详细的大纲。

` + desc + `

要求：
1. 按"卷→章"结构输出
2. 每卷包含：卷标题、卷简介、5-10章
3. 每章包含：章标题、章概要（100-200字）
4. 输出格式为 JSON：{"volumes":[{"title":"","summary":"","chapters":[{"title":"","summary":""}]]}
5. 只输出 JSON，不要有其他内容`
}

func buildPlotPrompt(context, command, style string) string {
	cmdMap := map[string]string{
		"continue": "请根据前文续写后续内容，保持情节连贯、文风一致",
		"polish":   "请润色以下内容，提升文笔质量，保持原意不变",
		"expand":   "请扩写以下内容，增加细节描写和环境渲染",
		"shorten":  "请精简以下内容，保留核心情节，去除冗余描述",
	}
	cmdDesc := cmdMap[command]
	if cmdDesc == "" {
		cmdDesc = "请续写以下内容"
	}

	styleHint := ""
	if style != "" {
		styleHint = "\n风格要求：" + style
	}

	return cmdDesc + styleHint + "\n\n前文：\n" + context + "\n\n请直接输出内容，不要添加解释。"
}
