// Handler — 记忆系统 API 处理器
// 所有接口需要认证（JWT 中间件提取 userID）

package memory

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"xiezuojun-server/internal/ai"
)

// Handler 记忆 API Handler
type Handler struct {
	storage  *MemoryStorage
	db       *gorm.DB
	aiClient *ai.Client
}

// NewHandler 创建 Handler
func NewHandler(db *gorm.DB, docRoot string) *Handler {
	storage := NewMemoryStorage(docRoot)
	return &Handler{storage: storage, db: db}
}

// Storage 返回底层存储实例（供 gateway 等外部使用）
func (h *Handler) Storage() *MemoryStorage {
	return h.storage
}

// SetAIClient 设置 AI 客户端（用于提取和蒸馏）
func (h *Handler) SetAIClient(client *ai.Client) {
	h.aiClient = client
}

// getUserID 从 context 获取用户 ID
func getUserID(c *gin.Context) (string, error) {
	uid := c.GetString("user_id")
	if uid != "" {
		return uid, nil
	}
	val, exists := c.Get("userID")
	if !exists {
		return "", fmt.Errorf("unauthorized")
	}
	return fmt.Sprintf("%v", val), nil
}

// getProjectID 从 URL 参数或 query 获取项目 ID
func getProjectID(c *gin.Context, param string) (string, error) {
	pid := c.Param(param)
	if pid != "" {
		return pid, nil
	}
	pid = c.Query("project_id")
	if pid == "" {
		return "", fmt.Errorf("project_id required")
	}
	return pid, nil
}

// errorJSON 返回统一错误响应
func errorJSON(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"error": msg})
}
