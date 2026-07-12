package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// Quota 用户用量配额
type Quota struct {
	TotalWords int `json:"total_words"`
	TotalFiles int `json:"total_files"`
	MaxWords   int `json:"max_words"` // free=1000000, VIP=0(不限)
}

// Save 保存配额文件
func (q *Quota) Save(paths *PathConfig, userID string) error {
	return AtomicWriteJSON(paths.QuotaPath(userID), q)
}

// RecalcQuota 重新计算用户总字数和文件数
func RecalcQuota(paths *PathConfig, userID string) (*Quota, error) {
	userDir := paths.UserDir(userID)
	entries, err := os.ReadDir(userDir)
	if err != nil {
		return nil, fmt.Errorf("read user dir: %w", err)
	}
	q := &Quota{MaxWords: 1000000}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectDir := filepath.Join(userDir, entry.Name())
		q.walkProjectDir(projectDir)
	}
	return q, nil
}

// walkProjectDir 遍历项目目录统计字数和文件数
func (q *Quota) walkProjectDir(dir string) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, f := range files {
		if f.IsDir() {
			continue // 跳过 .trash 等子目录
		}
		if filepath.Ext(f.Name()) == ".md" {
			q.TotalFiles++
			data, err := os.ReadFile(filepath.Join(dir, f.Name()))
			if err == nil {
				q.TotalWords += len([]rune(string(data)))
			}
		}
	}
}
