package project

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"xiezuojun-server/internal/model"
	"xiezuojun-server/internal/store"
)

// TXT 导入：前端完成编码识别+正则分章+预览，后端只负责按结构批量落库。
// 复用 CreateProject 的建项目流程（默认结构里已有种子「第一卷」和空「第一章」，
// 导入时复用它们承载第一卷第一章，避免残留空节点）。

type importChapterReq struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content"`
}

type importVolumeReq struct {
	Title    string             `json:"title"`
	Chapters []importChapterReq `json:"chapters" binding:"required"`
}

const (
	importMaxChapters = 3000
	importMaxBytes    = 30 << 20 // 30MB 纯文本，约 1500 万字，够用且防呆
)

// ImportBook POST /api/projects/import
func (h *Handler) ImportBook(c *gin.Context) {
	userID, _ := c.Get("userID")
	var req struct {
		Name    string            `json:"name" binding:"required"`
		Volumes []importVolumeReq `json:"volumes" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误：需要书名和章节结构"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "书名不能为空"})
		return
	}
	if len([]rune(req.Name)) > 60 {
		req.Name = string([]rune(req.Name)[:60])
	}

	totalChapters := 0
	totalBytes := 0
	for _, v := range req.Volumes {
		totalChapters += len(v.Chapters)
		for _, ch := range v.Chapters {
			totalBytes += len(ch.Content)
		}
	}
	if totalChapters == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有可导入的章节"})
		return
	}
	if totalChapters > importMaxChapters {
		c.JSON(http.StatusBadRequest, gin.H{"error": "章节数超过 3000，请把文件拆分后再导入"})
		return
	}
	if totalBytes > importMaxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文本超过 30MB，请把文件拆分后再导入"})
		return
	}

	uid := userID.(string)

	// ---- 与 CreateProject 同流程建项目 ----
	proj := model.Project{UserID: uid, Name: req.Name, Status: 1}
	if err := h.db.Create(&proj).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.locks.Lock(uid)
	err := h.ds.ensureProjectDir(uid, proj.ID, proj.Name)
	h.locks.Unlock(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create dir: " + err.Error()})
		return
	}
	mainBranch := model.StoryBranch{ProjectID: proj.ID, Name: "主线", Status: 1}
	if err := h.db.Create(&mainBranch).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create main branch: " + err.Error()})
		return
	}
	h.locks.Lock(uid)
	err = h.createDefaultProjectDocuments(proj, mainBranch.ID, "")
	h.locks.Unlock(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create default docs: " + err.Error()})
		return
	}

	// 找种子卷/种子章节用于复用
	var seedVol model.Document
	h.db.Where("project_id = ? AND type = 'volume' AND status = 1", proj.ID).First(&seedVol)
	var seedChapter model.Document
	h.db.Where("project_id = ? AND type = 'chapter' AND status = 1", proj.ID).First(&seedChapter)

	h.locks.Lock(uid)
	defer h.locks.Unlock(uid)

	seedChapterUsed := false
	totalWords := 0
	created := 0

	for vi, vol := range req.Volumes {
		volTitle := strings.TrimSpace(vol.Title)
		if volTitle == "" {
			volTitle = "第一卷"
		}
		if len([]rune(volTitle)) > 60 {
			volTitle = string([]rune(volTitle)[:60])
		}

		var volDoc model.Document
		if vi == 0 && seedVol.ID > 0 {
			// 复用种子卷，改成导入的第一个卷名
			h.db.Model(&seedVol).Update("title", volTitle)
			volDoc = seedVol
		} else {
			volDoc = model.Document{
				ProjectID: proj.ID, BranchID: &mainBranch.ID, Type: "volume",
				Title: volTitle, SortOrder: vi + 1, Status: 1,
			}
			if err := h.db.Create(&volDoc).Error; err != nil {
				continue
			}
			if err := h.saveDocWithMeta(uid, proj.ID, proj.Name, volDoc, ""); err != nil {
				continue
			}
		}

		for ci, ch := range vol.Chapters {
			title := strings.TrimSpace(ch.Title)
			if title == "" {
				title = "未命名章节"
			}
			if len([]rune(title)) > 100 {
				title = string([]rune(title)[:100])
			}
			// 与种子章节同构：内容首行是 # 标题
			content := "# " + title + "\n\n" + ch.Content
			wc := countWords(content)
			totalWords += wc

			if vi == 0 && ci == 0 && seedChapter.ID > 0 && !seedChapterUsed {
				// 复用种子空章节，避免残留一个空「第一章」
				seedChapterUsed = true
				h.db.Model(&seedChapter).Updates(map[string]interface{}{
					"title": title, "parent_id": volDoc.ID, "branch_id": mainBranch.ID,
					"sort_order": ci + 1, "word_count": wc, "search_text": content,
				})
				_ = h.saveDocWithMeta(uid, proj.ID, proj.Name, seedChapter, content)
				created++
				continue
			}

			doc := model.Document{
				ProjectID: proj.ID, BranchID: &mainBranch.ID, ParentID: &volDoc.ID,
				Type: "chapter", Title: title, SortOrder: ci + 1,
				WordCount: wc, SearchText: content, Status: 1,
			}
			if err := h.db.Create(&doc).Error; err != nil {
				continue
			}
			if err := h.saveDocWithMeta(uid, proj.ID, proj.Name, doc, content); err != nil {
				continue
			}
			created++
		}
	}

	// 兜底：种子空章节没被复用（导入卷为空等边界）则软删，不留空「第一章」
	if !seedChapterUsed && seedChapter.ID > 0 {
		h.db.Model(&seedChapter).Updates(map[string]interface{}{"status": 0, "search_text": ""})
	}

	h.updateQuota(uid)
	_ = store.AppendChangelog(h.paths, uid, store.ChangeEntry{
		Action: "import_project", Project: proj.Name, Summary: "TXT 导入建书",
	})

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"project_id": proj.ID,
		"volumes":    len(req.Volumes),
		"chapters":   created,
		"words":      totalWords,
	}})
}
