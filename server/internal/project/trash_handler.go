package project

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"xiezuojun-server/internal/model"
	"xiezuojun-server/internal/store"
)

// RestoreDocument 从回收站恢复
func (h *Handler) RestoreDocument(c *gin.Context) {
	userID := getUserID(c)
	var req struct {
		ProjectID uint `json:"project_id"`
		DocID     uint `json:"doc_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ponytail: project_id 有值但 doc_id=0 → 恢复整个 project（软删恢复）
	if req.ProjectID > 0 && req.DocID == 0 {
		if !h.requireProjectOwner(c, req.ProjectID) {
			return
		}
		if err := h.db.Model(&model.Project{}).Where("id = ?", req.ProjectID).Update("status", 1).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "restored"})
		return
	}

	if req.DocID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "doc_id required"})
		return
	}

	if !h.requireProjectOwner(c, req.ProjectID) {
		return
	}

	projName := h.getProjectName(req.ProjectID)
	trashDir := h.paths.TrashDir(userID, req.ProjectID, projName)
	docName := uintToStr(req.DocID) + ".md"
	trashPath := trashDir + "/" + docName
	destPath := h.ds.docPath(userID, req.ProjectID, projName, req.DocID)

	h.locks.Lock(userID)
	err := store.TrashRestore(trashPath, destPath)
	if err == nil {
		h.db.Exec("UPDATE documents SET status = 1 WHERE id = ?", req.DocID)
		_ = store.AppendChangelog(h.paths, userID, store.ChangeEntry{
			Action: "restore_doc", Project: projName,
			Doc: uintToStr(req.DocID), Summary: "恢复文档",
		})
	}
	h.locks.Unlock(userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "restored"})
}

// emptyProjectTrash 清空指定项目回收站
func (h *Handler) emptyProjectTrash(c *gin.Context, userID string, projectID uint) {
	projName := h.getProjectName(projectID)
	trashDir := h.paths.TrashDir(userID, projectID, projName)
	if err := os.RemoveAll(trashDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "emptied"})
}

// EmptyTrash 清空回收站
func (h *Handler) EmptyTrash(c *gin.Context) {
	userID := getUserID(c)
	projectID, _ := strconv.Atoi(c.Query("project_id"))
	if projectID > 0 {
		if !h.requireProjectOwner(c, uint(projectID)) {
			return
		}
		h.emptyProjectTrash(c, userID, uint(projectID))
		return
	}
	cleaned, err := store.TrashCleanup(h.paths, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// ponytail: 软删 project 的磁盘文件挪到回收目录，不真删
	var deadProjects []model.Project
	h.db.Where("user_id = ? AND status = 0", userID).Find(&deadProjects)
	dateStr := time.Now().Format("20060102")
	for _, p := range deadProjects {
		srcDir := h.paths.ProjectDir(userID, p.ID, p.Name)
		dstDir := filepath.Join(filepath.Dir(h.paths.DocRoot), "recycleBin", dateStr,
			fmt.Sprintf("%s_%s", store.SanitizeName(p.Name, p.ID), userID))
		_ = os.MkdirAll(filepath.Dir(dstDir), 0755)
		_ = os.Rename(srcDir, dstDir)
	}
	h.db.Where("user_id = ? AND status = 0", userID).Delete(&model.Project{})
	c.JSON(http.StatusOK, gin.H{"message": "emptied", "cleaned": cleaned})
}

// TrashListHandler 列出回收站内容
func (h *Handler) TrashListHandler(c *gin.Context) {
	userID := getUserID(c)
	items, err := store.TrashList(h.paths, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 也列出软删的 project（status=0）
	var deletedProjects []model.Project
	h.db.Where("user_id = ? AND status = 0", userID).Find(&deletedProjects)
	for _, p := range deletedProjects {
		items = append(items, map[string]interface{}{
			"type":       "project",
			"id":         p.ID,
			"title":      p.Name,
			"project_id": p.ID,
			"trashed_at": p.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}
