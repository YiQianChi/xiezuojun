// Package store — 文件操作基础层
// 所有上层存储都依赖此包进行文件读写

package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// atomicWriteFile 原子写入：写临时文件 → rename
// TODO: 等 store 包完成后，替换为完整的 AtomicWrite（含 fsync）
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	// TODO: f.Sync()
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) // 清理临时文件
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// AtomicWrite 原子写入文件
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	return atomicWriteFile(path, data, perm)
}

// AtomicAppend 原子追加内容到文件
func AtomicAppend(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return atomicWriteFile(path, data, 0644)
	}
	f.Close()
	existing, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read for append: %w", err)
	}
	return atomicWriteFile(path, append(existing, data...), 0644)
}

// AtomicWriteJSON 原子写入 JSON 文件
func AtomicWriteJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	return atomicWriteFile(path, append(data, '\n'), 0644)
}

// SafeName 书名/文件名安全化
// 过滤危险字符，截断50字符，空名返回 fallback
func SafeName(name string, fallback string) string {
	if name == "" {
		return fallback
	}
	// 过滤危险字符
	dangerous := "/\\<>|*?\"'\x00"
	var safe []rune
	for _, r := range name {
		if containsRune(dangerous, r) {
			continue
		}
		safe = append(safe, r)
	}
	result := string(safe)
	result = trimDot(result)
	if result == "" || result == "." || result == ".." {
		return fallback
	}
	if len(result) > 50 {
		result = result[:50]
	}
	return result
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

// trimDot 移除首尾的点号
func trimDot(s string) string {
	for len(s) > 0 && s[0] == '.' {
		s = s[1:]
	}
	for len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	return s
}
