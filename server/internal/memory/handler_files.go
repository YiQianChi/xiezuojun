// handler_files.go — 文件管理 API

package memory

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ListFiles 列出所有记忆文件
// GET /memory/files?project_id=xxx
func (h *Handler) ListFiles(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		errorJSON(c, http.StatusUnauthorized, err.Error())
		return
	}
	pid, err := getProjectID(c, "")
	if err != nil {
		errorJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	files, err := h.storage.ListMemoryFiles(userID, pid)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": files})
}

// GetFile 读取记忆文件
// GET /memory/file?project_id=xxx&path=xxx
func (h *Handler) GetFile(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		errorJSON(c, http.StatusUnauthorized, err.Error())
		return
	}
	pid, err := getProjectID(c, "")
	if err != nil {
		errorJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	path := c.Query("path")
	if path == "" {
		errorJSON(c, http.StatusBadRequest, "path required")
		return
	}
	content, err := h.storage.ReadMemoryFile(userID, pid, path)
	if err != nil {
		errorJSON(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"path": path, "content": content}})
}

// WriteFile 写入记忆文件
// PUT /memory/file?project_id=xxx&path=xxx
func (h *Handler) WriteFile(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		errorJSON(c, http.StatusUnauthorized, err.Error())
		return
	}
	pid, err := getProjectID(c, "")
	if err != nil {
		errorJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	path := c.Query("path")
	if path == "" {
		errorJSON(c, http.StatusBadRequest, "path required")
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errorJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.storage.WriteMemoryFile(userID, pid, path, req.Content); err != nil {
		errorJSON(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}

// DeleteFile 删除记忆文件
// DELETE /memory/file?project_id=xxx&path=xxx
func (h *Handler) DeleteFile(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		errorJSON(c, http.StatusUnauthorized, err.Error())
		return
	}
	pid, err := getProjectID(c, "")
	if err != nil {
		errorJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	path := c.Query("path")
	if path == "" {
		errorJSON(c, http.StatusBadRequest, "path required")
		return
	}
	if err := h.storage.DeleteMemoryFile(userID, pid, path); err != nil {
		errorJSON(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
