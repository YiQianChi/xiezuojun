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
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("open tmp: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := f.Sync(); err != nil { // fsync before rename for crash safety
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("sync tmp: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("close tmp: %w", err)
	}
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
