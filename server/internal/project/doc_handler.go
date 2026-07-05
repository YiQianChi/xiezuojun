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

	if err := h.db.Model(&model.Document{}).Where("id = ?", id).Updates(map[string]interface{}{"status": 0, "search_text": ""}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	proj, err := h.getProject(doc.ProjectID)
	if err == nil && proj != nil {
		h.locks.Lock(proj.UserID)
		_ = h.ds.trashMove(proj.UserID, proj.ID, proj.Name, doc.ID, doc.ParentID, map[string]interface{}{
			"id": doc.ID, "title": doc.Title, "type": doc.Type,
		})
		// 如果删除的是文件夹且目录空了，删除目录
		if doc.Type == "volume" {
			_ = h.ds.removeFolderDir(proj.UserID, proj.ID, proj.Name, doc.ID)
		}
		h.locks.Unlock(proj.UserID)
	}

	projName := h.getProjectName(doc.ProjectID)
	_ = store.AppendChangelog(h.paths, userID, store.ChangeEntry{
		Action: "delete_doc", Project: projName,
		Doc: uintToStr(doc.ID), Summary: "删除: " + doc.Title,
	})

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
