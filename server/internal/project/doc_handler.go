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

	proj, err := h.getProject(uint(projectID))
	if err == nil && proj != nil {
		// 尝试从 .structure.json 获取文档 ID 列表和排序
		structure, err := h.ds.loadStructureJSON(proj.UserID, proj.ID, proj.Name)
		if err == nil && len(structure) > 0 {
			// 收集 ID，从 DB 批量查出完整 Document
			ids := make([]uint, 0, len(structure))
			idSet := make(map[uint]int) // id -> structure index
			for i, item := range structure {
				if idVal, ok := item["id"]; ok {
					id := uint(idVal.(float64))
					ids = append(ids, id)
					idSet[id] = i
				}
			}
			var docs []model.Document
			h.db.Where("id IN ? AND status = 1", ids).Find(&docs)
			// 按 structure 顺序排列，并补充 structure 中的 sort_order
			result := make([]model.Document, 0, len(structure))
			for i, item := range structure {
				if idVal, ok := item["id"]; ok {
					id := uint(idVal.(float64))
					for _, d := range docs {
						if d.ID == id {
							if so, ok := item["sort_order"]; ok {
								d.SortOrder = int(so.(float64))
							} else {
								d.SortOrder = i
							}
							result = append(result, d)
							break
						}
					}
				}
			}
			c.JSON(http.StatusOK, gin.H{"data": result, "source": "file"})
			return
		}
	}

	// fallback: MySQL 查询
	var docs []model.Document
	if err := h.db.Where("project_id = ? AND status = 1", projectID).Order("sort_order, id").Find(&docs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": docs, "source": "db"})
}

// GetDocument 获取文档详情（含正文）
func (h *Handler) GetDocument(c *gin.Context) {
	doc, err := h.findDoc(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
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
