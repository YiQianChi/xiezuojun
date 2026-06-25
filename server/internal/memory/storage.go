// MemoryStorage — 文件即记忆：Markdown 记忆文件读写
// 文件结构：DATA_ROOT/users/{uid}/projects/{pid}/memory/

package memory

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MemoryStorage 记忆文件存储
type MemoryStorage struct {
	dataRoot string
}

func NewMemoryStorage(dataRoot string) *MemoryStorage {
	if dataRoot == "" {
		dataRoot = "./data"
	}
	return &MemoryStorage{dataRoot: dataRoot}
}

// memoryPath 返回用户项目的记忆目录
func (s *MemoryStorage) memoryPath(uid, pid string) string {
	return filepath.Join(s.dataRoot, "users", uid, "projects", pid, "memory")
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

// ListMemoryFiles 列出项目所有记忆文件
func (s *MemoryStorage) ListMemoryFiles(uid, pid string) ([]string, error) {
	base := s.memoryPath(uid, pid)
	if err := os.MkdirAll(base, 0755); err != nil {
		return nil, fmt.Errorf("ensure dir: %w", err)
	}

	var files []string
	err := filepath.Walk(base, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
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
	full := filepath.Join(s.memoryPath(uid, pid), path)
	data, err := os.ReadFile(full)
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
	if err := ensureDir(filepath.Dir(full)); err != nil {
		return err
	}
	return os.WriteFile(full, []byte(content), 0644)
}

// DeleteMemoryFile 删除记忆文件
func (s *MemoryStorage) DeleteMemoryFile(uid, pid, path string) error {
	if err := safePath(path); err != nil {
		return err
	}
	full := filepath.Join(s.memoryPath(uid, pid), path)
	return os.Remove(full)
}

// ========== 角色卡 ==========

func charDir(uid, pid string) string {
	return filepath.Join("./data", "users", uid, "projects", pid, "memory", "CHARACTERS")
}

// ListCharacters 列出所有角色
func (s *MemoryStorage) ListCharacters(uid, pid string) ([]string, error) {
	dir := charDir(uid, pid)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// ReadCharacter 读取角色卡
func (s *MemoryStorage) ReadCharacter(uid, pid, name string) (string, error) {
	if err := safePath(name); err != nil {
		return "", err
	}
	path := filepath.Join(charDir(uid, pid), name+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteCharacter 写角色卡
func (s *MemoryStorage) WriteCharacter(uid, pid, name, content string) error {
	if err := safePath(name); err != nil {
		return err
	}
	dir := charDir(uid, pid)
	if err := ensureDir(dir); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name+".md"), []byte(content), 0644)
}

// DeleteCharacter 删角色
func (s *MemoryStorage) DeleteCharacter(uid, pid, name string) error {
	if err := safePath(name); err != nil {
		return err
	}
	return os.Remove(filepath.Join(charDir(uid, pid), name+".md"))
}

// ========== 章节摘要 ==========

func chapterDir(uid, pid string) string {
	return filepath.Join("./data", "users", uid, "projects", pid, "memory", "chapters")
}

// ReadChapterSummary 读章节摘要
func (s *MemoryStorage) ReadChapterSummary(uid, pid, docID string) (string, error) {
	if err := safePath(docID); err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(chapterDir(uid, pid), docID+".summary.md"))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteChapterSummary 写章节摘要
func (s *MemoryStorage) WriteChapterSummary(uid, pid, docID, content string) error {
	if err := safePath(docID); err != nil {
		return err
	}
	dir := chapterDir(uid, pid)
	if err := ensureDir(dir); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, docID+".summary.md"), []byte(content), 0644)
}

// ListChapterSummaries 列章节摘要
func (s *MemoryStorage) ListChapterSummaries(uid, pid string) ([]string, error) {
	dir := chapterDir(uid, pid)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ids = append(ids, strings.TrimSuffix(e.Name(), ".summary.md"))
	}
	sort.Strings(ids)
	return ids, nil
}
