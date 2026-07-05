package project

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"xiezuojun-server/internal/model"
	"xiezuojun-server/internal/store"
)

// Handler 项目/文档 API
type Handler struct {
	db       *gorm.DB
	dataRoot string // 旧路径前缀，兼容期保留
	paths    *store.PathConfig
	locks    *store.UserLocks
	ds       *DocStore
}

type documentUpdateRequest struct {
	Title              string `json:"title"`
	Content            string `json:"content"`
	LastKnownUpdatedAt string `json:"last_known_updated_at"`
	Force              bool   `json:"force"`
	CreateVersion      bool   `json:"create_version"`
	VersionLabel       string `json:"version_label"`
	HasContent         bool   `json:"-"`
}

func (r *documentUpdateRequest) UnmarshalJSON(data []byte) error {
	type alias documentUpdateRequest
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if err := json.Unmarshal(data, (*alias)(r)); err != nil {
		return err
	}
	_, r.HasContent = raw["content"]
	return nil
}

func sameUpdateInstant(a time.Time, b string) bool {
	if b == "" {
		return true
	}
	parsed, err := time.Parse(time.RFC3339Nano, b)
	if err != nil {
		return false
	}
	return a.UTC().Truncate(time.Millisecond).Equal(parsed.UTC().Truncate(time.Millisecond))
}

// NewHandler 创建 Handler
func NewHandler(db *gorm.DB, dataRoot string, paths *store.PathConfig, locks *store.UserLocks) *Handler {
	return &Handler{
		db:       db,
		dataRoot: dataRoot,
		paths:    paths,
		locks:    locks,
		ds:       NewDocStore(paths, locks),
	}
}

// ============== 项目 CRUD ==============

// LoadDocContent 读取文档正文（供 gateway 调用）
func (h *Handler) LoadDocContent(docID uint) (string, error) {
	return h.ds.LoadDocContent(docID)
}

// SettingDocDTO 精简版设定文档，用于跨包传输
type SettingDocDTO struct {
	ID      uint
	Title   string
	Type    string
	Content string
}

// LoadSettingDocs 加载指定项目的所有设定文档（大纲/人物/世界观）
// 优先读 documents.search_text；空则回退读文件系统。
func (h *Handler) LoadSettingDocs(projectID uint) ([]SettingDocDTO, error) {
	var docs []model.Document
	settingTypes := []string{"outline", "characters", "worldbuilding"}
	err := h.db.
		Select("id", "title", "type", "search_text", "sort_order").
		Where("project_id = ? AND status = 1 AND type IN (?)", projectID, settingTypes).
		Order("type, sort_order, id").
		Find(&docs).Error
	if err != nil {
		return nil, err
	}
	result := make([]SettingDocDTO, 0, len(docs))
	for _, d := range docs {
		content := d.SearchText
		if content == "" {
			// 回退：读文件
			if c, err := h.ds.LoadDocContent(d.ID); err == nil {
				content = c
			}
		}
		content = strings.TrimSpace(content)
		if content == "" {
			continue // 空的设定文档不注入
		}
		result = append(result, SettingDocDTO{
			ID:      d.ID,
			Title:   d.Title,
			Type:    d.Type,
			Content: content,
		})
	}
	return result, nil
}

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
		Name     string `json:"name" binding:"required"`
		CoverURL string `json:"cover_url"`
		Genre    string `json:"genre"`
		Tags     string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	proj := model.Project{
		UserID: userID.(string), Name: req.Name,
		CoverURL: req.CoverURL, Genre: req.Genre, Tags: req.Tags, Status: 1,
	}
	if err := h.db.Create(&proj).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 创建文件系统目录
	h.locks.Lock(proj.UserID)
	err := h.ds.ensureProjectDir(proj.UserID, proj.ID, proj.Name)
	h.locks.Unlock(proj.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create dir: " + err.Error()})
		return
	}

	mainBranch := model.StoryBranch{ProjectID: proj.ID, Name: "主线", Status: 1}
	if err := h.db.Create(&mainBranch).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create main branch: " + err.Error()})
		return
	}

	// 创建默认作品结构：一级资料文件 + 二级「文章」正文目录
	h.locks.Lock(proj.UserID)
	err = h.createDefaultProjectDocuments(proj, mainBranch.ID)
	h.locks.Unlock(proj.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create default docs: " + err.Error()})
		return
	}

	// 追加 changelog
	_ = store.AppendChangelog(h.paths, proj.UserID, store.ChangeEntry{
		Action: "create_project", Project: proj.Name, Summary: "创建项目",
	})

	c.JSON(http.StatusOK, gin.H{"data": proj})
}

func (h *Handler) GetProject(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if !h.requireProjectOwner(c, uint(id)) {
		return
	}
	var proj model.Project
	if err := h.db.First(&proj, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": proj})
}

func (h *Handler) UpdateProject(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if !h.requireProjectOwner(c, uint(id)) {
		return
	}
	var req struct {
		Name     string `json:"name"`
		CoverURL string `json:"cover_url"`
		Genre    string `json:"genre"`
		Tags     string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := applyProjectFields(req)
	if err := h.db.Model(&model.Project{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *Handler) DeleteProject(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if !h.requireProjectOwner(c, uint(id)) {
		return
	}
	if err := h.db.Model(&model.Project{}).Where("id = ?", id).Update("status", 0).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *Handler) ExportProject(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if !h.requireProjectOwner(c, uint(id)) {
		return
	}
	var proj model.Project
	if err := h.db.First(&proj, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var docs []model.Document
	h.db.Where("project_id = ? AND status = 1", id).Order("sort_order, id").Find(&docs)
	docContents := make([]map[string]interface{}, 0)
	for _, doc := range docs {
		content, _ := h.ds.loadDocContent(proj.UserID, proj.ID, proj.Name, doc.ID)
		// 如果新路径找不到，尝试旧路径（兼容未迁移的数据）
		if content == "" && doc.ParentID != nil && *doc.ParentID > 0 {
			content, _ = h.ds.loadDocContent(proj.UserID, proj.ID, proj.Name, doc.ID, *doc.ParentID)
		} else if content == "" {
			content, _ = h.ds.loadDocContentFallback(doc.ID)
		}
		docContents = append(docContents, map[string]interface{}{
			"id": doc.ID, "title": doc.Title, "type": doc.Type, "content": content,
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": map[string]interface{}{
		"project": proj, "documents": docs, "document_contents": docContents,
	}})
}

func (h *Handler) SearchProjects(c *gin.Context) {
	userID, _ := c.Get("userID")
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q required"})
		return
	}
	var projects []model.Project
	h.db.Where("user_id = ? AND status = 1 AND (name LIKE ? OR tags LIKE ?)",
		userID, "%"+query+"%", "%"+query+"%").Order("updated_at DESC").Find(&projects)
	c.JSON(http.StatusOK, gin.H{"data": projects})
}
