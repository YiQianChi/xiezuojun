// Package store — 文件操作基础层
// 所有上层存储都依赖此包进行文件读写

package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// atomicWriteFile 原子写入：写临时文件 → fsync → rename
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

// AtomicAppend 追加内容到文件
// ponytail: O_APPEND — POSIX atomic per write up to PIPE_BUF (4096 on Linux).
// Ceiling: caller must serialize when single write exceeds PIPE_BUF; changelog
// single-line JSON may exceed on long summaries, fine here because
// AppendChangelog is called serially from request handlers. Add mutex if concurrency appears.
func AtomicAppend(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open append: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write append: %w", err)
	}
	return nil
}

// AtomicWriteJSON 原子写入 JSON 文件
func AtomicWriteJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	return atomicWriteFile(path, append(data, '\n'), 0644)
}
