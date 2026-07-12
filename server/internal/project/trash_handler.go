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

const MaxTrashItems = 30

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

	// ponytail: 恢复同 trash_group 的所有文档
	var trashed model.Document
	if err := h.db.First(&trashed, req.DocID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "doc not found"})
		return
	}
	groupID := trashed.TrashGroup
	if groupID == 0 {
		groupID = req.DocID // 单文档没有组，自身就是组
	}

	var docs []model.Document
	h.db.Where("project_id = ? AND status = 0 AND (trash_group = ? OR id = ?)", req.ProjectID, groupID, req.DocID).Find(&docs)

	projName := h.getProjectName(req.ProjectID)
	trashDir := h.paths.TrashDir(userID, req.ProjectID, projName)

	h.locks.Lock(userID)
	for _, d := range docs {
		docName := uintToStr(d.ID) + ".md"
		trashPath := filepath.Join(trashDir, docName)
		destPath := h.ds.docPath(userID, req.ProjectID, projName, d.ID, ptrUint(d.ParentID))
		if err := store.TrashRestore(trashPath, destPath); err == nil {
			h.db.Model(&model.Document{}).Where("id = ?", d.ID).
				Updates(map[string]interface{}{"status": 1, "trash_group": 0})
		}
	}
	_ = store.AppendChangelog(h.paths, userID, store.ChangeEntry{
		Action: "restore_doc", Project: projName,
		Doc: uintToStr(req.DocID), Summary: "恢复文档",
	})
	h.locks.Unlock(userID)

	c.JSON(http.StatusOK, gin.H{"message": "restored"})
}

// ptrUint 将 *uint 转为 uint（nil→0）
func ptrUint(p *uint) uint {
	if p == nil {
		return 0
	}
	return *p
}

// emptyProjectTrash 清空指定项目回收站
func (h *Handler) emptyProjectTrash(c *gin.Context, userID string, projectID uint) {
	projName := h.getProjectName(projectID)
	trashDir := h.paths.TrashDir(userID, projectID, projName)
	if err := os.RemoveAll(trashDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// ponytail: 硬删 DB 中 status=0 的文档
	h.db.Where("project_id = ? AND status = 0", projectID).Delete(&model.Document{})
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

	// ponytail: project_id 有值 → 只返回该项目的回收站，按 trash_group 去重显示顶级项
	if pidStr := c.Query("project_id"); pidStr != "" {
		pid, _ := strconv.Atoi(pidStr)
		if pid > 0 {
			if !h.requireProjectOwner(c, uint(pid)) {
				return
			}
			// 取每个 group 里 parent_id=NULL 或 trash_group=自身ID 的那个作为代表
			var docs []model.Document
			h.db.Where("project_id = ? AND status = 0", pid).Order("updated_at DESC").Find(&docs)

			// 按 group 去重，每组只显示代表项（trash_group=自身ID的记录，或没有group的第一条）
			seen := make(map[uint]bool)
			var result []map[string]interface{}
			for _, d := range docs {
				gid := d.TrashGroup
				if gid == 0 {
					gid = d.ID // 单文档自身为一组
				}
				if seen[gid] {
					continue
				}
				seen[gid] = true
				// 统计组内文件数
				var count int64
				h.db.Model(&model.Document{}).
					Where("project_id = ? AND status = 0 AND (trash_group = ? OR (trash_group = 0 AND id = ?))", pid, gid, gid).
					Count(&count)
				item := map[string]interface{}{
					"id":         d.ID,
					"title":      d.Title,
					"type":       d.Type,
					"updated_at": d.UpdatedAt,
				}
				if count > 1 {
					item["child_count"] = count - 1
				}
				result = append(result, item)
			}
			c.JSON(http.StatusOK, gin.H{"data": result})
			return
		}
	}

	// 无 project_id → 全局回收站（文件系统扫描，书架页用）
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

// enforceTrashLimit 删除文档后检查回收站上限，按 trash_group FIFO 淘汰最旧的组
func (h *Handler) enforceTrashLimit(userID string, projectID uint) {
	// ponytail: 每个 trash_group 算1个名额（单文档 trash_group=自身ID）
	var groups []uint
	h.db.Model(&model.Document{}).
		Where("project_id = ? AND status = 0 AND trash_group > 0", projectID).
		Distinct("trash_group").Pluck("trash_group", &groups)

	if int64(len(groups)) <= MaxTrashItems {
		return
	}

	// 按 updated_at ASC 淘汰最旧的组
	excess := len(groups) - MaxTrashItems
	type groupTime struct {
		TrashGroup uint
		UpdatedAt  time.Time
	}
	var oldestGroups []groupTime
	h.db.Model(&model.Document{}).
		Select("trash_group, MIN(updated_at) as updated_at").
		Where("project_id = ? AND status = 0 AND trash_group > 0", projectID).
		Group("trash_group").
		Order("updated_at ASC").
		Limit(excess).
		Scan(&oldestGroups)

	projName := h.getProjectName(projectID)
	trashDir := h.paths.TrashDir(userID, projectID, projName)
	for _, gt := range oldestGroups {
		// 硬删 DB 记录
		var docs []model.Document
		h.db.Where("trash_group = ?", gt.TrashGroup).Find(&docs)
		for _, d := range docs {
			h.db.Delete(&d)
			_ = os.Remove(filepath.Join(trashDir, fmt.Sprintf("%d.md", d.ID)))
			_ = os.Remove(filepath.Join(trashDir, fmt.Sprintf("%d.md.meta.json", d.ID)))
		}
	}
}
