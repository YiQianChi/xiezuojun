package project

import (
	"fmt"
	"strings"

	"xiezuojun-server/internal/model"
)

// 种子卷名直接叫「第一卷」——叫「文章」会和侧栏分区名混淆，新手不知道是干嘛的
const articleRootTitle = "第一卷"

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

func (h *Handler) createDefaultProjectDocuments(proj model.Project, branchID uint, firstChapterTitle string, settingTypes []string) error {
	type defaultDocX struct {
		Title    string
		Type     string
		Content  string
		ParentID *uint
	}

	// 2026-10-22 Boss 定：设定区平铺，不要「设定」文件夹——设定文档直接挂根级
	// 2026-10-22 起建书时作者自选要哪些设定文件；nil=全部（老调用方/导入保持原样），空数组=不建
	allSettingDefs := []defaultDocX{
		{Title: "大纲", Type: "outline", Content: "", ParentID: nil},
		{Title: "人物设定", Type: "characters", Content: "", ParentID: nil},
		{Title: "世界观", Type: "worldbuilding", Content: "", ParentID: nil},
		{Title: "故事主线", Type: "storyline", Content: "", ParentID: nil},
		{Title: "章节大纲", Type: "chapter_outline", Content: "", ParentID: nil},
	}
	var settingDefs []defaultDocX
	if settingTypes == nil {
		settingDefs = allSettingDefs
	} else {
		allowed := map[string]bool{}
		for _, t := range settingTypes {
			allowed[t] = true
		}
		for _, d := range allSettingDefs {
			if allowed[d.Type] {
				settingDefs = append(settingDefs, d)
			}
		}
	}

	for i, def := range settingDefs {
		doc := model.Document{
			ProjectID:  proj.ID,
			ParentID:   def.ParentID,
			Type:       def.Type,
			Title:      def.Title,
			SortOrder:  i,
			WordCount:  countWords(def.Content),
			SearchText: def.Content,
			Status:     1,
		}
		if err := h.db.Create(&doc).Error; err != nil {
			return err
		}
		if err := h.saveDocWithMeta(proj.UserID, proj.ID, proj.Name, doc, def.Content); err != nil {
			return err
		}
	}

	articleDoc := model.Document{
		ProjectID:  proj.ID,
		BranchID:   &branchID,
		Type:       "volume",
		Title:      articleRootTitle,
		SortOrder:  1,
		WordCount:  0,
		SearchText: "",
		Status:     1,
	}
	if err := h.db.Create(&articleDoc).Error; err != nil {
		return err
	}
	if err := h.saveDocWithMeta(proj.UserID, proj.ID, proj.Name, articleDoc, ""); err != nil {
		return err
	}

	chapterTitle := strings.TrimSpace(firstChapterTitle)
	if chapterTitle == "" {
		chapterTitle = "第一章"
	}
	if len([]rune(chapterTitle)) > 100 {
		chapterTitle = string([]rune(chapterTitle)[:100])
	}
	chapter := model.Document{
		ProjectID:  proj.ID,
		BranchID:   &branchID,
		ParentID:   &articleDoc.ID,
		Type:       "chapter",
		Title:      chapterTitle,
		SortOrder:  len(settingDefs) + 2,
		WordCount:  countWords(defaultDocContent("chapter", chapterTitle)),
		SearchText: defaultDocContent("chapter", chapterTitle),
		Status:     1,
	}
	if err := h.db.Create(&chapter).Error; err != nil {
		return err
	}
	return h.saveDocWithMeta(proj.UserID, proj.ID, proj.Name, chapter, defaultDocContent("chapter", chapterTitle))
}
