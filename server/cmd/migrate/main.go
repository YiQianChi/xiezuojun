// Package migrate — 旧数据迁移脚本
// 从平铺文件 + MySQL 迁移到新的目录结构
//
// 用法：
//   go run ./cmd/migrate/ [flags]
//
// Flags:
//   --dry-run         只打印不执行
//   --doc-root        新文档根目录 (默认 /opt/xiezuojun/documents)
//   --mem-root        新记忆根目录 (默认 /opt/xiezuojun/memory)
//   --data-root       旧数据根目录 (默认 ./data)
//   --db-dsn          MySQL DSN（默认从 DB_DSN 环境变量读取）

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

// ========== 配置 ==========

type migrateConfig struct {
	dryRun   bool
	docRoot  string
	memRoot  string
	dataRoot string
	dbDSN    string
}

// ========== 数据模型（只读） ==========

type dbProject struct {
	ID        uint   `gorm:"column:id"`
	UserID    uint   `gorm:"column:user_id"`
	Name      string `gorm:"column:name"`
	Genre     string `gorm:"column:genre"`
	Status    string `gorm:"column:status"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type dbDocument struct {
	ID        uint   `gorm:"column:id"`
	ProjectID uint   `gorm:"column:project_id"`
	Title     string `gorm:"column:title"`
	DocType   string `gorm:"column:doc_type"`
	SortOrder int    `gorm:"column:sort_order"`
	ParentID  *uint  `gorm:"column:parent_id"`
	Status    string `gorm:"column:status"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ========== 主流程 ==========

func main() {
	cfg := parseFlags()

	db, err := connectDB(cfg.dbDSN)
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	stats := &migrateStats{}

	migrateDocuments(db, cfg, stats)
	migrateMemory(db, cfg, stats)
	printStats(stats)

	if cfg.dryRun {
		fmt.Println("\n✅ dry-run 完成，未做任何修改")
	} else {
		fmt.Println("\n✅ 迁移完成！旧文件可在确认无误后安全删除")
	}
}

func parseFlags() migrateConfig {
	cfg := migrateConfig{}
	flag.BoolVar(&cfg.dryRun, "dry-run", false, "只打印不执行")
	flag.StringVar(&cfg.docRoot, "doc-root", "/opt/xiezuojun/documents", "新文档根目录")
	flag.StringVar(&cfg.memRoot, "mem-root", "/opt/xiezuojun/memory", "新记忆根目录")
	flag.StringVar(&cfg.dataRoot, "data-root", "./data", "旧数据根目录")
	flag.StringVar(&cfg.dbDSN, "db-dsn", "", "MySQL DSN")
	flag.Parse()
	if cfg.dbDSN == "" {
		cfg.dbDSN = os.Getenv("DB_DSN")
	}
	return cfg
}
