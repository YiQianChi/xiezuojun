package project

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"

	"xiezuojun-server/internal/model"
)

// ===== 伏笔/钩子追踪 =====

func (h *Handler) ListThreads(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		projectID = c.Query("project_id")
	}
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id required"})
		return
	}
	pid, err := strconv.ParseUint(projectID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	var threads []model.StoryThread
	h.db.Where("project_id = ?", pid).Order("sort_order ASC, id ASC").Find(&threads)
	c.JSON(http.StatusOK, gin.H{"data": threads})
}

func (h *Handler) CreateThread(c *gin.Context) {
	var t model.StoryThread
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if t.ProjectID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id required"})
		return
	}
	if t.Type == "" {
		t.Type = "foreshadowing"
	}
	if t.Status == "" {
		t.Status = "open"
	}
	if err := h.db.Create(&t).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": t})
}

func (h *Handler) UpdateThread(c *gin.Context) {
	id := c.Param("id")
	var t model.StoryThread
	if err := h.db.First(&t, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	var updates map[string]any
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	delete(updates, "id")
	delete(updates, "project_id")
	delete(updates, "created_at")

	if err := h.db.Model(&t).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.db.First(&t, id)
	c.JSON(http.StatusOK, gin.H{"data": t})
}

func (h *Handler) DeleteThread(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&model.StoryThread{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}

// LoadOpenThreads 加载项目未回收的伏笔/钩子，供 AI 上下文注入
func (h *Handler) LoadOpenThreads(projectID uint) ([]model.StoryThread, error) {
	var threads []model.StoryThread
	err := h.db.Where("project_id = ? AND status = ?", projectID, "open").
		Order("urgency DESC, sort_order ASC").Find(&threads).Error
	return threads, err
}

// FormatThreadsForAI 把伏笔列表格式化为 AI 上下文文本
func FormatThreadsForAI(threads []model.StoryThread) string {
	if len(threads) == 0 {
		return ""
	}
	typeLabel := map[string]string{
		"foreshadowing": "伏笔",
		"hook":          "钩子",
		"promise":       "承诺",
		"suspense":      "悬念",
	}

	sort.Slice(threads, func(i, j int) bool {
		return threads[i].Urgency > threads[j].Urgency
	})

	var b []byte
	b = append(b, "【未回收的伏笔与钩子】\n"...)
	for _, t := range threads {
		label := typeLabel[t.Type]
		if label == "" {
			label = t.Type
		}
		chapter := ""
		if t.PlantedChapter != nil {
			chapter = "（第" + strconv.Itoa(*t.PlantedChapter) + "章埋设）"
		}
		b = append(b, "- ["+label+"] "+t.Title+chapter+"\n"...)
		if t.Description != "" {
			b = append(b, "  "+t.Description+"\n"...)
		}
	}
	b = append(b, "\n请在续写时注意呼应以上伏笔，适时推进或回收。\n"...)
	return string(b)
}
