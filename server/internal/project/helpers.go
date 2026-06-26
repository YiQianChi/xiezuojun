package project

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"xiezuojun-server/internal/model"
	"xiezuojun-server/internal/store"
)

// getUserID 从 gin.Context 提取 userID
func getUserID(c *gin.Context) string {
	val, _ := c.Get("userID")
	if val == nil {
		return ""
	}
	return val.(string)
}

// getProjectName 通过 MySQL 查项目名
func (h *Handler) getProjectName(projectID uint) string {
	var name string
	h.db.Raw("SELECT name FROM projects WHERE id = ?", projectID).Scan(&name)
	return name
}

// getProject 查询项目记录
func (h *Handler) getProject(projectID uint) (*projectInfo, error) {
	var p projectInfo
	err := h.db.Raw("SELECT id, user_id, name FROM projects WHERE id = ?", projectID).Scan(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// projectInfo 项目基本信息（内部用）
type projectInfo struct {
	ID     uint   `gorm:"column:id"`
	UserID string `gorm:"column:user_id"`
	Name   string `gorm:"column:name"`
}

// updateStructure 更新 .structure.json（创建/更新/删除文档时调用）
func (h *Handler) updateStructure(userID string, projectID uint, projName string, docID uint, title, docType, action string) {
	structure, err := h.ds.loadStructureJSON(userID, projectID, projName)
	if err != nil || structure == nil {
		structure = make([]map[string]interface{}, 0)
	}
	if action == "create" {
		structure = append(structure, map[string]interface{}{
			"id": docID, "title": title, "type": docType, "status": "active",
		})
	} else if action == "delete" {
		for i, item := range structure {
			if uintVal, ok := item["id"].(float64); ok && uint(uintVal) == docID {
				structure = append(structure[:i], structure[i+1:]...)
				break
			}
		}
	} else {
		for _, item := range structure {
			if uintVal, ok := item["id"].(float64); ok && uint(uintVal) == docID {
				if title != "" {
					item["title"] = title
				}
				break
			}
		}
	}
	_ = h.ds.saveStructureJSON(userID, projectID, projName, structure)
}

// updateQuota 更新用户配额统计
func (h *Handler) updateQuota(userID string) {
	q, err := store.RecalcQuota(h.paths, userID)
	if err != nil {
		return
	}
	_ = q.Save(h.paths, userID)
}

// saveDocWithMeta 保存文档内容并更新结构/变更日志
func (h *Handler) saveDocWithMeta(userID string, projectID uint, projName string, doc model.Document, content string) error {
	err := h.ds.saveDocContent(userID, projectID, projName, doc.ID, content)
	if err != nil {
		return err
	}
	h.updateStructure(userID, projectID, projName, doc.ID, doc.Title, doc.Type, "create")
	return nil
}

// updateDocWithMeta 更新文档文件并刷新结构/配额/变更日志
func (h *Handler) updateDocWithMeta(userID string, projectID uint, projName string, doc model.Document, content string) {
	_ = h.ds.saveDocContent(userID, projectID, projName, doc.ID, content)
	h.updateStructure(userID, projectID, projName, doc.ID, doc.Title, doc.Type, "update")
	h.updateQuota(userID)
}

// findDoc 通过 ID 查文档（返回 *model.Document）
func (h *Handler) findDoc(id string) (*model.Document, error) {
	uid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	var doc model.Document
	if err := h.db.First(&doc, uint(uid)).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

// applyProjectFields 将非空字段合并到 updates map
func applyProjectFields(req struct {
	Name     string `json:"name"`
	CoverURL string `json:"cover_url"`
	Genre    string `json:"genre"`
	Tags     string `json:"tags"`
}) map[string]interface{} {
	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.CoverURL != "" {
		updates["cover_url"] = req.CoverURL
	}
	if req.Genre != "" {
		updates["genre"] = req.Genre
	}
	if req.Tags != "" {
		updates["tags"] = req.Tags
	}
	return updates
}
