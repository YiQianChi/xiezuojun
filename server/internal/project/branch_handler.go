package project

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"xiezuojun-server/internal/model"
)

func (h *Handler) ensureMainBranch(projectID uint) uint {
	var branch model.StoryBranch
	if err := h.db.Where("project_id = ? AND status = 1", projectID).Order("id ASC").First(&branch).Error; err == nil {
		return branch.ID
	}
	branch = model.StoryBranch{ProjectID: projectID, Name: "主线", Status: 1}
	if err := h.db.Create(&branch).Error; err != nil {
		return 0
	}
	return branch.ID
}

func (h *Handler) ListStoryBranches(c *gin.Context) {
	projectID, _ := strconv.Atoi(c.Param("id"))
	if projectID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project id required"})
		return
	}
	if !h.requireProjectOwner(c, uint(projectID)) {
		return
	}
	h.ensureMainBranch(uint(projectID))
	var branches []model.StoryBranch
	if err := h.db.Where("project_id = ? AND status = 1", projectID).Order("id ASC").Find(&branches).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": branches})
}

func (h *Handler) CreateStoryBranch(c *gin.Context) {
	projectID64, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	projectID := uint(projectID64)
	if projectID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project id required"})
		return
	}
	if !h.requireProjectOwner(c, projectID) {
		return
	}
	var req struct {
		Name           string `json:"name"`
		BaseBranchID   uint   `json:"base_branch_id"`
		ForkDocumentID uint   `json:"fork_document_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		req.Name = "新故事线"
	}
	if req.BaseBranchID == 0 {
		req.BaseBranchID = h.ensureMainBranch(projectID)
	}

	proj, err := h.getProject(projectID)
	if err != nil || proj == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	var forkDoc model.Document
	if err := h.db.Where("id = ? AND project_id = ? AND status = 1", req.ForkDocumentID, projectID).First(&forkDoc).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "fork document not found"})
		return
	}
	if forkDoc.Type != "chapter" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能从章节创建故事线"})
		return
	}

	base := req.BaseBranchID
	fork := req.ForkDocumentID
	branch := model.StoryBranch{
		ProjectID:      projectID,
		Name:           req.Name,
		BaseBranchID:   &base,
		ForkDocumentID: &fork,
		Status:         1,
	}
	if err := h.db.Create(&branch).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.copyBranchUpToFork(*proj, req.BaseBranchID, branch.ID, forkDoc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "copy branch docs: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": branch})
}

func (h *Handler) UpdateStoryBranch(c *gin.Context) {
	branchID, _ := strconv.Atoi(c.Param("branch_id"))
	if branchID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "branch id required"})
		return
	}
	// 校验分支所属项目归属
	var branch model.StoryBranch
	if err := h.db.First(&branch, branchID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "branch not found"})
		return
	}
	if !h.requireProjectOwner(c, branch.ProjectID) {
		return
	}
	var req struct {
		Name   string `json:"name"`
		Status *int   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{}
	if strings.TrimSpace(req.Name) != "" {
		updates["name"] = strings.TrimSpace(req.Name)
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if len(updates) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "noop"})
		return
	}
	if err := h.db.Model(&model.StoryBranch{}).Where("id = ?", branchID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *Handler) copyBranchUpToFork(proj projectInfo, baseBranchID uint, newBranchID uint, forkDoc model.Document) error {
	var baseDocs []model.Document
	if err := h.db.Where("project_id = ? AND status = 1 AND branch_id = ?", proj.ID, baseBranchID).
		Order("sort_order ASC, id ASC").Find(&baseDocs).Error; err != nil {
		return err
	}

	copyMap := map[uint]uint{}
	for _, d := range baseDocs {
		if d.Type != "volume" && d.Type != "chapter" {
			continue
		}
		if d.Type == "chapter" && d.SortOrder > forkDoc.SortOrder {
			continue
		}
		if d.Type == "volume" {
			// 复制所有正文文件夹，确保 1-30 章的父级目录完整。
		}

		oldID := d.ID
		newDoc := d
		newDoc.ID = 0
		newDoc.BranchID = &newBranchID
		if d.ParentID != nil {
			if mapped, ok := copyMap[*d.ParentID]; ok {
				newDoc.ParentID = &mapped
			} else {
				newDoc.ParentID = nil
			}
		}
		if err := h.db.Create(&newDoc).Error; err != nil {
			return err
		}
		copyMap[oldID] = newDoc.ID

		content := ""
		if d.Type != "volume" {
			content = h.loadDocumentContent(d)
		}
		if err := h.saveDocWithMeta(proj.UserID, proj.ID, proj.Name, newDoc, content); err != nil {
			return fmt.Errorf("save copied doc %d: %w", oldID, err)
		}
	}
	return nil
}
