package project

import (
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"xiezuojun-server/internal/store"
)

// RestoreDocument 从回收站恢复文档
func (h *Handler) RestoreDocument(c *gin.Context) {
	userID := getUserID(c)
	var req struct {
		ProjectID uint `json:"project_id" binding:"required"`
		DocID     uint `json:"doc_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		h.emptyProjectTrash(c, userID, uint(projectID))
		return
	}
	cleaned, err := store.TrashCleanup(h.paths, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
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
	c.JSON(http.StatusOK, gin.H{"data": items})
}
