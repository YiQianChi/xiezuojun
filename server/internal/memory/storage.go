// MemoryStorage — 项目级辅助记忆文件读写（角色卡、关系图等）
// 存储在项目文档目录下，与章节文件同级

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
	docRoot string
}

// NewMemoryStorage 创建记忆存储，docRoot 为文档根目录（如 /opt/xiezuojun/documents）
func NewMemoryStorage(docRoot string) *MemoryStorage {
	if docRoot == "" {
		docRoot = "./documents"
	}
	return &MemoryStorage{docRoot: docRoot}
}

// MemRoot 返回文档根目录（兼容旧代码）
func (s *MemoryStorage) MemRoot() string {
	return s.docRoot
}

// memoryPath 返回项目辅助记忆目录（.memory/ 子目录）
// ponytail: 隐藏目录，与 .structure.json 同级，不污染文档列表
func (s *MemoryStorage) memoryPath(uid, pid string) string {
	return filepath.Join(s.docRoot, uid, fmt.Sprintf("p-%010s", pid), ".memory")
}

// charDir 返回角色卡目录
func (s *MemoryStorage) charDir(uid, pid string) string {
	return filepath.Join(s.memoryPath(uid, pid), "CHARACTERS")
}

// chapterDir 返回章节摘要目录
func (s *MemoryStorage) chapterDir(uid, pid string) string {
	return filepath.Join(s.memoryPath(uid, pid), "chapters")
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
