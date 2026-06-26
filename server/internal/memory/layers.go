// layers.go — 三层记忆（短期/长期/风格）

package memory

import (
	"fmt"
)

// ReadWorkingMemory 读取短期记忆
func (s *MemoryStorage) ReadWorkingMemory(uid, pid string) (string, error) {
	return s.ReadMemoryFile(uid, pid, "working_memory.md")
}

// WriteWorkingMemory 写入短期记忆
func (s *MemoryStorage) WriteWorkingMemory(uid, pid, content string) error {
	return s.WriteMemoryFile(uid, pid, "working_memory.md", content)
}

// ReadLongTermMemory 读取长期记忆
func (s *MemoryStorage) ReadLongTermMemory(uid, pid string) (string, error) {
	return s.ReadMemoryFile(uid, pid, "long_term.md")
}

// WriteLongTermMemory 写入长期记忆
func (s *MemoryStorage) WriteLongTermMemory(uid, pid, content string) error {
	return s.WriteMemoryFile(uid, pid, "long_term.md", content)
}

// ReadStyleMemory 读取风格记忆
func (s *MemoryStorage) ReadStyleMemory(uid, pid string) (string, error) {
	return s.ReadMemoryFile(uid, pid, "STYLE.md")
}

// WriteStyleMemory 写入风格记忆
func (s *MemoryStorage) WriteStyleMemory(uid, pid, content string) error {
	return s.WriteMemoryFile(uid, pid, "STYLE.md", content)
}

// DistillMemory 蒸馏短期记忆 → 长期记忆
// 合并短期到长期，然后清空短期
func (s *MemoryStorage) DistillMemory(uid, pid string) error {
	working, err := s.ReadWorkingMemory(uid, pid)
	if err != nil || working == "" {
		return fmt.Errorf("no working memory to distill")
	}
	longTerm, _ := s.ReadLongTermMemory(uid, pid)
	distilled := longTerm + "\n\n---\n\n### 蒸馏记录\n" + working

	if err := s.WriteLongTermMemory(uid, pid, distilled); err != nil {
		return err
	}
	return s.WriteWorkingMemory(uid, pid, "")
}
