package project

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"xiezuojun-server/internal/model"
	"xiezuojun-server/internal/store"
)

// CreateDocument 创建文档
func (h *Handler) CreateDocument(c *gin.Context) {
	userID := getUserID(c)
	var req struct {
		ProjectID uint   `json:"project_id" binding:"required"`
		BranchID  *uint  `json:"branch_id"`
		ParentID  *uint  `json:"parent_id"`
		Type      string `json:"type" binding:"required"`
		Title     string `json:"title" binding:"required"`
		SortOrder int    `json:"sort_order"`
		Content   string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !h.requireProjectOwner(c, req.ProjectID) {
		return
	}

	doc := model.Document{
		ProjectID: req.ProjectID, BranchID: req.BranchID, ParentID: req.ParentID,
		Type: req.Type, Title: req.Title, SortOrder: req.SortOrder,
		WordCount: countWords(req.Content), SearchText: req.Content, Status: 1,
	}

	// 文件夹层级限制：最多 3 层
	if req.Type == "volume" && req.ParentID != nil {
		parentDepth := h.folderDepth(req.ProjectID, *req.ParentID)
		if parentDepth >= maxFolderDepth-1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件夹最多嵌套 3 层，请选择更上层的文件夹"})
			return
		}
	}

	if err := h.db.Create(&doc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	projName := h.getProjectName(req.ProjectID)
	h.locks.Lock(userID)
	err := h.saveDocWithMeta(userID, req.ProjectID, projName, doc, req.Content)
	h.locks.Unlock(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save file: " + err.Error()})
		return
	}

	_ = store.AppendChangelog(h.paths, userID, store.ChangeEntry{
		Action: "create_doc", Project: projName,
		Doc: uintToStr(doc.ID), Summary: "创建: " + doc.Title,
	})

	c.JSON(http.StatusOK, gin.H{"data": doc})
}

// UpdateDocument 更新文档
func (h *Handler) UpdateDocument(c *gin.Context) {
	userID := getUserID(c)
	id := c.Param("id")
	var req documentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doc, err := h.findDoc(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if !h.requireProjectOwner(c, doc.ProjectID) {
		return
	}

	if req.HasContent && !req.Force && !sameUpdateInstant(doc.UpdatedAt, req.LastKnownUpdatedAt) {
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
		c.JSON(http.StatusConflict, gin.H{
			"error": "云端已有新版本，本地草稿已保留",
			"code":  "document_conflict",
			"data": gin.H{
				"document": doc,
				"content":  content,
			},
		})
		return
	}

	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.HasContent {
		updates["word_count"] = countWords(req.Content)
		updates["search_text"] = req.Content
	}
	if err := h.db.Model(&model.Document{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	proj, err := h.getProject(doc.ProjectID)
	if err == nil && proj != nil {
		projName := proj.Name
		h.locks.Lock(proj.UserID)
		// 自动保存只更新当前正文；用户主动保存时才创建历史版本。
		if req.HasContent {
			h.updateDocWithMeta(proj.UserID, proj.ID, projName, *doc, req.Content, req.CreateVersion, req.VersionLabel)
		} else if req.Title != "" {
			// 仅标题变更，只更新 structure.json
			h.updateStructure(proj.UserID, proj.ID, projName, doc.ID, req.Title, doc.Type, "update")
		}
		h.locks.Unlock(proj.UserID)
	}

	projName := h.getProjectName(doc.ProjectID)
	_ = store.AppendChangelog(h.paths, userID, store.ChangeEntry{
		Action: "update_doc", Project: projName,
		Doc: uintToStr(doc.ID), Summary: "更新: " + req.Title,
	})

	var updatedDoc model.Document
	if err := h.db.First(&updatedDoc, doc.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated", "data": updatedDoc})
}
