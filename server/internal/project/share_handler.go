package project

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"xiezuojun-server/internal/model"
)

// ===== 章节分享：作者把单个章节生成公开只读链接 =====
// 开启 → documents.share_code 生成随机码 → /read/:code 免登录阅读；关闭 → 置 NULL 链接即死。

// EnableDocShare POST /documents/:id/share
func (h *Handler) EnableDocShare(c *gin.Context) {
	doc, err := h.findDoc(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if !h.requireProjectOwner(c, doc.ProjectID) {
		return
	}
	if doc.ShareCode == nil || *doc.ShareCode == "" {
		for i := 0; i < 3; i++ {
			code := newProjectPublicID()[:12]
			var cnt int64
			h.db.Model(&model.Document{}).Where("share_code = ?", code).Count(&cnt)
			if cnt == 0 {
				doc.ShareCode = &code
				break
			}
		}
		if doc.ShareCode == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "生成分享码失败"})
			return
		}
		if err := h.db.Model(&model.Document{}).Where("id = ?", doc.ID).Update("share_code", doc.ShareCode).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"share_code": *doc.ShareCode}})
}

// DisableDocShare DELETE /documents/:id/share —— 关闭后旧链接立刻 404
func (h *Handler) DisableDocShare(c *gin.Context) {
	doc, err := h.findDoc(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if !h.requireProjectOwner(c, doc.ProjectID) {
		return
	}
	h.db.Model(&model.Document{}).Where("id = ?", doc.ID).Update("share_code", nil)
	c.JSON(http.StatusOK, gin.H{"message": "已停止分享"})
}

// PublicRead GET /public/read/:code —— 免登录只读（路由注册在公开组）
func (h *Handler) PublicRead(c *gin.Context) {
	code := c.Param("code")
	if len(code) < 6 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var doc model.Document
	if err := h.db.Where("share_code = ? AND status = 1", code).First(&doc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在或已关闭"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	proj, err := h.getProject(doc.ProjectID)
	if err != nil || proj == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	content := ""
	if doc.ParentID != nil && *doc.ParentID > 0 {
		content, _ = h.ds.loadDocContent(proj.UserID, proj.ID, proj.Name, doc.ID, *doc.ParentID)
	} else {
		content, _ = h.ds.loadDocContent(proj.UserID, proj.ID, proj.Name, doc.ID)
	}
	if content == "" {
		content, _ = h.ds.loadDocContentFallback(doc.ID)
	}
	// unnamed 章（作者没起名）：分享出去显示自动编号「第N章」，与树/导出同口径
	title := strings.TrimSpace(doc.Title)
	if doc.Type == "chapter" && title == "" {
		var docs []model.Document
		h.db.Where("project_id = ? AND status = 1", doc.ProjectID).Order("sort_order, id").Find(&docs)
		ord := 0
		for _, d := range flattenDocTree(docs) {
			if d.Type != "chapter" || exportNoCountTitleRe.MatchString(strings.TrimSpace(d.Title)) {
				continue
			}
			ord++
			if d.ID == doc.ID {
				title = "第" + numToChinese(ord) + "章"
				break
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"code": code, "title": title, "project_name": proj.Name,
		"content": content, "word_count": doc.WordCount,
		"updated_at": doc.UpdatedAt.Format("2006-01-02"),
		"_ts":       time.Now().Unix(),
	}})
}
