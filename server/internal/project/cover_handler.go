package project

// 书籍封面上传：multipart "cover" → 存 dataRoot/covers/ → 写 projects.cover_url。
// 读取走 GET /api/covers/:file，nginx 不用改。

import (
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"xiezuojun-server/internal/model"
)

const maxCoverSize = 5 << 20 // 5MB

// UploadCover POST /projects/:id/cover（:id 兼容数字 ID 和 public_id）
func (h *Handler) UploadCover(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid, _ := userID.(string)

	projectID, err := h.resolveProjectParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}
	var project model.Project
	if err := h.db.Where("id = ? AND user_id = ? AND status = 1", projectID, uid).First(&project).Error; err != nil {
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
	// 解码 → 等比缩到最长边 600 的缩略图 → JPEG q82。
	// 保持完整构图不裁剪；用户传 20M 原图也压到几十 KB，展示流量扛得住
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取文件失败"})
		return
	}
	img, _, err := image.Decode(src)
	src.Close()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片格式无法识别"})
		return
	}
	cover := resizeThumb(img, 600)
	dir := filepath.Join(h.dataRoot, "covers")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage error"})
		return
	}
	name := fmt.Sprintf("%d-%s.jpg", project.ID, uuid.NewString()[:8])
	out, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save failed"})
		return
	}
	defer out.Close()
	if err := jpeg.Encode(out, cover, &jpeg.Options{Quality: 82}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encode failed"})
		return
	}
	url := "/api/covers/" + name
	if err := h.db.Model(&model.Project{}).Where("id = ?", project.ID).Update("cover_url", url).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"cover_url": url}})
}

// resizeThumb 等比缩小到最长边 max（不放大、不裁剪，保完整构图）
func resizeThumb(img image.Image, max int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= max && h <= max {
		return img
	}
	scale := float64(max) / float64(w)
	if h > w {
		scale = float64(max) / float64(h)
	}
	nw, nh := int(float64(w)*scale+0.5), int(float64(h)*scale+0.5)
	out := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(out, out.Bounds(), img, b, draw.Src, nil)
	return out
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
