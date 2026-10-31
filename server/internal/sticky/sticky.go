// Package sticky 便签/灵感随手记（右栏工作台 C 方案）。
// 一张表两种类型：type=note 贴在稿纸右缘的便签贴纸；type=insp 右栏「灵感」页签的点子。
package sticky

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Note 一条随手记
type Note struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    string    `json:"user_id" gorm:"index:idx_sticky_user_project;size:36"`
	ProjectID uint      `json:"project_id" gorm:"index:idx_sticky_user_project"`
	Type      string    `json:"type" gorm:"size:8;default:note"` // note=便签 insp=灵感
	Content   string    `json:"content" gorm:"type:text"`
	Color     string    `json:"color" gorm:"size:8;default:y"` // y/b/g
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var db *gorm.DB

// SetDB 注入（main 启动时；AutoMigrate 由 main 统一做）
func SetDB(d *gorm.DB) { db = d }

func RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/sticky", handleList)
	rg.POST("/sticky", handleCreate)
	rg.PATCH("/sticky/:id", handleUpdate)
	rg.DELETE("/sticky/:id", handleDelete)
}

func handleList(c *gin.Context) {
	uid := c.GetString("userID")
	pid, _ := strconv.ParseUint(c.Query("project_id"), 10, 64)
	if db == nil || pid == 0 {
		c.JSON(http.StatusOK, gin.H{"data": []Note{}})
		return
	}
	var out []Note
	db.Where("user_id = ? AND project_id = ?", uid, uint(pid)).Order("created_at DESC").Limit(200).Find(&out)
	if out == nil {
		out = []Note{}
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func handleCreate(c *gin.Context) {
	uid := c.GetString("userID")
	var req struct {
		ProjectID uint   `json:"project_id"`
		Type      string `json:"type"`
		Content   string `json:"content"`
		Color     string `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ProjectID == 0 || req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if req.Type != "insp" {
		req.Type = "note"
	}
	switch req.Color {
	case "y", "b", "g":
	default:
		req.Color = "y"
	}
	if len([]rune(req.Content)) > 500 {
		req.Content = string([]rune(req.Content)[:500])
	}
	n := Note{UserID: uid, ProjectID: req.ProjectID, Type: req.Type, Content: req.Content, Color: req.Color}
	if err := db.Create(&n).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": n})
}

func handleUpdate(c *gin.Context) {
	uid := c.GetString("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Content *string `json:"content"`
		Color   *string `json:"color"`
	}
	_ = c.ShouldBindJSON(&req)
	updates := map[string]interface{}{}
	if req.Content != nil && *req.Content != "" {
		v := *req.Content
		if len([]rune(v)) > 500 {
			v = string([]rune(v)[:500])
		}
		updates["content"] = v
	}
	if req.Color != nil && (*req.Color == "y" || *req.Color == "b" || *req.Color == "g") {
		updates["color"] = *req.Color
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nothing to update"})
		return
	}
	res := db.Model(&Note{}).Where("id = ? AND user_id = ?", uint(id), uid).Updates(updates)
	if res.Error != nil || res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "updated"})
}

func handleDelete(c *gin.Context) {
	uid := c.GetString("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	db.Where("id = ? AND user_id = ?", uint(id), uid).Delete(&Note{})
	c.JSON(http.StatusOK, gin.H{"data": "deleted"})
}
