package project

import (
	"fmt"

	"xiezuojun-server/internal/model"
)

const articleRootTitle = "文章"

func defaultDocContent(docType, title string) string {
	switch docType {
	case "outline":
		return "# 大纲\n\n## 一句话简介\n\n\n## 主线目标\n\n\n## 核心冲突\n\n\n## 重要剧情节点\n\n\n## 结局方向\n\n"
	case "characters":
		return "# 人物设定\n\n## 主角\n\n\n## 重要配角\n\n\n## 反派\n\n\n## 角色关系\n\n"
	case "worldbuilding":
		return "# 世界观\n\n## 时代背景\n\n\n## 地理/势力\n\n\n## 能力体系\n\n\n## 规则与禁忌\n\n"
	case "storyline":
		return "# 故事主线\n\n## 主线脉络\n\n\n## 核心矛盾\n\n\n## 关键转折\n\n"
	case "chapter_outline":
		return "# 章节大纲\n\n## 卷一\n\n\n## 卷二\n\n"
	case "chapter":
		return fmt.Sprintf("# %s\n\n", title)
	default:
		return ""
	}
}

func (h *Handler) createDefaultProjectDocuments(proj model.Project, branchID uint) error {
	// ponytail: tab_scope 按类型固定，与前端 createDoc 一致
	type defaultDocX struct {
		Title    string
		Type     string
		TabScope string
		Content  string
		ParentID *uint
	}
	defs := []defaultDocX{
		{Title: "大纲", Type: "outline", TabScope: "setting", Content: ""},
		{Title: "人物设定", Type: "characters", TabScope: "setting", Content: ""},
		{Title: "世界观", Type: "worldbuilding", TabScope: "setting", Content: ""},
		{Title: "故事主线", Type: "storyline", TabScope: "setting", Content: ""},
		{Title: "章节大纲", Type: "chapter_outline", TabScope: "setting", Content: ""},
		{Title: articleRootTitle, Type: "volume", TabScope: "chapter", Content: ""},
	}

	var articleRootID uint
	for i, def := range defs {
		doc := model.Document{
			ProjectID:  proj.ID,
			ParentID:   def.ParentID,
			Type:       def.Type,
			Title:      def.Title,
			SortOrder:  i,
			TabScope:   def.TabScope,
			WordCount:  countWords(def.Content),
			SearchText: def.Content,
			Status:     1,
		}
		if def.Title == articleRootTitle && def.Type == "volume" {
			doc.BranchID = &branchID
		}
		if err := h.db.Create(&doc).Error; err != nil {
			return err
		}
		if err := h.saveDocWithMeta(proj.UserID, proj.ID, proj.Name, doc, def.Content); err != nil {
			return err
		}
		if def.Title == articleRootTitle && def.Type == "volume" {
			articleRootID = doc.ID
		}
	}

	chapter := model.Document{
		ProjectID:  proj.ID,
		BranchID:   &branchID,
		ParentID:   &articleRootID,
		Type:       "chapter",
		Title:      "第一章",
		SortOrder:  len(defs),
		TabScope:   "chapter",
		WordCount:  countWords(defaultDocContent("chapter", "第一章")),
		SearchText: defaultDocContent("chapter", "第一章"),
		Status:     1,
	}
	if err := h.db.Create(&chapter).Error; err != nil {
		return err
	}
	return h.saveDocWithMeta(proj.UserID, proj.ID, proj.Name, chapter, defaultDocContent("chapter", "第一章"))
}
