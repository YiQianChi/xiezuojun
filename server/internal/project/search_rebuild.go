package project

import (
	"log"

	"xiezuojun-server/internal/model"
)

// RebuildMissingSearchText 从文件正文回填缺失的 search_text。
// search_text 是轻量搜索缓存；旧章节在版本上线前没有该字段，需要启动时补一次。
func (h *Handler) RebuildMissingSearchText() {
	var docs []model.Document
	if err := h.db.Where("status = 1 AND type = ? AND (search_text IS NULL OR search_text = '')", "chapter").Find(&docs).Error; err != nil {
		log.Printf("[search-index] list missing docs failed: %v", err)
		return
	}
	if len(docs) == 0 {
		return
	}

	updated := 0
	for _, doc := range docs {
		proj, err := h.getProject(doc.ProjectID)
		if err != nil || proj == nil {
			continue
		}

		content := ""
		if doc.ParentID != nil && *doc.ParentID > 0 {
			content, _ = h.ds.loadDocContent(proj.UserID, proj.ID, proj.Name, doc.ID, *doc.ParentID)
		} else {
			content, _ = h.ds.loadDocContent(proj.UserID, proj.ID, proj.Name, doc.ID)
		}
		if content == "" {
			content, _ = h.ds.loadDocContentFallback(doc.ID)
		}
		if content == "" {
			continue
		}

		if err := h.db.Model(&model.Document{}).Where("id = ?", doc.ID).Updates(map[string]interface{}{
			"search_text": content,
			"word_count":  countWords(content),
		}).Error; err != nil {
			log.Printf("[search-index] rebuild doc=%d failed: %v", doc.ID, err)
			continue
		}
		updated++
	}
	log.Printf("[search-index] rebuilt %d/%d missing chapter caches", updated, len(docs))
}
