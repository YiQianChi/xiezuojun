// migrate_memory.go — 记忆数据迁移逻辑

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
)

// dbMemory 旧数据库中的记忆记录
type dbMemory struct {
	ID        uint   `gorm:"column:id"`
	ProjectID uint   `gorm:"column:project_id"`
	Category  string `gorm:"column:category"`
	Key       string `gorm:"column:key"`
	Value     string `gorm:"column:value"`
	CreatedAt string `gorm:"column:created_at"`
	UpdatedAt string `gorm:"column:updated_at"`
}

// migrateMemory 迁移记忆数据
func migrateMemory(db *gorm.DB, cfg migrateConfig, stats *migrateStats) {
	log.Println("🧠 开始迁移记忆数据...")

	// 1. 从 MySQL 迁移结构化记忆
	migrateDBMemory(db, cfg, stats)

	// 2. 从旧文件系统迁移平铺记忆文件
	migrateFileMemory(cfg, stats)

	log.Printf("🧠 记忆迁移完成: 迁移 %d, 跳过 %d, 错误 %d", stats.memoryMigrated, stats.skipped, stats.errors)
}

// migrateDBMemory 从数据库迁移结构化记忆
func migrateDBMemory(db *gorm.DB, cfg migrateConfig, stats *migrateStats) {
	var memories []dbMemory
	result := db.Table("memories").Find(&memories)
	if result.Error != nil {
		log.Printf("❌ 查询记忆失败: %v", result.Error)
		return
	}

	log.Printf("📋 找到 %d 条记忆记录", len(memories))

	// 按 project 分组，减少目录创建次数
	seenProjects := make(map[uint]string)
	for _, m := range memories {
		projDir, ok := seenProjects[m.ProjectID]
		if !ok {
			projDir = filepath.Join(cfg.memRoot, "db", fmt.Sprintf("project-%d", m.ProjectID))
			if !cfg.dryRun {
				os.MkdirAll(projDir, 0755)
			}
			seenProjects[m.ProjectID] = projDir
		}

		// 类别目录
		catDir := filepath.Join(projDir, categoryDir(m.Category))
		if !cfg.dryRun {
			os.MkdirAll(catDir, 0755)
		}

		dstFile := filepath.Join(catDir, sanitizeKey(m.Key))
		if fileExists(dstFile) {
			stats.skipped++
			continue
		}

		// 写入 JSON 格式记忆
		entry := map[string]string{
			"value":      m.Value,
			"created_at": m.CreatedAt,
			"updated_at": m.UpdatedAt,
		}
		writeMemEntry(cfg, dstFile, entry, stats)
	}
}

// migrateFileMemory 从旧平铺文件迁移
func migrateFileMemory(cfg migrateConfig, stats *migrateStats) {
	oldDir := filepath.Join(cfg.dataRoot, "memory")
	entries, err := os.ReadDir(oldDir)
	if err != nil {
		// 旧目录不存在，无数据可迁移
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		srcFile := filepath.Join(oldDir, entry.Name())
		data, err := os.ReadFile(srcFile)
		if err != nil {
			log.Printf("❌ 读取旧记忆文件失败 %s: %v", srcFile, err)
			stats.errors++
			continue
		}

		dstFile := filepath.Join(cfg.memRoot, "files", entry.Name())
		if fileExists(dstFile) {
			stats.skipped++
			continue
		}

		if cfg.dryRun {
			log.Printf("[dry-run] cp %s → %s", srcFile, dstFile)
			stats.memoryMigrated++
		} else if err := os.WriteFile(dstFile, data, 0644); err != nil {
			log.Printf("❌ 写入失败 %s: %v", dstFile, err)
			stats.errors++
		} else {
			stats.memoryMigrated++
		}
	}
}

// writeMemEntry 写入单条记忆
func writeMemEntry(cfg migrateConfig, path string, entry map[string]string, stats *migrateStats) {
	if cfg.dryRun {
		log.Printf("[dry-run] write %s", path)
		stats.memoryMigrated++
		return
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		stats.errors++
		return
	}
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		log.Printf("❌ 写入记忆失败 %s: %v", path, err)
		stats.errors++
		return
	}
	stats.memoryMigrated++
}

// categoryDir 将类别转为目录名（小写、下划线）
func categoryDir(cat string) string {
	if cat == "" {
		return "uncategorized"
	}
	return cat
}

// sanitizeKey 安全化记忆键名
func sanitizeKey(key string) string {
	if key == "" {
		return "unnamed.json"
	}
	sanitized := toSafeName(key)
	if !strings.HasSuffix(sanitized, ".json") {
		sanitized += ".json"
	}
	return sanitized
}
