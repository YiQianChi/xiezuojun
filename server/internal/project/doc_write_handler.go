package project

import (
	"net/http"
	"strings"
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

		Icon      string `json:"icon"`
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
		Icon: req.Icon,
	}

	// 章节没带内容时自动给标题头模板，和种子首章保持一致（之前新建章节是空内容，两不一致）
	if req.Type == "chapter" && strings.TrimSpace(req.Content) == "" {
		doc.SearchText = "# " + req.Title + "\n\n"
		req.Content = doc.SearchText
		doc.WordCount = countWords(req.Content)
	}

	// 文件夹层级限制：最多 3 层
	if req.Type == "volume" && req.ParentID != nil {
		parentDepth := h.folderDepth(req.ProjectID, *req.ParentID)
		if parentDepth >= maxFolderDepth-1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件夹最多嵌套 3 层，请选择更上层的文件夹"})
			return
		}
	}

	// 设定区平铺后只有卷（volume）能当父级；设定文档是纯文件，不能挂子文档
	if req.ParentID != nil && *req.ParentID > 0 {
		var parent model.Document
		if err := h.db.First(&parent, *req.ParentID).Error; err == nil && parent.Type != "volume" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "只有卷下面才能新建内容"})
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
	if req.AIInclude != nil {
		updates["ai_include"] = *req.AIInclude
	}
	if req.Icon != "" {
		updates["icon"] = req.Icon
	}
	if req.DocStatus != "" {
		updates["doc_status"] = req.DocStatus
	}
	if req.WordGoal != nil {
		updates["word_goal"] = *req.WordGoal
	}
	if err := h.db.Model(&model.Document{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ponytail: 写字数日志——只有 delta!=0 才记，避免噪音
	if req.HasContent {
		newWords := countWords(req.Content)
		delta := newWords - doc.WordCount
		if delta != 0 {
			h.db.Create(&model.WordCountLog{
				UserID:    userID,
				ProjectID: doc.ProjectID,
				DocID:     doc.ID,
				Delta:     delta,
				Total:     newWords,
			})
		}
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
