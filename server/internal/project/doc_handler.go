package project

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"xiezuojun-server/internal/model"
	"xiezuojun-server/internal/store"
)

// ListDocuments 列出项目文档
func (h *Handler) ListDocuments(c *gin.Context) {
	projectID, _ := strconv.Atoi(c.Query("project_id"))
	if projectID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id required"})
		return
	}
	if !h.requireProjectOwner(c, uint(projectID)) {
		return
	}

	branchID, _ := strconv.Atoi(c.Query("branch_id"))
	if branchID == 0 {
		branchID = int(h.ensureMainBranch(uint(projectID)))
	}

	var docs []model.Document
	q := h.db.Where("project_id = ? AND status = 1", projectID)
	if branchID > 0 {
		q = q.Where("branch_id IS NULL OR branch_id = ?", branchID)
	}
	if err := q.Order("sort_order, id").Find(&docs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": docs, "source": "db", "branch_id": branchID})
}

// GetDocument 获取文档详情（含正文）
func (h *Handler) GetDocument(c *gin.Context) {
	doc, err := h.findDoc(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if !h.requireProjectOwner(c, doc.ProjectID) {
		return
	}

	content := ""
	proj, err := h.getProject(doc.ProjectID)
	if err == nil && proj != nil {
		if doc.ParentID != nil && *doc.ParentID > 0 {
			content, _ = h.ds.loadDocContent(proj.UserID, proj.ID, proj.Name, doc.ID, *doc.ParentID)
		} else {
			content, _ = h.ds.loadDocContent(proj.UserID, proj.ID, proj.Name, doc.ID)
		}
	}
	if content == "" {
		content, _ = h.ds.loadDocContentFallback(doc.ID)
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"document": doc, "content": content}})
}

// DeleteDocument 删除文档（移入回收站）
// 文件夹删除：文件夹+所有子文档作为1个 trash_group 整体进回收站，算1个名额
func (h *Handler) DeleteDocument(c *gin.Context) {
	userID := getUserID(c)
	id := c.Param("id")
	doc, err := h.findDoc(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if !h.requireProjectOwner(c, doc.ProjectID) {
		return
	}

	isFolder := doc.Type == "volume" || doc.Type == "outline" || doc.Type == "characters" ||
		doc.Type == "worldbuilding" || doc.Type == "storyline" || doc.Type == "chapter_outline"

	trashGroup := doc.ID // ponytail: 用文件夹doc ID作为组号，保证唯一
	softDeleteIDs := []uint{doc.ID}

	if isFolder {
		// 递归收集所有子文档
		var children []model.Document
		h.db.Where("project_id = ? AND status = 1 AND parent_id = ?", doc.ProjectID, doc.ID).Find(&children)
		for _, ch := range children {
			softDeleteIDs = append(softDeleteIDs, ch.ID)
		}
		// 嵌套文件夹的孙子
		for len(children) > 0 {
			var nextLevel []model.Document
			for _, ch := range children {
				var gc []model.Document
				h.db.Where("project_id = ? AND status = 1 AND parent_id = ?", doc.ProjectID, ch.ID).Find(&gc)
				for _, g := range gc {
					softDeleteIDs = append(softDeleteIDs, g.ID)
					nextLevel = append(nextLevel, g)
				}
			}
			children = nextLevel
		}
	}

	// 软删：status=0 + 设 trash_group + 清搜索缓存
	h.db.Model(&model.Document{}).Where("id IN ?", softDeleteIDs).
		Updates(map[string]interface{}{"status": 0, "search_text": "", "trash_group": trashGroup})

	// 移文件到 .trash/
	proj, err := h.getProject(doc.ProjectID)
	if err == nil && proj != nil {
		h.locks.Lock(proj.UserID)
		for _, did := range softDeleteIDs {
			var d model.Document
			if h.db.First(&d, did).Error != nil {
				continue
			}
			_ = h.ds.trashMove(proj.UserID, proj.ID, proj.Name, d.ID, d.ParentID, map[string]interface{}{
				"id": d.ID, "title": d.Title, "type": d.Type, "trash_group": trashGroup,
			})
		}
		if isFolder {
			_ = h.ds.removeFolderDir(proj.UserID, proj.ID, proj.Name, doc.ID)
		}
		h.locks.Unlock(proj.UserID)
	}

	projName := h.getProjectName(doc.ProjectID)
	_ = store.AppendChangelog(h.paths, userID, store.ChangeEntry{
		Action: "delete_doc", Project: projName,
		Doc: uintToStr(doc.ID), Summary: "删除: " + doc.Title,
	})

	// ponytail: FIFO 淘汰 — 按 trash_group 算名额，超过30组删最旧的组
	h.enforceTrashLimit(userID, doc.ProjectID)

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
