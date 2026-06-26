// handler_layers.go — 三层记忆 API

package memory

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetProjectMemory 读取项目所有三层记忆
// GET /api/memory/:project_id
func (h *Handler) GetProjectMemory(c *gin.Context) {
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

	working, _ := h.storage.ReadWorkingMemory(userID, pid)
	longTerm, _ := h.storage.ReadLongTermMemory(userID, pid)
	style, _ := h.storage.ReadStyleMemory(userID, pid)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"working_memory": working,
			"long_term":      longTerm,
			"style":           style,
		},
	})
}

// UpdateMemoryFile 手动编辑记忆文件
// PUT /api/memory/:project_id/:file
func (h *Handler) UpdateMemoryFile(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		errorJSON(c, http.StatusUnauthorized, err.Error())
		return
	}
	pid := c.Param("project_id")
	file := c.Param("file")
	if pid == "" || file == "" {
		errorJSON(c, http.StatusBadRequest, "project_id and file required")
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errorJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.storage.WriteMemoryFile(userID, pid, file, req.Content); err != nil {
		errorJSON(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}

// DistillMemory 蒸馏短期→长期
// POST /api/memory/:project_id/distill
func (h *Handler) DistillMemory(c *gin.Context) {
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
	if err := h.storage.DistillMemory(userID, pid); err != nil {
		errorJSON(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "distilled"})
}
