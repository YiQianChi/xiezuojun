// APIHandler — HTTP API 处理器

package gateway

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GenerateHTTPRequest HTTP API 请求体
type GenerateHTTPRequest struct {
	ProjectID  string `json:"project_id"`
	DocumentID string `json:"document_id"`
	Mode       string `json:"mode" binding:"required"`
	Content    string `json:"content" binding:"required"`
	Stream     bool   `json:"stream"`
}

// HandleGenerate AI 生成接口
func (g *DefaultGateway) HandleGenerate(c *gin.Context) {
	var req GenerateHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取 user_id（字符串形式）
	userIDStr := ""
	if val, exists := c.Get("userID"); exists {
		switch v := val.(type) {
		case uint:
			userIDStr = strconv.FormatUint(uint64(v), 10)
		case string:
			userIDStr = v
		default:
			userIDStr = fmt.Sprintf("%v", v)
		}
	}

	// 项目 ID 为空时给默认值
	if req.ProjectID == "" {
		req.ProjectID = "default"
	}

	// 构建 AI 生成请求
	genReq := GenerateRequest{
		ProjectID:  req.ProjectID,
		DocumentID: req.DocumentID,
		UserID:     userIDStr,
		AIMode:     req.Mode,
		UserPrompt: req.Content,
	}

	// 从 DB 读取文档内容作为 SceneText（通过 docContentLoader）
	if req.DocumentID != "" && g.docContentLoader != nil {
		docID, err := strconv.ParseUint(req.DocumentID, 10, 64)
		if err == nil {
			if content, err := g.docContentLoader(docID); err == nil && content != "" {
				genReq.SceneText = content
			}
		}
	}

	if !req.Stream {
		// 非流式：直接返回完整结果
		result, err := g.Generate(c.Request.Context(), genReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"content": result.Content,
			"model":   result.ModelUsed,
			"usage": map[string]interface{}{
				"total_tokens": result.Usage.TotalTokens,
				"cost":         result.Usage.Cost,
			},
		})
		return
	}

	// 流式：SSE 推送
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ch, err := g.GenerateStream(c.Request.Context(), genReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			return false
		case chunk, ok := <-ch:
			if !ok {
				return false
			}
			if chunk.Error != "" {
				c.SSEvent("error", gin.H{"error": chunk.Error})
				return false
			}
			if chunk.Done {
				c.SSEvent("done", "")
				return false
			}
			// 只发送有实际内容的 chunk（跳过思维链）
			if chunk.Content == "" {
				return true
			}
			c.SSEvent("content", gin.H{"content": chunk.Content})
			return true
		}
	})
}

// HandleGetUsage 获取用量信息
func (g *DefaultGateway) HandleGetUsage(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "anonymous"
	}
	stats := g.usageTracker.GetUsageStats(userID)

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"usage":   stats,
	})
}

// HandleGetModels 获取可用模型列表
func (g *DefaultGateway) HandleGetModels(c *gin.Context) {
	models := g.modelRouter.GetModels()
	c.JSON(http.StatusOK, gin.H{
		"models": models,
	})
}
