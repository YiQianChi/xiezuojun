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

// docPath 返回文档正文路径（根据 parentID 决定是否在文件夹下）
func (ds *DocStore) docPath(userID string, projectID uint, projectName string, docID uint, parentID ...uint) string {
	if len(parentID) > 0 && parentID[0] > 0 {
		return ds.paths.DocPath(userID, projectID, projectName, docID, parentID[0])
	}
	return ds.paths.DocPath(userID, projectID, projectName, docID)
}

// saveDocContent 原子写入文档正文（支持文件夹层级）
func (ds *DocStore) saveDocContent(userID string, projectID uint, projectName string, docID uint, content string, parentID ...uint) error {
	path := ds.docPath(userID, projectID, projectName, docID, parentID...)
	return store.AtomicWrite(path, []byte(content), 0644)
}

// loadDocContent 读取文档正文（支持文件夹层级）
func (ds *DocStore) loadDocContent(userID string, projectID uint, projectName string, docID uint, parentID ...uint) (string, error) {
	data, err := os.ReadFile(ds.docPath(userID, projectID, projectName, docID, parentID...))
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

// ensureFolderDir 确保文件夹目录存在
func (ds *DocStore) ensureFolderDir(userID string, projectID uint, projectName string, folderID uint) error {
	dir := ds.paths.FolderPath(userID, projectID, projectName, folderID)
	return os.MkdirAll(dir, 0755)
}

// moveDocFile 移动文档文件到新位置（创建文件夹目录 + mv 文件）
func (ds *DocStore) moveDocFile(userID string, projectID uint, projectName string, docID uint, oldParentID *uint, newParentID *uint) error {
	var oldPath string
	if oldParentID != nil && *oldParentID > 0 {
		oldPath = ds.paths.DocPath(userID, projectID, projectName, docID, *oldParentID)
	} else {
		oldPath = ds.paths.DocPath(userID, projectID, projectName, docID)
	}

	var newPath string
	if newParentID != nil && *newParentID > 0 {
		newPath = ds.paths.DocPath(userID, projectID, projectName, docID, *newParentID)
	} else {
		newPath = ds.paths.DocPath(userID, projectID, projectName, docID)
	}

	if oldPath == newPath {
		return nil
	}

	if newParentID != nil && *newParentID > 0 {
		if err := os.MkdirAll(ds.paths.FolderPath(userID, projectID, projectName, *newParentID), 0755); err != nil {
			return fmt.Errorf("mkdir folder: %w", err)
		}
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("mv file: %w", err)
	}
	return nil
}

// removeFolderDir 删除空文件夹目录
func (ds *DocStore) removeFolderDir(userID string, projectID uint, projectName string, folderID uint) error {
	dir := ds.paths.FolderPath(userID, projectID, projectName, folderID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.Name() != ".placeholder" {
			return nil
		}
	}
	return os.Remove(dir)
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

// trashMove 移入回收站（支持文件夹层级）
func (ds *DocStore) trashMove(userID string, projectID uint, projectName string, docID uint, parentID *uint, meta interface{}) error {
	var path string
	if parentID != nil && *parentID > 0 {
		path = ds.docPath(userID, projectID, projectName, docID, *parentID)
	} else {
		path = ds.docPath(userID, projectID, projectName, docID)
	}
	return store.TrashMove(path, meta)
}

// countWords 统计字数
func countWords(s string) int {
	return len([]rune(s))
}

// uintToStr uint 转 string
func uintToStr(u uint) string {
	return strconv.FormatUint(uint64(u), 10)
}
