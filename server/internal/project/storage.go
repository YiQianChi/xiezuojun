package project

import (
	"fmt"
	"os"
	"path/filepath"
)

func (h *Handler) docPath(docID uint) string {
	return filepath.Join(h.dataRoot, "documents", fmt.Sprintf("%d.txt", docID))
}

func (h *Handler) saveDocContent(docID uint, content string) error {
	path := h.docPath(docID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// LoadDocContent 导出给 gateway 使用
func (h *Handler) LoadDocContent(docID uint) (string, error) {
	data, err := os.ReadFile(h.docPath(docID))
	if err != nil {
		return "", err
	}
	return string(data), nil
}
