// chapters.go — 章节摘要存储

package memory

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"xiezuojun-server/internal/store"
)

// ReadChapterSummary 读章节摘要
func (s *MemoryStorage) ReadChapterSummary(uid, pid, docID string) (string, error) {
	if err := safePath(docID); err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(s.chapterDir(uid, pid), docID+".summary.md"))
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
	dir := s.chapterDir(uid, pid)
	if err := ensureDir(dir); err != nil {
		return err
	}
	return store.AtomicWrite(filepath.Join(dir, docID+".summary.md"), []byte(content), 0644)
}

// ListChapterSummaries 列章节摘要
func (s *MemoryStorage) ListChapterSummaries(uid, pid string) ([]string, error) {
	dir := s.chapterDir(uid, pid)
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
