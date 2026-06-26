// characters.go — 角色卡存储

package memory

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"xiezuojun-server/internal/store"
)

// ListCharacters 列出所有角色
func (s *MemoryStorage) ListCharacters(uid, pid string) ([]string, error) {
	dir := s.charDir(uid, pid)
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
		names = append(names, strings.TrimSuffix(e.Name(), ".md"))
	}
	sort.Strings(names)
	return names, nil
}

// ReadCharacter 读取角色卡
func (s *MemoryStorage) ReadCharacter(uid, pid, name string) (string, error) {
	if err := safePath(name); err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(s.charDir(uid, pid), name+".md"))
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
	dir := s.charDir(uid, pid)
	if err := ensureDir(dir); err != nil {
		return err
	}
	return store.AtomicWrite(filepath.Join(dir, name+".md"), []byte(content), 0644)
}

// DeleteCharacter 删角色
func (s *MemoryStorage) DeleteCharacter(uid, pid, name string) error {
	if err := safePath(name); err != nil {
		return err
	}
	return os.Remove(filepath.Join(s.charDir(uid, pid), name+".md"))
}
