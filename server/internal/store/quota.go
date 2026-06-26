package store

import (
	"encoding/json"
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

// LoadQuota 读取用户配额文件
func LoadQuota(paths *PathConfig, userID string) (*Quota, error) {
	data, err := os.ReadFile(paths.QuotaPath(userID))
	if err != nil {
		if os.IsNotExist(err) {
			return &Quota{MaxWords: 1000000}, nil
		}
		return nil, fmt.Errorf("read quota: %w", err)
	}
	var q Quota
	if err := json.Unmarshal(data, &q); err != nil {
		return nil, fmt.Errorf("unmarshal quota: %w", err)
	}
	if q.MaxWords == 0 {
		q.MaxWords = 1000000
	}
	return &q, nil
}

// CheckQuota 检查是否超限
func (q *Quota) CheckQuota(addWords int) error {
	if q.MaxWords <= 0 {
		return nil // VIP 不限
	}
	if q.TotalWords+addWords > q.MaxWords {
		return fmt.Errorf("配额不足：已用 %d 字，上限 %d 字", q.TotalWords, q.MaxWords)
	}
	return nil
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
