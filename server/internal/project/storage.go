package project

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"xiezuojun-server/internal/store"
)

// DocStore 文件存储层
type DocStore struct {
	paths *store.PathConfig
	locks *store.UserLocks
}

// NewDocStore 创建文件存储层
func NewDocStore(paths *store.PathConfig, locks *store.UserLocks) *DocStore {
	return &DocStore{paths: paths, locks: locks}
}

// docPath 返回文档正文路径
// docPath returns the document file path
func (ds *DocStore) docPath(userID string, projectID uint, projectName string, docID uint) string {
	return ds.paths.DocPath(userID, projectID, projectName, docID)
}

// saveDocContent 原子写入文档正文
// saveDocContent writes document content atomically
func (ds *DocStore) saveDocContent(userID string, projectID uint, projectName string, docID uint, content string) error {
	path := ds.docPath(userID, projectID, projectName, docID)
	return store.AtomicWrite(path, []byte(content), 0644)
}

// loadDocContent 读取文档正文
// loadDocContent reads document content
func (ds *DocStore) loadDocContent(userID string, projectID uint, projectName string, docID uint) (string, error) {
	data, err := os.ReadFile(ds.docPath(userID, projectID, projectName, docID))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// LoadDocContent 导出接口（兼容 gateway 调用，从旧路径读取）
func (ds *DocStore) LoadDocContent(docID uint) (string, error) {
	path := fmt.Sprintf("data/documents/%d.txt", docID)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// loadDocContentFallback 从旧路径读取（兼容期）
func (ds *DocStore) loadDocContentFallback(docID uint) (string, error) {
	path := fmt.Sprintf("data/documents/%d.txt", docID)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ensureProjectDir 创建项目目录和基础文件
// ensureProjectDir creates project directory and base files
func (ds *DocStore) ensureProjectDir(userID string, projectID uint, projectName string) error {
	dir := ds.paths.ProjectDir(userID, projectID, projectName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir project: %w", err)
	}
	metaPath := ds.paths.MetaPath(userID, projectID, projectName)
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		meta := map[string]interface{}{
			"id": projectID, "name": projectName, "structure": "tree",
		}
		if err := store.AtomicWriteJSON(metaPath, meta); err != nil {
			return fmt.Errorf("write meta: %w", err)
		}
	}
	return nil
}

// saveStructureJSON 更新项目文档树
// saveStructureJSON updates project document tree
func (ds *DocStore) saveStructureJSON(userID string, projectID uint, projectName string, structure interface{}) error {
	path := ds.paths.StructurePath(userID, projectID, projectName)
	return store.AtomicWriteJSON(path, structure)
}

// loadStructureJSON 读取项目文档树
// loadStructureJSON reads project document tree
func (ds *DocStore) loadStructureJSON(userID string, projectID uint, projectName string) ([]map[string]interface{}, error) {
	data, err := os.ReadFile(ds.paths.StructurePath(userID, projectID, projectName))
	if err != nil {
		return nil, err
	}
	var structure []map[string]interface{}
	if err := json.Unmarshal(data, &structure); err != nil {
		return nil, err
	}
	return structure, nil
}

// trashMove 移入回收站
// trashMove moves file to trash
func (ds *DocStore) trashMove(userID string, projectID uint, projectName string, docID uint, meta interface{}) error {
	docPath := ds.docPath(userID, projectID, projectName, docID)
	return store.TrashMove(docPath, meta)
}

// countWords 统计字数
func countWords(s string) int {
	return len([]rune(s))
}

// uintToStr uint 转 string
func uintToStr(u uint) string {
	return strconv.FormatUint(uint64(u), 10)
}
