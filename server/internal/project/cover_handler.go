package project

// 书籍封面上传：multipart "cover" → 存 dataRoot/covers/ → 写 projects.cover_url。
// 读取走 GET /api/covers/:file，nginx 不用改。

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"xiezuojun-server/internal/model"
)

const maxCoverSize = 5 << 20 // 5MB

// UploadCover POST /projects/:id/cover
func (h *Handler) UploadCover(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid, _ := userID.(string)

	var project model.Project
	if err := h.db.Where("id = ? AND user_id = ? AND status = 1", c.Param("id"), uid).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}
	file, err := c.FormFile("cover")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cover file required"})
		return
	}
	if file.Size > maxCoverSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片不能超过 5MB"})
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 jpg/png/webp"})
		return
	}
	dir := filepath.Join(h.dataRoot, "covers")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage error"})
		return
	}
	name := fmt.Sprintf("%d-%s%s", project.ID, uuid.NewString()[:8], ext)
	if err := c.SaveUploadedFile(file, filepath.Join(dir, name)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save failed"})
		return
	}
	url := "/api/covers/" + name
	if err := h.db.Model(&model.Project{}).Where("id = ?", project.ID).Update("cover_url", url).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"cover_url": url}})
}

// CoverFile GET /covers/:file（挂 /api/covers，文件名含随机段不可枚举）
func (h *Handler) CoverFile(c *gin.Context) {
	name := filepath.Base(c.Param("file"))
	if strings.Contains(name, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad name"})
		return
	}
	c.File(filepath.Join(h.dataRoot, "covers", name))
}
