package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// TrashMove 移入回收站：将文件移到项目 .trash/ 目录
func TrashMove(docPath string, meta interface{}) error {
	trashDir := filepath.Join(filepath.Dir(docPath), ".trash")
	if err := os.MkdirAll(trashDir, 0755); err != nil {
		return fmt.Errorf("mkdir trash: %w", err)
	}
	// 移动正文文件
	destPath := filepath.Join(trashDir, filepath.Base(docPath))
	if err := os.Rename(docPath, destPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("move to trash: %w", err)
	}
	// 保存元信息到 .trash/{base}.meta.json
	if meta != nil {
		metaName := filepath.Base(docPath) + ".meta.json"
		metaPath := filepath.Join(trashDir, metaName)
		return AtomicWriteJSON(metaPath, meta)
	}
	return nil
}

// TrashRestore 从回收站恢复文件
func TrashRestore(trashPath string, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("mkdir dest: %w", err)
	}
	if err := os.Rename(trashPath, destPath); err != nil {
		return fmt.Errorf("restore from trash: %w", err)
	}
	// 删除对应的 meta.json
	os.Remove(trashPath + ".meta.json")
	return nil
}

// TrashCleanup 清理超过7天的回收站文件
func TrashCleanup(paths *PathConfig, userID string) (int, error) {
	userDir := paths.UserDir(userID)
	entries, err := os.ReadDir(userDir)
	if err != nil {
		return 0, err
	}
	cleaned := 0
	cutoff := time.Now().AddDate(0, 0, -7)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		trashDir := filepath.Join(userDir, entry.Name(), ".trash")
		n, err := cleanupDir(trashDir, cutoff)
		cleaned += n
		if err != nil {
			continue // 跳过错误继续清理
		}
	}
	return cleaned, nil
}

// cleanupDir 清理单个 .trash 目录中超过 cutoff 的文件
func cleanupDir(dir string, cutoff time.Time) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, nil // 不存在则跳过
	}
	cleaned := 0
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		os.Remove(filepath.Join(dir, e.Name()))
		cleaned++
	}
	return cleaned, nil
}

// TrashList 列出回收站内容
func TrashList(paths *PathConfig, userID string) ([]map[string]interface{}, error) {
	userDir := paths.UserDir(userID)
	var items []map[string]interface{}
	entries, err := os.ReadDir(userDir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		trashDir := filepath.Join(userDir, entry.Name(), ".trash")
		items = append(items, listTrashDir(trashDir, entry.Name())...)
	}
	return items, nil
}

// listTrashDir 列出单个项目的回收站
func listTrashDir(dir string, project string) []map[string]interface{} {
	var items []map[string]interface{}
	files, _ := os.ReadDir(dir)
	for _, f := range files {
		if filepath.Ext(f.Name()) != ".meta.json" {
			continue
		}
		metaPath := filepath.Join(dir, f.Name())
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		info, _ := os.Stat(filepath.Join(dir, f.Name()))
		var meta map[string]interface{}
		_ = json.Unmarshal(data, &meta)
		meta["trash_path"] = metaPath
		meta["project"] = project
		meta["trashed_at"] = info.ModTime().Format(time.RFC3339)
		items = append(items, meta)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i]["trashed_at"].(string) > items[j]["trashed_at"].(string)
	})
	return items
}
