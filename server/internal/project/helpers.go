package project

import (
	"log"
	"os"
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

// MigrateFolderFiles 将所有有 parentID 的文档文件迁移到文件夹目录
// 用于一次性迁移存量数据，幂等执行（已迁移的跳过）
func (h *Handler) MigrateFolderFiles() {
	var docs []model.Document
	h.db.Where("status = 1 AND parent_id IS NOT NULL").Find(&docs)
	if len(docs) == 0 {
		return
	}

	// 按 project 分组处理
	projCache := make(map[uint]*projectInfo)
	for _, doc := range docs {
		proj, ok := projCache[doc.ProjectID]
		if !ok {
			var err error
			proj, err = h.getProject(doc.ProjectID)
			if err != nil || proj == nil {
				continue
			}
			projCache[doc.ProjectID] = proj
		}

		// 文件夹类型：创建目录
		if doc.Type == "volume" {
			_ = h.ds.ensureFolderDir(proj.UserID, proj.ID, proj.Name, doc.ID)
			continue
		}

		// 文档类型：从根目录迁移到文件夹目录
		if doc.ParentID != nil && *doc.ParentID > 0 {
			oldPath := h.ds.docPath(proj.UserID, proj.ID, proj.Name, doc.ID)
			newPath := h.ds.docPath(proj.UserID, proj.ID, proj.Name, doc.ID, *doc.ParentID)
			// 新路径不存在但旧路径存在 → 需要迁移
			if _, err := os.Stat(oldPath); err == nil {
				if _, err := os.Stat(newPath); err != nil {
					_ = h.ds.ensureFolderDir(proj.UserID, proj.ID, proj.Name, *doc.ParentID)
					os.Rename(oldPath, newPath)
				}
			}
		}
	}
	log.Printf("[MIGRATE] folder file migration checked: %d docs with parent_id", len(docs))
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
	if doc.ParentID != nil && *doc.ParentID > 0 {
		_ = h.ds.ensureFolderDir(userID, projectID, projName, *doc.ParentID)
	}
	if err != nil {
		return err
	}
	h.updateStructure(userID, projectID, projName, doc.ID, doc.Title, doc.Type, "create")
	return nil
}

// updateDocWithMeta 更新文档文件并刷新结构/配额/变更日志
func (h *Handler) updateDocWithMeta(userID string, projectID uint, projName string, doc model.Document, content string) {
	_ = h.ds.saveDocContent(userID, projectID, projName, doc.ID, content)
	if doc.ParentID != nil && *doc.ParentID > 0 {
		_ = h.ds.ensureFolderDir(userID, projectID, projName, *doc.ParentID)
	}
	h.updateStructure(userID, projectID, projName, doc.ID, doc.Title, doc.Type, "update")
	h.updateQuota(userID)
}

// rebuildStructure 从 DB 全量重建 .structure.json
func (h *Handler) rebuildStructure(userID string, projectID uint, projName string) {
	var docs []model.Document
	h.db.Where("project_id = ? AND status = 1", projectID).Order("sort_order ASC, id ASC").Find(&docs)
	structure := make([]map[string]interface{}, 0, len(docs))
	for i, d := range docs {
		entry := map[string]interface{}{
			"id": d.ID, "title": d.Title, "type": d.Type, "status": "active",
		}
		if d.ParentID != nil {
			entry["parent_id"] = *d.ParentID
		}
		entry["sort_order"] = i
		structure = append(structure, entry)
	}
	_ = h.ds.saveStructureJSON(userID, projectID, projName, structure)
}

// isDescendant 检查 candidate 是否是 ancestor 自身或其后代（在 projectID 项目内）。
// 用于防止把节点拖到自己的子孙下造成循环引用。
func (h *Handler) isDescendant(projectID uint, candidate, ancestor uint) bool {
	if candidate == ancestor {
		return true
	}
	var docs []model.Document
	if err := h.db.Select("id", "parent_id").
		Where("project_id = ? AND status = 1", projectID).
		Find(&docs).Error; err != nil {
		return false
	}
	// 构建 child -> parent 映射
	parentOf := make(map[uint]uint, len(docs))
	for _, d := range docs {
		if d.ParentID != nil {
			parentOf[d.ID] = *d.ParentID
		}
	}
	// 从 candidate 沿 parent 向上爬，看是否能到 ancestor
	cur := candidate
	for i := 0; i < len(docs)+1; i++ {
		if cur == ancestor {
			return true
		}
		next, ok := parentOf[cur]
		if !ok {
			return false
		}
		cur = next
	}
	// 防御性：循环跳出，认为存在异常环
	return true
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
