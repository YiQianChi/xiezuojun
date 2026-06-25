package project

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"xiezuojun-server/internal/model"
)

// Handler 项目/文档 API
type Handler struct {
	db       *gorm.DB
	dataRoot string
}

func NewHandler(db *gorm.DB, dataRoot string) *Handler {
	return &Handler{db: db, dataRoot: dataRoot}
}

// ============== 项目 CRUD ==============

func (h *Handler) ListProjects(c *gin.Context) {
	userID, _ := c.Get("userID")
	var projects []model.Project
	if err := h.db.Where("user_id = ? AND status = 1", userID).Order("updated_at DESC").Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": projects})
}

func (h *Handler) CreateProject(c *gin.Context) {
	userID, _ := c.Get("userID")
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	proj := model.Project{
		UserID: userID.(uint),
		Name:   req.Name,
		Status: 1,
	}
	if err := h.db.Create(&proj).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": proj})
}

func (h *Handler) UpdateProject(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Name     string `json:"name"`
		CoverURL string `json:"cover_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.CoverURL != "" {
		updates["cover_url"] = req.CoverURL
	}

	if err := h.db.Model(&model.Project{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *Handler) DeleteProject(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.db.Model(&model.Project{}).Where("id = ?", id).Update("status", 0).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ============== 文档 CRUD ==============

func (h *Handler) ListDocuments(c *gin.Context) {
	projectID, _ := strconv.Atoi(c.Query("project_id"))
	if projectID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id required"})
		return
	}

	var docs []model.Document
	if err := h.db.Where("project_id = ? AND status = 1", projectID).Order("sort_order, id").Find(&docs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": docs})
}

func (h *Handler) CreateDocument(c *gin.Context) {
	var req struct {
		ProjectID uint    `json:"project_id" binding:"required"`
		ParentID  *uint   `json:"parent_id"`
		Type      string  `json:"type" binding:"required"` // volume | chapter | note
		Title     string  `json:"title" binding:"required"`
		SortOrder int     `json:"sort_order"`
		Content   string  `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doc := model.Document{
		ProjectID: req.ProjectID,
		ParentID:  req.ParentID,
		Type:      req.Type,
		Title:     req.Title,
		SortOrder: req.SortOrder,
		WordCount: len([]rune(req.Content)),
		Status:    1,
	}
	if err := h.db.Create(&doc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 保存内容到文件
	if req.Content != "" {
		_ = h.saveDocContent(doc.ID, req.Content)
	}

	c.JSON(http.StatusOK, gin.H{"data": doc})
}

func (h *Handler) GetDocument(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var doc model.Document
	if err := h.db.First(&doc, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	content, _ := h.LoadDocContent(uint(id))
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"document": doc,
			"content":  content,
		},
	})
}

func (h *Handler) UpdateDocument(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Content != "" {
		updates["word_count"] = len([]rune(req.Content))
		_ = h.saveDocContent(uint(id), req.Content)
	}

	if err := h.db.Model(&model.Document{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *Handler) DeleteDocument(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.db.Model(&model.Document{}).Where("id = ?", id).Update("status", 0).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
