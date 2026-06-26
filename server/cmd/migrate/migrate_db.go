// migrate_db.go — 数据库连接与统计

package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// connectDB 连接 MySQL 数据库
func connectDB(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("请提供 --db-dsn 参数或设置 DB_DSN 环境变量")
	}

	log.Printf("🔗 连接数据库...")
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层 DB 失败: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库 ping 失败: %w", err)
	}
	log.Printf("✅ 数据库连接成功")
	return db, nil
}

// ========== 统计 ==========

type migrateStats struct {
	projects       int
	documents      int
	written        int
	skipped        int
	errors         int
	memoryMigrated int
}

func printStats(stats *migrateStats) {
	fmt.Println("\n========== 迁移统计 ==========")
	fmt.Printf("项目数:     %d\n", stats.projects)
	fmt.Printf("文档数:     %d\n", stats.documents)
	fmt.Printf("文件写入:   %d\n", stats.written)
	fmt.Printf("记忆迁移:   %d\n", stats.memoryMigrated)
	fmt.Printf("跳过:       %d\n", stats.skipped)
	fmt.Printf("错误:       %d\n", stats.errors)
	fmt.Println("==============================")
	if stats.errors > 0 {
		log.Printf("⚠️  有 %d 个错误，请检查日志", stats.errors)
	}
}

// ========== 工具函数 ==========

// toSafeName 将字符串转为安全的文件名
func toSafeName(s string) string {
	safe := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			safe = append(safe, r)
		case r == '-' || r == '_':
			safe = append(safe, r)
		default:
			safe = append(safe, '_')
		}
	}
	if len(safe) == 0 {
		return "unnamed"
	}
	return string(safe)
}
