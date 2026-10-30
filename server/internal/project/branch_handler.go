package project

// 故事线（分支）功能已于 2026-10-30 移除（Boss 指示：UI/路由全下线）。
// 仅保留 ensureMainBranch：建书/导入时给文档一个默认 branch_id（数据列休眠保留，不迁移）。
import (
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
