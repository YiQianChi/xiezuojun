package store

import (
	"encoding/json"
	"fmt"
	"time"
)

// ChangeEntry 变更日志条目
type ChangeEntry struct {
	Time    string `json:"time"`
	Action  string `json:"action"` // create_doc, update_doc, delete_doc, create_project, restore_doc
	Project string `json:"project"`
	Doc     string `json:"doc"`
	Summary string `json:"summary"`
}

// AppendChangelog 追加变更记录到用户 changelog
func AppendChangelog(paths *PathConfig, userID string, entry ChangeEntry) error {
	if entry.Time == "" {
		entry.Time = time.Now().Format(time.RFC3339)
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal changelog entry: %w", err)
	}
	// 每行一条 JSON，末尾加换行
	line = append(line, '\n')
	return AtomicAppend(paths.ChangelogPath(userID), line)
}
