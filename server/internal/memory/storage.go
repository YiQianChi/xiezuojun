// MemoryStorage — 文件即记忆：Markdown 记忆文件读写
// 新目录结构：memory/{uid}/{pid}/

package memory

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"xiezuojun-server/internal/store"
)

// MemoryStorage 记忆文件存储
type MemoryStorage struct {
	memRoot string
}

// NewMemoryStorage 创建记忆存储，memRoot 为记忆根目录（如 /opt/xiezuojun/memory）
func NewMemoryStorage(memRoot string) *MemoryStorage {
	if memRoot == "" {
		memRoot = "./memory"
	}
	return &MemoryStorage{memRoot: memRoot}
}

// MemRoot 返回记忆根目录（供外部查询）
func (s *MemoryStorage) MemRoot() string {
	return s.memRoot
}

// memoryPath 返回用户项目的记忆目录
func (s *MemoryStorage) memoryPath(uid, pid string) string {
	return filepath.Join(s.memRoot, uid, pid)
}

// charDir 返回角色卡目录
func (s *MemoryStorage) charDir(uid, pid string) string {
	return filepath.Join(s.memRoot, uid, pid, "CHARACTERS")
}

// chapterDir 返回章节摘要目录
func (s *MemoryStorage) chapterDir(uid, pid string) string {
	return filepath.Join(s.memRoot, uid, pid, "chapters")
}

// safePath 检查路径安全性（防目录遍历）
func safePath(p string) error {
	cleaned := filepath.Clean(p)
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("path traversal detected: %s", p)
	}
	return nil
}

// ensureDir 确保目录存在
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// ========== 通用记忆文件 ==========

// ListMemoryFiles 列出项目所有记忆文件
func (s *MemoryStorage) ListMemoryFiles(uid, pid string) ([]string, error) {
	base := s.memoryPath(uid, pid)
	if err := ensureDir(base); err != nil {
		return nil, fmt.Errorf("ensure dir: %w", err)
	}
	var files []string
	err := filepath.Walk(base, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(base, path)
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// ReadMemoryFile 读取单个记忆文件
func (s *MemoryStorage) ReadMemoryFile(uid, pid, path string) (string, error) {
	if err := safePath(path); err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(s.memoryPath(uid, pid), path))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteMemoryFile 写入记忆文件
func (s *MemoryStorage) WriteMemoryFile(uid, pid, path, content string) error {
	if err := safePath(path); err != nil {
		return err
	}
	full := filepath.Join(s.memoryPath(uid, pid), path)
	return store.AtomicWrite(full, []byte(content), 0644)
}

// DeleteMemoryFile 删除记忆文件
func (s *MemoryStorage) DeleteMemoryFile(uid, pid, path string) error {
	if err := safePath(path); err != nil {
		return err
	}
	return os.Remove(filepath.Join(s.memoryPath(uid, pid), path))
}
