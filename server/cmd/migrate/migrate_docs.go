// migrate_docs.go — 文档迁移逻辑

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gorm.io/gorm"

	"xiezuojun-server/internal/store"
)

// migrateDocuments 迁移文档数据
func migrateDocuments(db *gorm.DB, cfg migrateConfig, stats *migrateStats) {
	var projects []dbProject
	if err := db.Table("projects").Where("status != ?", "deleted").Find(&projects).Error; err != nil {
		log.Printf("❌ 查询项目失败: %v", err)
		return
	}
	log.Printf("📋 找到 %d 个项目", len(projects))

	for _, proj := range projects {
		stats.projects++
		projDir := filepath.Join(cfg.docRoot, fmt.Sprintf("%d", proj.UserID), sanitizeName(proj.Name, proj.ID))
		if cfg.dryRun {
			log.Printf("[dry-run] mkdir -p %s", projDir)
		} else if err := os.MkdirAll(projDir, 0755); err != nil {
			log.Printf("❌ 创建项目目录失败 %s: %v", projDir, err)
			stats.errors++
			continue
		}

		// 写 .meta.json
		meta := map[string]interface{}{
			"name":       proj.Name,
			"genre":      proj.Genre,
			"status":     proj.Status,
			"created_at": proj.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"updated_at": proj.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		writeMetaJSON(cfg, projDir, ".meta.json", meta, stats)

		// 查询文档列表并写 .structure.json
		structure := buildStructure(db, proj.ID, stats)
		writeMetaJSON(cfg, projDir, ".structure.json", structure, stats)

		// 迁移每个文档的正文
		migrateDocContents(db, cfg, proj.UserID, projDir, structure, stats)
	}
}

// buildStructure 从 MySQL 构建文档树
func buildStructure(db *gorm.DB, projectID uint, stats *migrateStats) []map[string]interface{} {
	var docs []dbDocument
	if err := db.Table("documents").Where("project_id = ? AND status != ?", projectID, "deleted").
		Order("sort_order ASC, id ASC").Find(&docs).Error; err != nil {
		log.Printf("❌ 查询文档失败 project=%d: %v", projectID, err)
		return nil
	}

	structure := make([]map[string]interface{}, 0, len(docs))
	for _, doc := range docs {
		item := map[string]interface{}{
			"id":         doc.ID,
			"title":      doc.Title,
			"type":       doc.DocType,
			"sort_order": doc.SortOrder,
		}
		if doc.ParentID != nil {
			item["parent_id"] = *doc.ParentID
		}
		structure = append(structure, item)
		stats.documents++
	}
	return structure
}

// migrateDocContents 迁移文档正文内容
func migrateDocContents(db *gorm.DB, cfg migrateConfig, userID uint, projDir string, structure []map[string]interface{}, stats *migrateStats) {
	for _, item := range structure {
		docID := uint(item["id"].(float64))
		title, _ := item["title"].(string)

		srcFile := filepath.Join(cfg.dataRoot, "documents", fmt.Sprintf("%d.txt", docID))
		data, err := os.ReadFile(srcFile)
		if err != nil {
			// 旧文件不存在，跳过
			stats.skipped++
			continue
		}

		dstFile := filepath.Join(projDir, fmt.Sprintf("%d.md", docID))
		if fileExists(dstFile) {
			stats.skipped++
			log.Printf("⏭️ 跳过已存在: %s", dstFile)
			continue
		}

		if cfg.dryRun {
			log.Printf("[dry-run] cp %s → %s (%s, %d bytes)", srcFile, dstFile, title, len(data))
			stats.written++
		} else if err := os.WriteFile(dstFile, data, 0644); err != nil {
			log.Printf("❌ 写入失败 %s: %v", dstFile, err)
			stats.errors++
		} else {
			stats.written++
			if stats.written%100 == 0 {
				log.Printf("📦 已迁移 %d 个文档...", stats.written)
			}
		}
	}
}

// sanitizeName 安全化目录名
func sanitizeName(name string, projID uint) string {
	return store.SanitizeName(name, projID)
}

func writeMetaJSON(cfg migrateConfig, dir, filename string, v interface{}, stats *migrateStats) {
	path := filepath.Join(dir, filename)
	if cfg.dryRun {
		log.Printf("[dry-run] write %s", path)
		stats.written++
		return
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Printf("❌ JSON 序列化失败 %s: %v", path, err)
		stats.errors++
		return
	}
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		log.Printf("❌ 写入失败 %s: %v", path, err)
		stats.errors++
	}
	stats.written++
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
