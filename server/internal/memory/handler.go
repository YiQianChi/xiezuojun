// Handler — 记忆系统 API 处理器
// 所有接口需要认证（JWT 中间件提取 user_id）

package memory

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler 记忆 API Handler
type Handler struct {
	storage *MemoryStorage
	db      *gorm.DB
}

// NewHandler 创建 Handler
func NewHandler(storage *MemoryStorage, db *gorm.DB) *Handler {
	return &Handler{storage: storage, db: db}
}

// uid 从 JWT context 获取用户 ID（字符串）
func uid(c *gin.Context) (string, error) {
	val, exists := c.Get("userID")
	if !exists {
		return "", fmt.Errorf("unauthorized")
	}
	return strconv.FormatUint(uint64(val.(uint)), 10), nil
}

// ============== 记忆文件 CRUD ==============

// ListFiles 列出项目记忆文件树
// GET /api/memory/files?project_id=xxx
func (h *Handler) ListFiles(c *gin.Context) {
	userID, err := uid(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	pid := c.Query("project_id")
	if pid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id required"})
		return
	}

	files, err := h.storage.ListMemoryFiles(userID, pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": files})
}

// GetFile 读取记忆文件
// GET /api/memory/file?project_id=xxx&path=WORLD.md
func (h *Handler) GetFile(c *gin.Context) {
	userID, err := uid(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	pid := c.Query("project_id")
	path := c.Query("path")
	if pid == "" || path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id and path required"})
		return
	}

	content, err := h.storage.ReadMemoryFile(userID, pid, path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("file not found: %s", path)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": content})
}

// WriteFile 写入记忆文件
// PUT /api/memory/file  body: {project_id, path, content}
func (h *Handler) WriteFile(c *gin.Context) {
	userID, err := uid(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req struct {
		ProjectID string `json:"project_id" binding:"required"`
		Path      string `json:"path" binding:"required"`
		Content   string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.storage.WriteMemoryFile(userID, req.ProjectID, req.Path, req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}

// DeleteFile 删除记忆文件
// DELETE /api/memory/file?project_id=xxx&path=xxx
func (h *Handler) DeleteFile(c *gin.Context) {
	userID, err := uid(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	pid := c.Query("project_id")
	path := c.Query("path")
	if pid == "" || path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id and path required"})
		return
	}

	if err := h.storage.DeleteMemoryFile(userID, pid, path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ============== 角色卡 CRUD ==============

// ListCharacters 列出角色
// GET /api/memory/characters?project_id=xxx
func (h *Handler) ListCharacters(c *gin.Context) {
	userID, err := uid(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	pid := c.Query("project_id")
	if pid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id required"})
		return
	}

	names, err := h.storage.ListCharacters(userID, pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": names})
}

// GetCharacter 读取角色卡
// GET /api/memory/characters/:name?project_id=xxx
func (h *Handler) GetCharacter(c *gin.Context) {
	userID, err := uid(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	pid := c.Query("project_id")
	name := c.Param("name")
	if pid == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id and name required"})
		return
	}

	content, err := h.storage.ReadCharacter(userID, pid, name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("character not found: %s", name)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": content})
}

// WriteCharacter 写角色卡
// PUT /api/memory/characters/:name  body: {project_id, content}
func (h *Handler) WriteCharacter(c *gin.Context) {
	userID, err := uid(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	name := c.Param("name")

	var req struct {
		ProjectID string `json:"project_id" binding:"required"`
		Content   string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.storage.WriteCharacter(userID, req.ProjectID, name, req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}

// DeleteCharacter 删角色
// DELETE /api/memory/characters/:name?project_id=xxx
func (h *Handler) DeleteCharacter(c *gin.Context) {
	userID, err := uid(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	pid := c.Query("project_id")
	name := c.Param("name")
	if pid == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id and name required"})
		return
	}

	if err := h.storage.DeleteCharacter(userID, pid, name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ============== 章节摘要 ==============

// ListSummaries 列章节摘要
// GET /api/memory/summaries?project_id=xxx
func (h *Handler) ListSummaries(c *gin.Context) {
	userID, err := uid(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	pid := c.Query("project_id")
	if pid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id required"})
		return
	}

	ids, err := h.storage.ListChapterSummaries(userID, pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ids})
}

// GetSummary 读章节摘要
// GET /api/memory/summaries/:chapter_id?project_id=xxx
func (h *Handler) GetSummary(c *gin.Context) {
	userID, err := uid(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	pid := c.Query("project_id")
	chapterID := c.Param("chapter_id")
	if pid == "" || chapterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id and chapter_id required"})
		return
	}

	content, err := h.storage.ReadChapterSummary(userID, pid, chapterID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("summary not found: %s", chapterID)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": content})
}

// WriteSummary 写章节摘要
// PUT /api/memory/summaries/:chapter_id  body: {project_id, content}
func (h *Handler) WriteSummary(c *gin.Context) {
	userID, err := uid(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	chapterID := c.Param("chapter_id")

	var req struct {
		ProjectID string `json:"project_id" binding:"required"`
		Content   string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.storage.WriteChapterSummary(userID, req.ProjectID, chapterID, req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}
