package project

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"xiezuojun-server/internal/model"
)

// MoveDocument 移动文档（改 parentId + sortOrder，同步 structure.json，迁移文件）
func (h *Handler) MoveDocument(c *gin.Context) {
	userID := getUserID(c)
	id := c.Param("id")
	var req struct {
		ParentID  *string `json:"parent_id"`
		SortOrder int     `json:"sort_order"`
	}
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

	// 保存旧 parentID
	oldParentID := doc.ParentID

	updates := map[string]interface{}{"sort_order": req.SortOrder}
	if req.ParentID == nil || *req.ParentID == "" || *req.ParentID == "0" {
		updates["parent_id"] = nil
	} else {
		pid, _ := strconv.ParseUint(*req.ParentID, 10, 64)
		// 防止自引用
		if uint(pid) == doc.ID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot set parent to self"})
			return
		}
		// 防止循环引用：新 parent 不能是自身的后代
		if h.isDescendant(doc.ProjectID, uint(pid), doc.ID) {
			c.Writer.Header().Set("X-Move-Debug", "docID="+strconv.FormatUint(uint64(doc.ID), 10)+",newParent="+strconv.FormatUint(uint64(pid), 10))
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot move into own descendant", "debug": gin.H{"doc_id": doc.ID, "new_parent": pid, "project_id": doc.ProjectID}})
			return
		}
		// 文件夹层级限制：最多 3 层
		if doc.Type == "volume" {
			newParentDepth := h.folderDepth(doc.ProjectID, uint(pid))
			if newParentDepth >= maxFolderDepth-1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "文件夹最多嵌套 3 层，请选择更上层的文件夹"})
				return
			}
		}
		updates["parent_id"] = uint(pid)
	}

	if err := h.db.Model(&model.Document{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重载 doc 以获取最新值
	h.db.First(doc, id)

	// 同步 structure.json + 迁移文件
	proj, err := h.getProject(doc.ProjectID)
	if err == nil && proj != nil {
		h.locks.Lock(userID)
		h.rebuildStructure(proj.UserID, proj.ID, proj.Name)
		// 迁移文件到新的文件夹目录（非文件夹类型的文档）
		if doc.Type != "volume" {
			_ = h.ds.moveDocFile(proj.UserID, proj.ID, proj.Name, doc.ID, oldParentID, doc.ParentID)
		}
		// 文件夹移出时，尝试删除旧的空目录
		if doc.Type == "volume" && oldParentID != nil && *oldParentID > 0 {
			_ = h.ds.removeFolderDir(proj.UserID, proj.ID, proj.Name, *oldParentID)
		}
		h.locks.Unlock(userID)
	}

	c.JSON(http.StatusOK, gin.H{"data": doc})
}
