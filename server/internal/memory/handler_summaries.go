// handler_summaries.go — 章节摘要 API

package memory

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ListSummaries 列出摘要
// GET /memory/summaries?project_id=xxx
func (h *Handler) ListSummaries(c *gin.Context) {
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
	ids, err := h.storage.ListChapterSummaries(userID, pid)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ids})
}

// GetSummary 读摘要
// GET /memory/summaries/:chapter_id?project_id=xxx
func (h *Handler) GetSummary(c *gin.Context) {
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
	chapterID := c.Param("chapter_id")
	if chapterID == "" {
		errorJSON(c, http.StatusBadRequest, "chapter_id required")
		return
	}
	content, err := h.storage.ReadChapterSummary(userID, pid, chapterID)
	if err != nil {
		errorJSON(c, http.StatusNotFound, "summary not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"chapter_id": chapterID, "content": content}})
}

// WriteSummary 写摘要
// PUT /memory/summaries/:chapter_id?project_id=xxx
func (h *Handler) WriteSummary(c *gin.Context) {
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
	chapterID := c.Param("chapter_id")
	if chapterID == "" {
		errorJSON(c, http.StatusBadRequest, "chapter_id required")
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errorJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.storage.WriteChapterSummary(userID, pid, chapterID, req.Content); err != nil {
		errorJSON(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}
