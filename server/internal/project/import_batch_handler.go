package project

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"xiezuojun-server/internal/model"
)

// 大书分批导入的追加接口：第一批由 ImportBook 建项目+建第一卷，
// 后续批次（ImportBookBatch）往指定项目里追加卷/章。
// 前端按 importBatchSize(500) 章一批顺序提交，每批带 project_id 续传。

// ImportBookBatch POST /api/projects/import/batch
func (h *Handler) ImportBookBatch(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := userID.(string)

	var req struct {
		ProjectID uint              `json:"project_id" binding:"required"`
		Name      string            `json:"name" binding:"required"` // 冗余携带，方便校验归属
		Volumes   []importVolumeReq `json:"volumes" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if len(req.Volumes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "本批没有内容"})
		return
	}

	var proj model.Project
	if err := h.db.First(&proj, req.ProjectID).Error; err != nil || proj.Name != strings.TrimSpace(req.Name) {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}
	if !h.requireProjectOwner(c, proj.ID) {
		return
	}

	var mainBranch model.StoryBranch
	h.db.Where("project_id = ? AND status = 1", proj.ID).First(&mainBranch)

	h.locks.Lock(uid)
	defer h.locks.Unlock(uid)

	created := 0
	totalWords := 0
	structureItems, _ := h.ds.loadStructureJSON(uid, proj.ID, proj.Name)
	if structureItems == nil {
		structureItems = make([]map[string]interface{}, 0, len(req.Volumes)*8)
	}

	for vi, vol := range req.Volumes {
		volTitle := strings.TrimSpace(vol.Title)
		if volTitle == "" {
			// 前端跨批摊平时空卷名的批 → 归入第一卷（ImportBook 的种子卷），不再另建「未命名卷」
			volTitle = "第一卷"
		}
		if len([]rune(volTitle)) > 60 {
			volTitle = string([]rune(volTitle)[:60])
		}

		// 同名卷复用（前端跨批的同一卷），否则新建
		var volDoc model.Document
		h.db.Where("project_id = ? AND type = 'volume' AND status = 1 AND title = ?", proj.ID, volTitle).First(&volDoc)
		if volDoc.ID == 0 {
			volDoc = model.Document{
				ProjectID: proj.ID, BranchID: &mainBranch.ID, Type: "volume",
				Title: volTitle, SortOrder: 100 + vi, Status: 1,
			}
			if err := h.db.Create(&volDoc).Error; err != nil {
				continue
			}
			_ = h.ds.ensureFolderDir(uid, proj.ID, proj.Name, volDoc.ID)
			structureItems = append(structureItems, map[string]interface{}{
				"id": volDoc.ID, "title": volTitle, "type": "volume", "status": "active",
			})
		}

		// 卷内已有章节数，接着排 sort_order
		var existing int64
		h.db.Model(&model.Document{}).
			Where("project_id = ? AND type = 'chapter' AND status = 1 AND parent_id = ?", proj.ID, volDoc.ID).
			Count(&existing)

		for ci, ch := range vol.Chapters {
			title := strings.TrimSpace(ch.Title)
			if title == "" {
				title = "未命名章节"
			}
			if len([]rune(title)) > 100 {
				title = string([]rune(title)[:100])
			}
			content := "# " + title + "\n\n" + ch.Content
			wc := countWords(content)
			totalWords += wc

			doc := model.Document{
				ProjectID: proj.ID, BranchID: &mainBranch.ID, ParentID: &volDoc.ID,
				Type: "chapter", Title: title, SortOrder: int(existing) + ci + 1,
				WordCount: wc, SearchText: content, Status: 1,
			}
			if err := h.db.Create(&doc).Error; err != nil {
				continue
			}
			if err := h.ds.saveDocContent(uid, proj.ID, proj.Name, doc.ID, content, volDoc.ID); err != nil {
				continue
			}
			structureItems = append(structureItems, map[string]interface{}{
				"id": doc.ID, "title": title, "type": "chapter", "status": "active",
			})
			created++
		}
	}

	_ = h.ds.saveStructureJSON(uid, proj.ID, proj.Name, structureItems)
	h.updateQuota(uid)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"project_id": proj.ID,
		"chapters":   created,
		"words":      totalWords,
	}})
}
