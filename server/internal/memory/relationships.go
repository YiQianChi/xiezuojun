package memory

import (
	"encoding/json"
	"os"
	"path/filepath"

	"xiezuojun-server/internal/store"
)

type RelationshipNode struct {
	Name  string `json:"name"`
	Role  string `json:"role"`
	Color string `json:"color"`
	Shape string `json:"shape"`
}

type RelationshipLink struct {
	Source string `json:"source"` // node name
	Target string `json:"target"` // node name
	Type   string `json:"type"`
	Label  string `json:"label"`
	FromCh int    `json:"from_ch"` // 关系起始章节序号 (1-based, 0=从一开始)
	ToCh   int    `json:"to_ch"`   // 关系结束章节序号 (0=至今)
}

type RelType struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Color string `json:"color"`
	Dash  []int  `json:"dash"`
}

type RelationshipData struct {
	Nodes []RelationshipNode `json:"nodes"`
	Links []RelationshipLink `json:"links"`
	Types []RelType          `json:"types"`
}

func (s *MemoryStorage) relationshipsPath(uid, pid string) string {
	return filepath.Join(s.memoryPath(uid, pid), "RELATIONSHIPS.json")
}

func (s *MemoryStorage) ReadRelationships(uid, pid string) (*RelationshipData, error) {
	data, err := os.ReadFile(s.relationshipsPath(uid, pid))
	if err != nil {
		if os.IsNotExist(err) {
			return &RelationshipData{Nodes: []RelationshipNode{}, Links: []RelationshipLink{}, Types: []RelType{}}, nil
		}
		return nil, err
	}
	var rd RelationshipData
	if err := json.Unmarshal(data, &rd); err != nil {
		return nil, err
	}
	if rd.Nodes == nil {
		rd.Nodes = []RelationshipNode{}
	}
	if rd.Links == nil {
		rd.Links = []RelationshipLink{}
	}
	if rd.Types == nil {
		rd.Types = []RelType{}
	}
	return &rd, nil
}

func (s *MemoryStorage) WriteRelationships(uid, pid string, data *RelationshipData) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return store.AtomicWrite(s.relationshipsPath(uid, pid), b, 0644)
}
