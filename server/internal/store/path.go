package store

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

// PathConfig 存储根目录配置
type PathConfig struct {
	DocRoot string // /opt/xiezuojun/documents
	MemRoot string // /opt/xiezuojun/memory
}

// SanitizeName 书名/文件名安全化
// 过滤危险字符，截断50字符，空名默认 untitled-{id}
func SanitizeName(name string, fallbackID uint) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Sprintf("untitled-%d", fallbackID)
	}
	// 过滤非法字符
	var b strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == ' ' {
			b.WriteRune(r)
		}
	}
	name = strings.TrimSpace(b.String())
	if name == "" {
		return fmt.Sprintf("untitled-%d", fallbackID)
	}
	// 截断到50字符（不在单词中间截断的简单版本）
	if len([]rune(name)) > 50 {
		name = string([]rune(name)[:50])
	}
	return name
}

// UserDir 返回用户根目录
func (p *PathConfig) UserDir(userID string) string {
	return filepath.Join(p.DocRoot, userID)
}

// ProjectDir 返回项目目录
func (p *PathConfig) ProjectDir(userID string, projectID uint, projectName string) string {
	return filepath.Join(p.UserDir(userID), SanitizeName(projectName, projectID))
}

// FolderPath 返回文件夹目录路径（项目根下的子目录）
func (p *PathConfig) FolderPath(userID string, projectID uint, projectName string, folderID uint) string {
	return filepath.Join(p.ProjectDir(userID, projectID, projectName), fmt.Sprintf("folder-%d", folderID))
}

// DocPath 返回文档正文路径（支持文件夹层级）
// 如果 parentFolderID > 0，文件放在文件夹目录下；否则放项目根目录
func (p *PathConfig) DocPath(userID string, projectID uint, projectName string, docID uint, parentFolderID ...uint) string {
	base := p.ProjectDir(userID, projectID, projectName)
	if len(parentFolderID) > 0 && parentFolderID[0] > 0 {
		base = filepath.Join(base, fmt.Sprintf("folder-%d", parentFolderID[0]))
	}
	return filepath.Join(base, fmt.Sprintf("%d.md", docID))
}

// MemoryDir 返回记忆目录
func (p *PathConfig) MemoryDir(userID string, projectID uint) string {
	return filepath.Join(p.MemRoot, userID, fmt.Sprintf("%d", projectID))
}

// QuotaPath 返回配额文件路径
func (p *PathConfig) QuotaPath(userID string) string {
	return filepath.Join(p.UserDir(userID), ".quota.json")
}

// ChangelogPath 返回变更日志路径
func (p *PathConfig) ChangelogPath(userID string) string {
	return filepath.Join(p.UserDir(userID), ".changelog")
}

// TrashDir 返回项目回收站目录
func (p *PathConfig) TrashDir(userID string, projectID uint, projectName string) string {
	return filepath.Join(p.ProjectDir(userID, projectID, projectName), ".trash")
}

// StructurePath 返回项目文档树路径
func (p *PathConfig) StructurePath(userID string, projectID uint, projectName string) string {
	return filepath.Join(p.ProjectDir(userID, projectID, projectName), ".structure.json")
}

// MetaPath 返回项目元信息路径
func (p *PathConfig) MetaPath(userID string, projectID uint, projectName string) string {
	return filepath.Join(p.ProjectDir(userID, projectID, projectName), ".meta.json")
}
